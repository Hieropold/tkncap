/**
 * package provider — GeminiProvider
 *
 * <purpose-start>
 * Implementation of the Provider interface for Gemini API quota. Validates
 * that the account has an API_KEY, then makes a lightweight API call
 * (countTokens) to the Generative Language API to extract current rate limit
 * and usage metrics from the response headers.
 * <purpose-end>
 *
 * <inputs-start>
 * - account.Account with Provider == ProviderGemini and a Fields map that
 *   should contain "API_KEY".
 * <inputs-end>
 *
 * <outputs-start>
 * - Quota with Status=StatusMisconfigured if API_KEY is missing.
 * - Quota with Status=StatusError if the API call fails or no headers are found.
 * - Quota with Status=StatusOK containing Used/Limit for RPM, TPM, RPD based on headers.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Makes an outbound HTTP POST request to generativelanguage.googleapis.com.
 * - Logs the fetch attempt and its outcome at debug level.
 * - Registers itself in the provider registry via init().
 * <side-effects-end>
 */
package provider

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hieropold/tkncap/internal/account"
)

func init() {
	Register(&GeminiProvider{})
}

// GeminiProvider is the quota provider for Gemini accounts.
type GeminiProvider struct{}

/**
 * Kind
 *
 * <purpose-start>
 * Returns the provider kind so the registry can route Gemini accounts here.
 * <purpose-end>
 *
 * <inputs-start>
 * - None.
 * <inputs-end>
 *
 * <outputs-start>
 * - account.ProviderGemini.
 * <outputs-end>
 *
 * <side-effects-start>
 * - None.
 * <side-effects-end>
 */
func (g *GeminiProvider) Kind() account.Provider {
	return account.ProviderGemini
}

/**
 * Fetch
 *
 * <purpose-start>
 * Fetches the current usage quota for a Gemini account. Validates that API_KEY
 * is present, then sends a lightweight `countTokens` request to the Generative
 * Language API. The API returns rate limit information in headers such as
 * `x-ratelimit-limit-*` and `x-ratelimit-remaining-*`. These are parsed into
 * Quota results for requests per minute (RPM), tokens per minute (TPM), and
 * requests per day (RPD).
 * <purpose-end>
 *
 * <inputs-start>
 * - ctx context.Context: request context.
 * - a account.Account: the Gemini account whose quota is requested.
 * <inputs-end>
 *
 * <outputs-start>
 * - []Quota containing the fetch status, usage details, and any error message.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Performs an HTTP POST request to https://generativelanguage.googleapis.com.
 * - Logs debugging information via slog.
 * <side-effects-end>
 */
func (g *GeminiProvider) Fetch(ctx context.Context, a account.Account) []Quota {
	slog.Debug("gemini: fetching quota", "account", a.Name)

	apiKey := a.Fields["API_KEY"]
	if apiKey == "" {
		slog.Debug("gemini: account missing API_KEY field", "account", a.Name)
		return []Quota{{
			Account: a,
			Status:  StatusMisconfigured,
			Message: "missing required field TKNCAP_GEMINI_<ACCOUNT>_API_KEY",
		}}
	}

	// Use countTokens as a lightweight health/quota check
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:countTokens?key=" + apiKey
	body := strings.NewReader(`{"contents": [{"parts": [{"text": "Quota check"}]}]}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		slog.Debug("gemini: failed to create request", "account", a.Name, "error", err)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: fmt.Sprintf("failed to create HTTP request: %v", err),
		}}
	}
	req.Header.Set("Content-Type", "application/json")

	slog.Debug("gemini: sending API request", "account", a.Name, "url", "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:countTokens")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Debug("gemini: API request failed", "account", a.Name, "error", err)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: fmt.Sprintf("API request failed: %v", err),
		}}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("gemini: API returned non-OK status", "account", a.Name, "status", resp.StatusCode)
		msg := fmt.Sprintf("API returned status %d", resp.StatusCode)
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			msg = "API returned 401 Unauthorized: check if your API key is valid"
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

	var results []Quota

	// Helper to extract quota from headers (handling both x-ratelimit and x-goog-ratelimit)
	appendHeaderQuota := func(name, suffix string) {
		limitHeader := "x-ratelimit-limit-" + suffix
		remHeader := "x-ratelimit-remaining-" + suffix

		limitStr := resp.Header.Get(limitHeader)
		if limitStr == "" {
			limitStr = resp.Header.Get("x-goog-ratelimit-limit-" + suffix)
		}

		remStr := resp.Header.Get(remHeader)
		if remStr == "" {
			remStr = resp.Header.Get("x-goog-ratelimit-remaining-" + suffix)
		}

		if limitStr != "" && remStr != "" {
			limit, errL := strconv.ParseInt(limitStr, 10, 64)
			rem, errR := strconv.ParseInt(remStr, 10, 64)
			if errL == nil && errR == nil {
				used := limit - rem
				slog.Debug("gemini: successfully parsed quota header", "account", a.Name, "name", name, "used", used, "limit", limit)
				results = append(results, Quota{
					Account: a,
					Name:    name,
					Status:  StatusOK,
					Used:    &used,
					Limit:   &limit,
				})
			} else {
				slog.Debug("gemini: failed to parse rate limit headers as int", "account", a.Name, "name", name, "limitStr", limitStr, "remStr", remStr)
			}
		}
	}

	// Extract standard Gemini rate limit dimensions
	appendHeaderQuota("RPM", "requests-per-minute")
	appendHeaderQuota("TPM", "tokens-per-minute")
	appendHeaderQuota("RPD", "requests-per-day")

	if len(results) == 0 {
		slog.Debug("gemini: no rate limit headers found in response", "account", a.Name)
		return []Quota{{
			Account: a,
			Status:  StatusError,
			Message: "no rate limit headers returned by API; usage could not be determined",
		}}
	}

	return results
}
