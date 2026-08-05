// CopilotProvider implements Provider for GitHub Copilot accounts,
// authenticating via a long-lived gho_ token and querying the undocumented
// copilot_internal/user GitHub endpoint for quota snapshots.
//
// The token is resolved from two sources, in order: the real Copilot CLI's
// own config file (CREDENTIALS_PATH, if set and it contains a copilotTokens
// entry), falling back to `gh auth token`. The file-based path was the
// original design, but in practice not every Copilot CLI install persists a
// plaintext token to config.json (e.g. when auth is backed by an OS
// keyring), so CREDENTIALS_PATH is optional and the gh CLI fallback covers
// that case as long as `gh auth login` has been run.
//
// Ref: docs/task-copilot-usage-limits.md
package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/hieropold/tkncap/internal/account"
)

func init() {
	Register(&CopilotProvider{})
}

// CopilotProvider is the quota provider for GitHub Copilot accounts.
type CopilotProvider struct{}

// Kind identifies this as the Copilot provider so the registry can route
// Copilot accounts here.
func (c *CopilotProvider) Kind() account.Provider {
	return account.ProviderCopilot
}

// copilotConfig models the subset of ~/.copilot/config.json this provider
// needs: the map of copilotTokens keyed by "https://github.com:<login>".
type copilotConfig struct {
	CopilotTokens map[string]string `json:"copilotTokens"`
}

// copilotQuotaSnapshot mirrors one entry under quota_snapshots in the paid
// plan response shape.
type copilotQuotaSnapshot struct {
	Entitlement int64 `json:"entitlement"`
	Remaining   int64 `json:"remaining"`
	Unlimited   bool  `json:"unlimited"`
}

// copilotUserResponse models the fields this provider reads from the
// GET copilot_internal/user response. Both the "paid" (quota_snapshots) and
// "free/limited" (monthly_quotas) response shapes are represented since
// GitHub returns different bodies depending on plan.
type copilotUserResponse struct {
	CopilotPlan     string `json:"copilot_plan"`
	AccessTypeSKU   string `json:"access_type_sku"`
	QuotaResetDateU string `json:"quota_reset_date_utc"`
	QuotaResetDate  string `json:"quota_reset_date"`
	QuotaSnapshots  struct {
		PremiumInteractions copilotQuotaSnapshot `json:"premium_interactions"`
		Completions         copilotQuotaSnapshot `json:"completions"`
		Chat                copilotQuotaSnapshot `json:"chat"`
	} `json:"quota_snapshots"`
	MonthlyQuotas        map[string]int64 `json:"monthly_quotas"`
	LimitedUserQuotas    map[string]int64 `json:"limited_user_quotas"`
	LimitedUserResetDate string           `json:"limited_user_reset_date"`
}

// stripJSONCComments removes lines whose trimmed content starts with "//" so
// the JSONC contents of ~/.copilot/config.json can be parsed with the
// standard encoding/json package.
func stripJSONCComments(data []byte) []byte {
	var out strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		out.WriteString(line)
		out.WriteString("\n")
	}
	return []byte(out.String())
}

