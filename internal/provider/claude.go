/**
 * package provider — ClaudeProvider
 *
 * <purpose-start>
 * Implementation of the Provider interface for Claude Code quota. Uses the
 * credentials file specified in CREDENTIALS_PATH to authenticate with the
 * Anthropic API. It queries the undocumented OAuth usage endpoint to fetch
 * the current 5-hour rolling utilization limit.
 * <purpose-end>
 *
 * <inputs-start>
 * - account.Account with Provider == ProviderClaude and a Fields map that
 *   should contain "CREDENTIALS_PATH".
 * <inputs-end>
 *
 * <outputs-start>
 * - Quota with Status=StatusMisconfigured if CREDENTIALS_PATH is missing.
 * - Quota with Status=StatusError if file read, JSON decode, or API call fails.
 * - Quota with Status=StatusOK containing Used/Limit (as percentage) on success.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Reads a local file specified by CREDENTIALS_PATH.
 * - Makes an outbound HTTP GET request to api.anthropic.com.
 * - Logs the fetch attempt and its outcome at debug level.
 * - Registers itself in the provider registry via init().
 * <side-effects-end>
 */
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/hieropold/tkncap/internal/account"
)

func init() {
	Register(&ClaudeProvider{})
}

// ClaudeProvider is the quota provider for Claude Code accounts.
type ClaudeProvider struct{}

/**
 * Kind
 *
 * <purpose-start>
 * Returns the provider kind so the registry can route Claude accounts here.
 * <purpose-end>
 *
 * <inputs-start>
 * - None.
 * <inputs-end>
 *
 * <outputs-start>
 * - account.ProviderClaude.
 * <outputs-end>
 *
 * <side-effects-start>
 * - None.
 * <side-effects-end>
 */
func (c *ClaudeProvider) Kind() account.Provider {
	return account.ProviderClaude
}

type claudeCredentials struct {
	ClaudeAiOauth struct {
		AccessToken string `json:"accessToken"`
	} `json:"claudeAiOauth"`
}

type anthropicUsageResponse struct {
	FiveHour struct {
		Utilization float64 `json:"utilization"`
		ResetsAt    string  `json:"resets_at"`
	} `json:"five_hour"`
	SevenDay struct {
		Utilization float64 `json:"utilization"`
		ResetsAt    string  `json:"resets_at"`
	} `json:"seven_day"`
}

/**
 * Fetch
 *
 * <purpose-start>
 * Fetches the 5-hour rolling usage quota for a Claude Code account. Validates
 * that CREDENTIALS_PATH is present, reads the credentials JSON file to extract
 * the OAuth access token, and queries the undocumented Anthropic usage API.
 * The utilization percentage is mapped to `Used` out of a `Limit` of 100.
 * <purpose-end>
 *
 * <inputs-start>
 * - ctx context.Context: request context.
 * - a account.Account: the Claude account whose quota is requested.
 * <inputs-end>
 *
 * <outputs-start>
 * - []Quota containing the fetch status, usage details, and any error message.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Performs local file I/O to read credentials.
 * - Performs an HTTP GET request to https://api.anthropic.com.
 * - Logs debugging information via slog.
 * <side-effects-end>
 */
func (c *ClaudeProvider) Fetch(ctx context.Context, a account.Account) []Quota {
	slog.Debug("claude: fetching quota", "account", a.Name)

	credPath := a.Fields["CREDENTIALS_PATH"]
	if credPath == "" {
		slog.Debug("claude: account missing CREDENTIALS_PATH field", "account", a.Name)
		return []Quota{{
			Account: a,
			Status:  StatusMisconfigured,
			Message: "missing required field TKNCAP_CLAUDE_<ACCOUNT>_CREDENTIALS_PATH",
		}}
	}

	// 1. Read credentials file
	slog.Debug("claude: reading credentials file", "account", a.Name, "path", credPath)
	fileData, err := os.ReadFile(credPath)
	if err != nil {
		slog.Debug("claude: failed to read credentials file", "account", a.Name, "error", err)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: fmt.Sprintf("failed to read credentials file: %v", err),
		}}
	}

	var creds claudeCredentials
	if err := json.Unmarshal(fileData, &creds); err != nil {
		slog.Debug("claude: failed to parse credentials JSON", "account", a.Name, "error", err)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: fmt.Sprintf("failed to parse credentials JSON: %v", err),
		}}
	}

	token := creds.ClaudeAiOauth.AccessToken
	if token == "" {
		slog.Debug("claude: access token not found in credentials", "account", a.Name)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: "access token not found in credentials file",
		}}
	}

	// 2. Fetch usage from Anthropic API
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		slog.Debug("claude: failed to create request", "account", a.Name, "error", err)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: fmt.Sprintf("failed to create HTTP request: %v", err),
		}}
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("User-Agent", "tkncap/1.0 (claude-code)")

	slog.Debug("claude: sending API request", "account", a.Name, "url", req.URL.String())
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Debug("claude: API request failed", "account", a.Name, "error", err)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: fmt.Sprintf("API request failed: %v", err),
		}}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("claude: API returned non-OK status", "account", a.Name, "status", resp.StatusCode)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: fmt.Sprintf("API returned status %d", resp.StatusCode),
		}}
	}

	var usage anthropicUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		slog.Debug("claude: failed to decode API response", "account", a.Name, "error", err)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: fmt.Sprintf("failed to decode API response: %v", err),
		}}
	}

	var limit int64 = 100
	var results []Quota

	// Helper to safely parse and append
	appendQuota := func(name string, util float64, resetsAtStr string) {
		used := int64(util)
		var resetsAtPtr *time.Time
		if resetsAtStr != "" {
			if parsedTime, err := time.Parse(time.RFC3339, resetsAtStr); err == nil {
				resetsAtPtr = &parsedTime
			} else {
				slog.Debug("claude: failed to parse resets_at timestamp", "account", a.Name, "name", name, "error", err)
			}
		}
		results = append(results, Quota{
			Account:  a,
			Name:     name,
			Status:   StatusOK,
			Used:     &used,
			Limit:    &limit,
			ResetsAt: resetsAtPtr,
		})
	}

	// 5-hour window
	slog.Debug("claude: successfully fetched 5-hour quota", "account", a.Name, "utilization", usage.FiveHour.Utilization, "resets_at", usage.FiveHour.ResetsAt)
	appendQuota("5-hour", usage.FiveHour.Utilization, usage.FiveHour.ResetsAt)

	// 7-day window
	slog.Debug("claude: successfully fetched 7-day quota", "account", a.Name, "utilization", usage.SevenDay.Utilization, "resets_at", usage.SevenDay.ResetsAt)
	appendQuota("7-day", usage.SevenDay.Utilization, usage.SevenDay.ResetsAt)

	return results
}
