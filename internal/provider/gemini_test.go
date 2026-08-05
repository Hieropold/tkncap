// Package provider tests cover GeminiProvider: Kind() and the missing-API_KEY
// validation path in Fetch.
package provider

import (
	"context"
	"testing"

	"github.com/hieropold/tkncap/internal/account"
)

func TestGeminiProvider_Kind(t *testing.T) {
	p := &GeminiProvider{}
	if got := p.Kind(); got != account.ProviderGemini {
		t.Errorf("Kind() = %v, want %v", got, account.ProviderGemini)
	}
}

func TestGeminiProvider_Fetch_Validation(t *testing.T) {
	p := &GeminiProvider{}
	ctx := context.Background()

	t.Run("missing API_KEY", func(t *testing.T) {
		a := account.Account{
			Provider: account.ProviderGemini,
			Name:     "test",
			Fields:   map[string]string{},
		}
		got := p.Fetch(ctx, a)
		if len(got) != 1 {
			t.Fatalf("len(got) = %d, want 1", len(got))
		}
		if got[0].Status != StatusMisconfigured {
			t.Errorf("status = %v, want %v", got[0].Status, StatusMisconfigured)
		}
	})
}
