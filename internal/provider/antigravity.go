/**
 * package provider — AntigravityProvider
 *
 * <purpose-start>
 * Stub implementation of the Provider interface for Antigravity quota. Returns
 * StatusMisconfigured if the account is missing the TOKEN field, otherwise
 * returns StatusUnimplemented. The real implementation will use the token to
 * authenticate with the Antigravity API and fetch the current quota data.
 * <purpose-end>
 *
 * <inputs-start>
 * - account.Account with Provider == ProviderAntigravity and a Fields map
 *   that should contain "TOKEN".
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
	Register(&AntigravityProvider{})
}

// AntigravityProvider is the stub quota provider for Antigravity accounts.
type AntigravityProvider struct{}

/**
 * Kind
 *
 * <purpose-start>
 * Returns the provider kind so the registry can route Antigravity accounts here.
 * <purpose-end>
 *
 * <inputs-start>
 * - None.
 * <inputs-end>
 *
 * <outputs-start>
 * - account.ProviderAntigravity.
 * <outputs-end>
 *
 * <side-effects-start>
 * - None.
 * <side-effects-end>
 */
func (a *AntigravityProvider) Kind() account.Provider {
	return account.ProviderAntigravity
}

/**
 * Fetch
 *
 * <purpose-start>
 * Stub quota fetch for an Antigravity account. Validates that TOKEN is present
 * (required for future real implementation). Returns StatusUnimplemented so
 * the CLI renders a placeholder row. Replace this body once the Antigravity
 * quota endpoint is integrated.
 * <purpose-end>
 *
 * <inputs-start>
 * - ctx context.Context: request context (unused in stub).
 * - a account.Account: the Antigravity account whose quota is requested.
 * <inputs-end>
 *
 * <outputs-start>
 * - Quota with Status=StatusMisconfigured if TOKEN is missing, or
 *   Status=StatusUnimplemented otherwise.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Logs the fetch attempt at debug level.
 * <side-effects-end>
 */
func (ag *AntigravityProvider) Fetch(ctx context.Context, a account.Account) Quota {
	slog.Debug("antigravity: fetching quota (stub)", "account", a.Name)

	if a.Fields["TOKEN"] == "" {
		slog.Debug("antigravity: account missing TOKEN field", "account", a.Name)
		return Quota{
			Account: a,
			Status:  StatusMisconfigured,
			Message: "missing required field TKNCAP_ANTIGRAVITY_<ACCOUNT>_TOKEN",
		}
	}

	slog.Debug("antigravity: stub — returning unimplemented", "account", a.Name)
	return Quota{
		Account: a,
		Status:  StatusUnimplemented,
		Message: "Antigravity quota API integration not yet implemented",
	}
}
