/**
 * package provider — GeminiProvider
 *
 * <purpose-start>
 * Stub implementation of the Provider interface for Gemini quota. Returns
 * StatusMisconfigured if the account is missing the API_KEY field, otherwise
 * returns StatusUnimplemented. The real implementation will call the Gemini
 * API to retrieve the current token quota and usage for the key's project.
 * <purpose-end>
 *
 * <inputs-start>
 * - account.Account with Provider == ProviderGemini and a Fields map that
 *   should contain "API_KEY".
 * <inputs-end>
 *
 * <outputs-start>
 * - Quota with Status=StatusMisconfigured (missing field) or
 *   Status=StatusUnimplemented (field present, not yet implemented).
 * <outputs-end>
 *
 * <side-effects-start>
 * - Logs the fetch attempt and its outcome at debug level.
 * - Registers itself in the provider registry via init().
 * <side-effects-end>
 */
package provider

import (
	"context"
	"log/slog"

	"github.com/hieropold/tkncap/internal/account"
)

func init() {
	Register(&GeminiProvider{})
}

// GeminiProvider is the stub quota provider for Gemini accounts.
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
 * Stub quota fetch for a Gemini account. Validates that API_KEY is present
 * (required for future real implementation). Returns StatusUnimplemented so
 * the CLI renders a placeholder row. Replace this body once the Gemini quota
 * endpoint is integrated.
 * <purpose-end>
 *
 * <inputs-start>
 * - ctx context.Context: request context (unused in stub).
 * - a account.Account: the Gemini account whose quota is requested.
 * <inputs-end>
 *
 * <outputs-start>
 * - Quota with Status=StatusMisconfigured if API_KEY is missing, or
 *   Status=StatusUnimplemented otherwise.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Logs the fetch attempt at debug level.
 * <side-effects-end>
 */
func (g *GeminiProvider) Fetch(ctx context.Context, a account.Account) Quota {
	slog.Debug("gemini: fetching quota (stub)", "account", a.Name)

	if a.Fields["API_KEY"] == "" {
		slog.Debug("gemini: account missing API_KEY field", "account", a.Name)
		return Quota{
			Account: a,
			Status:  StatusMisconfigured,
			Message: "missing required field TKNCAP_GEMINI_<ACCOUNT>_API_KEY",
		}
	}

	slog.Debug("gemini: stub — returning unimplemented", "account", a.Name)
	return Quota{
		Account: a,
		Status:  StatusUnimplemented,
		Message: "Gemini quota API integration not yet implemented",
	}
}
