/**
 * package provider — ClaudeProvider
 *
 * <purpose-start>
 * Stub implementation of the Provider interface for Claude Code quota. Returns
 * StatusMisconfigured if the account is missing the CREDENTIALS_PATH field,
 * otherwise returns StatusUnimplemented. The real implementation will use
 * the credentials file to authenticate with the Anthropic API and fetch the
 * current usage limits for the authenticated user.
 * <purpose-end>
 *
 * <inputs-start>
 * - account.Account with Provider == ProviderClaude and a Fields map that
 *   should contain "CREDENTIALS_PATH".
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
	Register(&ClaudeProvider{})
}

// ClaudeProvider is the stub quota provider for Claude Code accounts.
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

/**
 * Fetch
 *
 * <purpose-start>
 * Stub quota fetch for a Claude Code account. Validates that CREDENTIALS_PATH
 * is present in the account fields (required for future real implementation).
 * Returns StatusUnimplemented so the CLI renders a meaningful placeholder row
 * rather than an error. Replace this body when the Anthropic quota API is known.
 * <purpose-end>
 *
 * <inputs-start>
 * - ctx context.Context: request context (unused in stub).
 * - a account.Account: the Claude account whose quota is requested.
 * <inputs-end>
 *
 * <outputs-start>
 * - Quota with Status=StatusMisconfigured if CREDENTIALS_PATH is missing, or
 *   Status=StatusUnimplemented otherwise.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Logs the fetch attempt at debug level.
 * <side-effects-end>
 */
func (c *ClaudeProvider) Fetch(ctx context.Context, a account.Account) Quota {
	slog.Debug("claude: fetching quota (stub)", "account", a.Name)

	if a.Fields["CREDENTIALS_PATH"] == "" {
		slog.Debug("claude: account missing CREDENTIALS_PATH field", "account", a.Name)
		return Quota{
			Account: a,
			Status:  StatusMisconfigured,
			Message: "missing required field TKNCAP_CLAUDE_<ACCOUNT>_CREDENTIALS_PATH",
		}
	}

	slog.Debug("claude: stub — returning unimplemented", "account", a.Name)
	return Quota{
		Account: a,
		Status:  StatusUnimplemented,
		Message: "Claude quota API integration not yet implemented",
	}
}
