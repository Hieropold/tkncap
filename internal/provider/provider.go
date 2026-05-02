/**
 * package provider
 *
 * <purpose-start>
 * Defines the Provider interface that all quota-provider implementations must
 * satisfy, the Quota type that carries a single account's quota data, and a
 * package-level registry that maps account.Provider values to their Provider
 * implementations. Provider implementations register themselves via Register
 * in their package init() functions so that cmd/show.go can look them up
 * without importing each concrete package explicitly.
 * <purpose-end>
 *
 * <inputs-start>
 * - N/A (package definition).
 * <inputs-end>
 *
 * <outputs-start>
 * - N/A (package definition).
 * <outputs-end>
 *
 * <side-effects-start>
 * - The registry map is mutated by Register calls from init() functions.
 * <side-effects-end>
 */
package provider

import (
	"context"
	"time"

	"github.com/hieropold/tkncap/internal/account"
)

// Status describes the outcome of a quota fetch attempt.
type Status string

const (
	// StatusOK means the quota was fetched successfully.
	StatusOK Status = "ok"
	// StatusUnimplemented means the provider stub has not been implemented yet.
	StatusUnimplemented Status = "unimplemented"
	// StatusError means the fetch failed due to a runtime error.
	StatusError Status = "error"
	// StatusMisconfigured means the account is missing required fields.
	StatusMisconfigured Status = "misconfigured"
)

/**
 * Quota
 *
 * <purpose-start>
 * Carries the result of a single quota fetch for one account. Pointer fields
 * (Used, Limit, ResetsAt) use nil to distinguish "not available" from zero.
 * The Message field provides human-readable detail for non-OK statuses.
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
type Quota struct {
	Account  account.Account
	Status   Status
	Used     *int64     // nil when unknown
	Limit    *int64     // nil when unknown
	ResetsAt *time.Time // nil when unknown
	Message  string
}

/**
 * Provider
 *
 * <purpose-start>
 * Interface that each quota-provider implementation must satisfy. Kind returns
 * the provider type so the registry can route accounts to the correct
 * implementation. Fetch performs the quota lookup for a single account and
 * returns a populated Quota (never returns error; errors are encoded in
 * Quota.Status and Quota.Message to allow partial results).
 * <purpose-end>
 *
 * <inputs-start>
 * - N/A (interface definition).
 * <inputs-end>
 *
 * <outputs-start>
 * - N/A (interface definition).
 * <outputs-end>
 *
 * <side-effects-start>
 * - None (interface contract; concrete implementations may have side effects).
 * <side-effects-end>
 */
type Provider interface {
	Kind() account.Provider
	Fetch(ctx context.Context, a account.Account) Quota
}

// registry holds the mapping from account.Provider → Provider implementation.
var registry = map[account.Provider]Provider{}

/**
 * Register
 *
 * <purpose-start>
 * Adds a Provider implementation to the package-level registry. Intended to be
 * called from the init() function of each provider package (claude, gemini,
 * antigravity). Panics if a provider for the same Kind is registered twice to
 * catch accidental double-registration at startup.
 * <purpose-end>
 *
 * <inputs-start>
 * - p Provider: the implementation to register.
 * <inputs-end>
 *
 * <outputs-start>
 * - None.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Mutates the package-level registry map.
 * - Panics on duplicate registration.
 * <side-effects-end>
 */
func Register(p Provider) {
	kind := p.Kind()
	if _, exists := registry[kind]; exists {
		panic("provider: duplicate registration for " + string(kind))
	}
	registry[kind] = p
}

/**
 * For
 *
 * <purpose-start>
 * Looks up the registered Provider for a given account.Provider kind. Returns
 * nil if no implementation has been registered for that kind, which callers
 * should treat as a configuration error.
 * <purpose-end>
 *
 * <inputs-start>
 * - kind account.Provider: the provider type to look up.
 * <inputs-end>
 *
 * <outputs-start>
 * - Provider: the registered implementation, or nil if not found.
 * <outputs-end>
 *
 * <side-effects-start>
 * - None.
 * <side-effects-end>
 */
func For(kind account.Provider) Provider {
	return registry[kind]
}