// extractToken picks the gho_ token to use from the copilotTokens map. If
// more than one entry is present, keys are sorted alphabetically and the
// first one is used, since Go map iteration order isn't stable and this
// needs to be deterministic across runs.
func extractToken(tokens map[string]string) string {
	if len(tokens) == 0 {
		return ""
	}
	keys := make([]string, 0, len(tokens))
	for k := range tokens {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return tokens[keys[0]]
}

// lookPath and runGhAuthToken are indirection seams over exec.LookPath and
// shelling out to `gh auth token`, overridden in tests so the gh CLI
// fallback path can be exercised deterministically without depending on the
// host machine's actual gh installation/auth state.
var lookPath = exec.LookPath

var runGhAuthToken = func(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// tokenFromCredentialsFile reads credPath (the real Copilot CLI's
// ~/.copilot/config.json, JSONC), extracts the gho_ token from its
// copilotTokens map, and returns an empty token with a non-nil error when
// the file can't be read/parsed. An empty token with a nil error means the
// file parsed fine but simply has no copilotTokens entry (e.g. this Copilot
// CLI install's auth is keyring-backed rather than file-backed).
//
// Side effects: reads credPath from disk.
func tokenFromCredentialsFile(credPath string) (string, error) {
	fileData, err := os.ReadFile(credPath)
	if err != nil {
		return "", fmt.Errorf("failed to read credentials file: %w", err)
	}

	var cfg copilotConfig
	if err := json.Unmarshal(stripJSONCComments(fileData), &cfg); err != nil {
		return "", fmt.Errorf("failed to parse credentials JSON: %w", err)
	}

	return extractToken(cfg.CopilotTokens), nil
}

// resolveCopilotToken determines the gho_ token to use for account a,
// trying CREDENTIALS_PATH first (if set) and falling back to `gh auth
// token`. It returns the resolved token, or an empty token with a Status and
// human-readable Message describing why no token could be obtained. A
// gh-not-on-PATH condition with no CREDENTIALS_PATH configured either is
// treated as StatusMisconfigured (nothing usable was configured, checked
// without any I/O beyond a PATH lookup); every other failure to obtain a
// token is StatusError, since at least one source was actually attempted.
//
// Side effects: may read CREDENTIALS_PATH from disk and/or shell out to
// `gh auth token`.
func resolveCopilotToken(ctx context.Context, a account.Account, credPath string) (token string, status Status, message string) {
	if credPath != "" {
		slog.Debug("copilot: reading credentials file", "account", a.Name, "path", credPath)
		fileToken, err := tokenFromCredentialsFile(credPath)
		if err != nil {
			slog.Debug("copilot: credentials file unusable, will try gh CLI fallback", "account", a.Name, "error", err)
		} else if fileToken != "" {
			return fileToken, "", ""
		} else {
			slog.Debug("copilot: credentials file has no copilotTokens entry, will try gh CLI fallback", "account", a.Name)
		}
	}

	if _, err := lookPath("gh"); err != nil {
		slog.Debug("copilot: gh CLI not found on PATH", "account", a.Name)
		if credPath == "" {
			return "", StatusMisconfigured, "missing TKNCAP_COPILOT_<ACCOUNT>_CREDENTIALS_PATH and gh CLI not found on PATH; set CREDENTIALS_PATH or install/auth the gh CLI"
		}
		return "", StatusError, "credentials file had no usable token and gh CLI not found on PATH; re-authenticate via the copilot CLI or install/auth the gh CLI"
	}

	slog.Debug("copilot: falling back to gh auth token", "account", a.Name)
	ghToken, err := runGhAuthToken(ctx)
	if err != nil {
		slog.Debug("copilot: gh auth token failed", "account", a.Name, "error", err)
		return "", StatusError, "no Copilot token available: CREDENTIALS_PATH had none and `gh auth token` failed; run `gh auth login` or re-authenticate via the copilot CLI"
	}
	if ghToken == "" {
		return "", StatusError, "no Copilot token available: `gh auth token` returned an empty token; run `gh auth login`"
	}

	return ghToken, "", ""
}

// Fetch retrieves the premium, completions, and chat quota rows for a
// Copilot account by resolving a gho_ token (see resolveCopilotToken) and
// querying the undocumented copilot_internal/user GitHub endpoint, sending
// request headers that mimic the real Copilot CLI's identity.
//
// Side effects: may read the local credentials file, shell out to
// `gh auth token`, and makes an outbound HTTPS GET request to api.github.com.
func (c *CopilotProvider) Fetch(ctx context.Context, a account.Account) []Quota {
	slog.Debug("copilot: fetching quota", "account", a.Name)

	credPath := a.Fields["CREDENTIALS_PATH"]

	token, status, message := resolveCopilotToken(ctx, a, credPath)
	if token == "" {
		slog.Debug("copilot: could not resolve a token", "account", a.Name, "status", status)
		return []Quota{{
			Account: a,
			Status:  status,
			Message: message,
		}}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/copilot_internal/user", nil)
	if err != nil {
		slog.Debug("copilot: failed to create request", "account", a.Name, "error", err)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: fmt.Sprintf("failed to create HTTP request: %v", err),
		}}
	}

	// Headers deliberately mimic the real Copilot CLI's request identity
	// (legacy "token" auth scheme, hardcoded User-Agent/Copilot-Integration-Id,
	// no Editor-Version headers) since this is an undocumented internal
	// endpoint with no public API contract. See docs/task-copilot-usage-limits.md
	// for the security rationale.
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("User-Agent", "GitHubCopilotCLI/1.0.78")
	req.Header.Set("Copilot-Integration-Id", "copilot-cli")
	req.Header.Set("Accept", "application/json")

	slog.Debug("copilot: sending API request", "account", a.Name, "url", req.URL.String())
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Debug("copilot: API request failed", "account", a.Name, "error", err)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: fmt.Sprintf("API request failed: %v", err),
		}}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("copilot: API returned non-OK status", "account", a.Name, "status", resp.StatusCode)
		msg := fmt.Sprintf("API returned status %d", resp.StatusCode)
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			msg = fmt.Sprintf("API returned %d: token expired or revoked; re-authenticate via the copilot CLI", resp.StatusCode)
		case http.StatusTooManyRequests:
			msg = "API returned 429 Too Many Requests: you are being rate limited"
		}
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: msg,
		}}
	}

	var body copilotUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		slog.Debug("copilot: failed to decode API response", "account", a.Name, "error", err)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: fmt.Sprintf("failed to decode API response: %v", err),
		}}
	}

	plan := body.CopilotPlan
	if plan == "" {
		plan = body.AccessTypeSKU
	}

	results := buildCopilotQuotas(a, body, plan)
	slog.Debug("copilot: successfully fetched quota", "account", a.Name, "rows", len(results))
	return results
}

