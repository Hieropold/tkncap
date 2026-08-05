// GeminiProvider implements Provider for Gemini API accounts. It sends a
// lightweight countTokens request and reads rate-limit info out of the
// response headers rather than a dedicated quota endpoint, since the
// Generative Language API has no such endpoint.
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

// Kind identifies this as the Gemini provider so the registry can route
// Gemini accounts here.
func (g *GeminiProvider) Kind() account.Provider {
	return account.ProviderGemini
}

// Fetch retrieves the current requests-per-minute, tokens-per-minute, and
// requests-per-day quota for a Gemini account. There is no dedicated quota
// endpoint, so this sends a minimal countTokens request and parses the rate
// limit info Google returns in x-ratelimit-*/x-goog-ratelimit-* response
// headers.
//
// Side effects: makes an outbound HTTP POST request to
// generativelanguage.googleapis.com.
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
