// Package account discovers configured provider accounts from environment
// variables of the form TKNCAP_<PROVIDER>_<ACCOUNT>_<FIELD>=<value>, where
// PROVIDER is one of the known providers (see knownProviders) and ACCOUNT is
// a user-chosen label. This is the only supported configuration source —
// there is no config file — so this convention must stay in sync with
// docs/architecture.md and AGENTS.md whenever a provider is added.
package account

import (
	"log/slog"
	"strings"
)

// Provider is a typed string that identifies which quota service an account belongs to.
type Provider string

const (
	ProviderClaude  Provider = "claude"
	ProviderGemini  Provider = "gemini"
	ProviderCopilot Provider = "copilot"
)

// knownProviders maps the uppercase env-var segment to the canonical Provider value.
var knownProviders = map[string]Provider{
	"CLAUDE":  ProviderClaude,
	"GEMINI":  ProviderGemini,
	"COPILOT": ProviderCopilot,
}

// Account represents a single named account for a quota provider, derived
// from the group of environment variables sharing one
// TKNCAP_<PROVIDER>_<ACCOUNT>_ prefix.
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

// Discover extracts all accounts encoded in env as
// TKNCAP_<PROVIDER>_<ACCOUNT>_<FIELD>=<value> variables. Vars that don't
// match the prefix, have fewer than three segments, or name an unknown
// provider are silently skipped rather than erroring, so unrelated
// environment variables never break account discovery. The returned slice
// preserves first-occurrence order for stable output; the error return is
// reserved for future validation and is currently always nil.
//
// Side effects: logs each parsed and skipped variable at debug level.
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
