/**
 * package account
 *
 * <purpose-start>
 * Defines the Account type that represents a single named user account for a
 * quota provider, and the Discover function that reads all TKNCAP_* environment
 * variables to build the list of configured accounts.
 *
 * The env-var convention is:
 *   TKNCAP_<PROVIDER>_<ACCOUNT>_<FIELD>=<value>
 * where PROVIDER ∈ {CLAUDE, GEMINI, ANTIGRAVITY}, ACCOUNT is a user label
 * (single underscore-free token), and FIELD is a provider-specific key.
 *
 * Variables whose PROVIDER segment is not one of the known providers are
 * silently ignored. Malformed variables (fewer than three segments after
 * stripping the prefix) are also ignored.
 * <purpose-end>
 *
 * <inputs-start>
 * - env []string: slice of "KEY=VALUE" strings, typically os.Environ().
 * <inputs-end>
 *
 * <outputs-start>
 * - []Account: one Account per (provider, name) pair found in env.
 * - error: non-nil only if internal invariants are violated (currently unused,
 *   reserved for future validation).
 * <outputs-end>
 *
 * <side-effects-start>
 * - None. Pure function over the provided env slice.
 * <side-effects-end>
 */
package account

import (
	"log/slog"
	"strings"
)

// Provider is a typed string that identifies which quota service an account belongs to.
type Provider string

const (
	ProviderClaude      Provider = "claude"
	ProviderGemini      Provider = "gemini"
	ProviderAntigravity Provider = "antigravity"
)

// knownProviders maps the uppercase env-var segment to the canonical Provider value.
var knownProviders = map[string]Provider{
	"CLAUDE":      ProviderClaude,
	"GEMINI":      ProviderGemini,
	"ANTIGRAVITY": ProviderAntigravity,
}

/**
 * Account
 *
 * <purpose-start>
 * Represents a single named account for a quota provider. Each Account is
 * derived from one or more environment variables that share the same
 * TKNCAP_<PROVIDER>_<ACCOUNT>_ prefix. The Fields map holds the raw
 * key-value pairs discovered for that account (e.g. {"CREDENTIALS_PATH": "/home/..."}).
 * <purpose-end>
 *
 * <inputs-start>
 * - N/A (struct definition).
 * <inputs-end>
 *
 * <outputs-start>
 * - N/A (struct definition).
 * <outputs-end>
 *
 * <side-effects-start>
 * - None.
 * <side-effects-end>
 */
type Account struct {
	// Provider is the quota service this account belongs to.
	Provider Provider
	// Name is the user-chosen label (lowercased), e.g. "work", "personal".
	Name string
	// Fields holds the raw env-var field names and their values for this account.
	Fields map[string]string
}

// accountKey is used internally to group env-var segments by (provider, name).
type accountKey struct {
	provider string // uppercase, e.g. "CLAUDE"
	name     string // uppercase, e.g. "WORK"
}

/**
 * Discover
 *
 * <purpose-start>
 * Walks the provided env slice and extracts all accounts encoded as
 * TKNCAP_<PROVIDER>_<ACCOUNT>_<FIELD>=<value> variables. Variables that do
 * not match the prefix, have fewer than three segments after the prefix, or
 * reference an unknown provider are silently skipped. The returned slice is
 * stable: accounts appear in the order their first env-var was encountered.
 * <purpose-end>
 *
 * <inputs-start>
 * - env []string: "KEY=VALUE" pairs, typically os.Environ().
 * <inputs-end>
 *
 * <outputs-start>
 * - []Account: deduplicated accounts sorted by first-occurrence order.
 * - error: reserved for future validation; always nil in current implementation.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Logs each parsed and skipped variable at debug level via slog.
 * <side-effects-end>
 */
func Discover(env []string) ([]Account, error) {
	const prefix = "TKNCAP_"

	// Use a map to accumulate fields per (provider, account) key.
	fieldMap := map[accountKey]map[string]string{}
	// Preserve insertion order for stable output.
	var order []accountKey

	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}

		if !strings.HasPrefix(key, prefix) {
			continue
		}

		// Strip the "TKNCAP_" prefix and split into at least 3 segments.
		rest := key[len(prefix):]
		parts := strings.SplitN(rest, "_", 3)
		if len(parts) < 3 {
			slog.Debug("account: skipping malformed env var (fewer than 3 segments after prefix)",
				"var", key)
			continue
		}

		providerSeg := parts[0] // e.g. "CLAUDE"
		accountSeg := parts[1]  // e.g. "WORK"
		fieldSeg := parts[2]    // e.g. "CREDENTIALS_PATH"

		if _, known := knownProviders[providerSeg]; !known {
			slog.Debug("account: skipping env var with unknown provider segment",
				"var", key, "provider_segment", providerSeg)
			continue
		}

		ak := accountKey{provider: providerSeg, name: accountSeg}
		if _, exists := fieldMap[ak]; !exists {
			fieldMap[ak] = map[string]string{}
			order = append(order, ak)
			slog.Debug("account: discovered new account",
				"provider", providerSeg, "account", accountSeg)
		}

		fieldMap[ak][fieldSeg] = value
		slog.Debug("account: recorded field",
			"provider", providerSeg, "account", accountSeg, "field", fieldSeg)
	}

	accounts := make([]Account, 0, len(order))
	for _, ak := range order {
		p := knownProviders[ak.provider]
		accounts = append(accounts, Account{
			Provider: p,
			Name:     strings.ToLower(ak.name),
			Fields:   fieldMap[ak],
		})
	}

	slog.Debug("account: discovery complete", "count", len(accounts))
	return accounts, nil
}
