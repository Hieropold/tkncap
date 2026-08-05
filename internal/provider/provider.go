// Package provider defines the Provider interface and Quota type shared by
// all quota-provider implementations, plus a registry mapping
// account.Provider values to Provider implementations. Implementations
// self-register via Register from their package init() functions so
// cmd/show.go can look them up without importing each concrete package
// explicitly.
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

// Quota carries the result of a single quota fetch for one account. Pointer
// fields (Used, Limit, ResetsAt) use nil to distinguish "not available" from
// a genuine zero value.
type Quota struct {
	Account  account.Account
	Name     string     // e.g. "5-hour" or "7-day" (optional, for multiple limits per account)
	Status   Status
	Used     *int64     // nil when unknown
	Limit    *int64     // nil when unknown
	ResetsAt *time.Time // nil when unknown
	Message  string
}

// Provider is the interface each quota-provider implementation must satisfy.
// Fetch returns a slice rather than a single Quota to support providers that
// expose multiple limits per account (e.g. 5-hour and 7-day windows).
type Provider interface {
	Kind() account.Provider
	Fetch(ctx context.Context, a account.Account) []Quota
}

// registry holds the mapping from account.Provider → Provider implementation.
var registry = map[account.Provider]Provider{}

// Register adds a Provider implementation to the package-level registry.
// Intended to be called from each provider package's init() (see claude.go,
// gemini.go).
//
// Side effects: panics on duplicate registration for the same Kind, to catch
// accidental double-registration at startup.
func Register(p Provider) {
	kind := p.Kind()
	if _, exists := registry[kind]; exists {
		panic("provider: duplicate registration for " + string(kind))
	}
	registry[kind] = p
}

// For looks up the registered Provider for kind. Returns nil if none is
// registered, which callers should treat as a configuration error.
func For(kind account.Provider) Provider {
	return registry[kind]
}