// buildCopilotQuotas maps a decoded copilot_internal/user response into the
// three named Quota rows (premium, completions, chat) tkncap reports for
// Copilot accounts. It prefers the paid-plan quota_snapshots shape when
// present, falling back to the free/limited monthly_quotas shape otherwise.
// The plan name, if known, is appended to the first row's Message since the
// shared Quota struct has no dedicated field for it.
func buildCopilotQuotas(a account.Account, body copilotUserResponse, plan string) []Quota {
	resetAt := parseCopilotResetDate(body.QuotaResetDateU, body.QuotaResetDate)

	hasSnapshots := body.QuotaSnapshots.PremiumInteractions.Entitlement != 0 ||
		body.QuotaSnapshots.Completions.Entitlement != 0 ||
		body.QuotaSnapshots.Chat.Entitlement != 0 ||
		body.QuotaSnapshots.PremiumInteractions.Unlimited ||
		body.QuotaSnapshots.Completions.Unlimited ||
		body.QuotaSnapshots.Chat.Unlimited

	var results []Quota
	if hasSnapshots {
		results = append(results, quotaFromSnapshot(a, "premium", body.QuotaSnapshots.PremiumInteractions, resetAt))
		results = append(results, quotaFromSnapshot(a, "completions", body.QuotaSnapshots.Completions, resetAt))
		results = append(results, quotaFromSnapshot(a, "chat", body.QuotaSnapshots.Chat, resetAt))
	} else {
		limitedResetAt := parseCopilotResetDate("", body.LimitedUserResetDate)
		for _, name := range []string{"completions", "chat"} {
			total, ok := body.MonthlyQuotas[name]
			if !ok {
				continue
			}
			used := int64(0)
			results = append(results, Quota{
				Account:  a,
				Name:     name,
				Status:   StatusOK,
				Used:     &used,
				Limit:    &total,
				ResetsAt: limitedResetAt,
			})
		}
	}

	if plan != "" && len(results) > 0 {
		if results[0].Message == "" {
			results[0].Message = plan
		} else {
			results[0].Message = results[0].Message + "; " + plan
		}
	}

	return results
}

// quotaFromSnapshot converts one quota_snapshots entry (paid-plan response
// shape) into a Quota row. Unlimited entries report Used = 0 and no Limit,
// with Message set to "unlimited", since a numeric limit doesn't apply.
func quotaFromSnapshot(a account.Account, name string, snap copilotQuotaSnapshot, resetAt *time.Time) Quota {
	if snap.Unlimited {
		used := int64(0)
		return Quota{
			Account:  a,
			Name:     name,
			Status:   StatusOK,
			Used:     &used,
			Limit:    nil,
			ResetsAt: resetAt,
			Message:  "unlimited",
		}
	}

	used := snap.Entitlement - snap.Remaining
	limit := snap.Entitlement
	return Quota{
		Account:  a,
		Name:     name,
		Status:   StatusOK,
		Used:     &used,
		Limit:    &limit,
		ResetsAt: resetAt,
	}
}

// parseCopilotResetDate parses the RFC3339 quota_reset_date_utc field,
// falling back to the date-only quota_reset_date (format "2006-01-02") when
// the UTC field is absent. Returns nil if neither field parses.
func parseCopilotResetDate(utcStr, dateStr string) *time.Time {
	if utcStr != "" {
		if t, err := time.Parse(time.RFC3339, utcStr); err == nil {
			return &t
		}
		slog.Debug("copilot: failed to parse quota_reset_date_utc", "value", utcStr)
	}
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			return &t
		}
		slog.Debug("copilot: failed to parse quota_reset_date", "value", dateStr)
	}
	return nil
}
