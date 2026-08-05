// ClaudeProvider implements Provider for Claude Code accounts, authenticating
// via the OAuth access token in the account's credentials file and querying
// the undocumented Anthropic usage API for the 5-hour rolling utilization.
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

// Kind identifies this as the Claude provider so the registry can route
// Claude accounts here.
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
	ExtraUsage struct {
		IsEnabled    bool    `json:"is_enabled"`
		MonthlyLimit int64   `json:"monthly_limit"`
		UsedCredits  float64 `json:"used_credits"`
		Utilization  float64 `json:"utilization"`
		Currency     string  `json:"currency"`
	} `json:"extra_usage"`
}

// Fetch retrieves the 5-hour, 7-day, and (if enabled) extra-usage quota for a
// Claude Code account by reading its OAuth access token from the
// CREDENTIALS_PATH file and querying the undocumented Anthropic usage API.
// The utilization percentage returned by the API is mapped to Used out of a
// Limit of 100.
//
// Side effects: reads the local credentials file and makes an outbound HTTP
// GET request to api.anthropic.com.
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
		msg := fmt.Sprintf("API returned status %d", resp.StatusCode)
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			msg = "API returned 401 Unauthorized: check if your access token is valid"
		case http.StatusForbidden:
			msg = "API returned 403 Forbidden: you may not have permission to access this endpoint"
		case http.StatusTooManyRequests:
			msg = "API returned 429 Too Many Requests: you are being rate limited"
		}
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: msg,
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

	// Extra window
	if usage.ExtraUsage.IsEnabled {
		slog.Debug("claude: successfully fetched extra quota", "account", a.Name, "used", usage.ExtraUsage.UsedCredits, "limit", usage.ExtraUsage.MonthlyLimit)
		used := int64(usage.ExtraUsage.UsedCredits)
		extraLimit := usage.ExtraUsage.MonthlyLimit
		results = append(results, Quota{
			Account: a,
			Name:    "extra",
			Status:  StatusOK,
			Used:    &used,
			Limit:   &extraLimit,
		})
	}

	return results
}
