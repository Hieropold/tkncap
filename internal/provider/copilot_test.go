// Package provider tests cover CopilotProvider: Kind(), token resolution
// (CREDENTIALS_PATH file path, gh CLI fallback, and misconfigured/error
// cases) via resolveCopilotToken, JSONC comment stripping, token extraction
// determinism, and quota-row construction for both the paid
// (quota_snapshots) and free/limited (monthly_quotas) response shapes.
package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hieropold/tkncap/internal/account"
)

func TestCopilotProvider_Kind(t *testing.T) {
	p := &CopilotProvider{}
	if got := p.Kind(); got != account.ProviderCopilot {
		t.Errorf("Kind() = %v, want %v", got, account.ProviderCopilot)
	}
}

// withCopilotTokenSeams overrides the lookPath/runGhAuthToken package-level
// seams for the duration of a test and restores the originals on cleanup, so
// tests never depend on the host's actual gh CLI installation/auth state.
func withCopilotTokenSeams(t *testing.T, lp func(string) (string, error), ghToken func(context.Context) (string, error)) {
	t.Helper()
	origLookPath, origRunGhAuthToken := lookPath, runGhAuthToken
	lookPath = lp
	runGhAuthToken = ghToken
	t.Cleanup(func() {
		lookPath = origLookPath
		runGhAuthToken = origRunGhAuthToken
	})
}

func TestResolveCopilotToken(t *testing.T) {
	ctx := context.Background()
	a := account.Account{Provider: account.ProviderCopilot, Name: "test"}

	ghNotFound := func(string) (string, error) { return "", errors.New("executable file not found in $PATH") }
	ghFound := func(string) (string, error) { return "/usr/bin/gh", nil }

	t.Run("no CREDENTIALS_PATH and gh CLI not found is misconfigured", func(t *testing.T) {
		withCopilotTokenSeams(t, ghNotFound, func(context.Context) (string, error) {
			t.Fatal("runGhAuthToken should not be called when gh is not on PATH")
			return "", nil
		})

		token, status, _ := resolveCopilotToken(ctx, a, "")
		if token != "" {
			t.Errorf("token = %q, want empty", token)
		}
		if status != StatusMisconfigured {
			t.Errorf("status = %v, want %v", status, StatusMisconfigured)
		}
	})

	t.Run("credentials file missing token and gh CLI unavailable is an error", func(t *testing.T) {
		dir := t.TempDir()
		credPath := filepath.Join(dir, "config.json")
		if err := os.WriteFile(credPath, []byte(`{}`), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		withCopilotTokenSeams(t, ghNotFound, func(context.Context) (string, error) {
			t.Fatal("runGhAuthToken should not be called when gh is not on PATH")
			return "", nil
		})

		token, status, _ := resolveCopilotToken(ctx, a, credPath)
		if token != "" {
			t.Errorf("token = %q, want empty", token)
		}
		if status != StatusError {
			t.Errorf("status = %v, want %v", status, StatusError)
		}
	})

	t.Run("credentials file has token: gh CLI fallback is never attempted", func(t *testing.T) {
		dir := t.TempDir()
		credPath := filepath.Join(dir, "config.json")
		content := `{"copilotTokens": {"https://github.com:alice": "gho_fromfile"}}`
		if err := os.WriteFile(credPath, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		withCopilotTokenSeams(t, func(string) (string, error) {
			t.Fatal("lookPath should not be called when the file already yields a token")
			return "", nil
		}, func(context.Context) (string, error) {
			t.Fatal("runGhAuthToken should not be called when the file already yields a token")
			return "", nil
		})

		token, status, _ := resolveCopilotToken(ctx, a, credPath)
		if token != "gho_fromfile" {
			t.Errorf("token = %q, want gho_fromfile", token)
		}
		if status != "" {
			t.Errorf("status = %v, want empty", status)
		}
	})

	t.Run("no CREDENTIALS_PATH falls back to gh auth token on success", func(t *testing.T) {
		withCopilotTokenSeams(t, ghFound, func(context.Context) (string, error) {
			return "gho_fromgh", nil
		})

		token, status, _ := resolveCopilotToken(ctx, a, "")
		if token != "gho_fromgh" {
			t.Errorf("token = %q, want gho_fromgh", token)
		}
		if status != "" {
			t.Errorf("status = %v, want empty", status)
		}
	})

	t.Run("gh CLI present but gh auth token fails is an error, not misconfigured", func(t *testing.T) {
		withCopilotTokenSeams(t, ghFound, func(context.Context) (string, error) {
			return "", errors.New("not logged in")
		})

		token, status, _ := resolveCopilotToken(ctx, a, "")
		if token != "" {
			t.Errorf("token = %q, want empty", token)
		}
		if status != StatusError {
			t.Errorf("status = %v, want %v", status, StatusError)
		}
	})

	t.Run("credentials file has no copilotTokens entry falls back to gh auth token", func(t *testing.T) {
		dir := t.TempDir()
		credPath := filepath.Join(dir, "config.json")
		// Mirrors a real ~/.copilot/config.json that has no copilotTokens key at
		// all (e.g. keyring-backed auth), including JSONC comment lines.
		content := "// managed file\n{\n  \"appTipShown\": true\n}\n"
		if err := os.WriteFile(credPath, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		withCopilotTokenSeams(t, ghFound, func(context.Context) (string, error) {
			return "gho_fromgh", nil
		})

		token, status, _ := resolveCopilotToken(ctx, a, credPath)
		if token != "gho_fromgh" {
			t.Errorf("token = %q, want gho_fromgh", token)
		}
		if status != "" {
			t.Errorf("status = %v, want empty", status)
		}
	})
}

func TestCopilotProvider_Fetch_Misconfigured(t *testing.T) {
	p := &CopilotProvider{}
	ctx := context.Background()

	t.Run("no CREDENTIALS_PATH and no gh CLI on PATH", func(t *testing.T) {
		withCopilotTokenSeams(t, func(string) (string, error) { return "", errors.New("not found") }, runGhAuthToken)

		a := account.Account{
			Provider: account.ProviderCopilot,
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

func TestStripJSONCComments(t *testing.T) {
	input := []byte("{\n  // a comment\n  \"a\": 1,\n    // another\n  \"b\": 2\n}\n")
	want := "{\n  \"a\": 1,\n  \"b\": 2\n}\n"
	got := string(stripJSONCComments(input))
	if got != want {
		t.Errorf("stripJSONCComments() = %q, want %q", got, want)
	}
}

func TestExtractToken(t *testing.T) {
	t.Run("empty map returns empty string", func(t *testing.T) {
		if got := extractToken(map[string]string{}); got != "" {
			t.Errorf("extractToken() = %q, want empty", got)
		}
	})

	t.Run("single entry returned as-is", func(t *testing.T) {
		tokens := map[string]string{"https://github.com:alice": "gho_alice"}
		if got := extractToken(tokens); got != "gho_alice" {
			t.Errorf("extractToken() = %q, want gho_alice", got)
		}
	})

	t.Run("multiple entries pick alphabetically first key deterministically", func(t *testing.T) {
		tokens := map[string]string{
			"https://github.com:zed":   "gho_zed",
			"https://github.com:alice": "gho_alice",
			"https://github.com:bob":   "gho_bob",
		}
		for i := 0; i < 10; i++ {
			if got := extractToken(tokens); got != "gho_alice" {
				t.Fatalf("extractToken() = %q, want gho_alice (iteration %d)", got, i)
			}
		}
	})
}

func TestBuildCopilotQuotas_PaidPlanSnapshots(t *testing.T) {
	a := account.Account{Provider: account.ProviderCopilot, Name: "work"}
	body := copilotUserResponse{
		CopilotPlan:     "individual",
		QuotaResetDateU: "2026-09-01T00:00:00Z",
	}
	body.QuotaSnapshots.PremiumInteractions = copilotQuotaSnapshot{Entitlement: 300, Remaining: 250}
	body.QuotaSnapshots.Completions = copilotQuotaSnapshot{Unlimited: true}
	body.QuotaSnapshots.Chat = copilotQuotaSnapshot{Unlimited: true}

	got := buildCopilotQuotas(a, body, body.CopilotPlan)
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3", len(got))
	}

	premium := got[0]
	if premium.Name != "premium" || premium.Used == nil || *premium.Used != 50 || premium.Limit == nil || *premium.Limit != 300 {
		t.Errorf("premium row = %+v, want Used=50 Limit=300", premium)
	}
	if premium.Message != "individual" {
		t.Errorf("premium.Message = %q, want %q", premium.Message, "individual")
	}
	wantReset, _ := time.Parse(time.RFC3339, "2026-09-01T00:00:00Z")
	if premium.ResetsAt == nil || !premium.ResetsAt.Equal(wantReset) {
		t.Errorf("premium.ResetsAt = %v, want %v", premium.ResetsAt, wantReset)
	}

	completions := got[1]
	if completions.Name != "completions" || completions.Limit != nil || completions.Used == nil || *completions.Used != 0 {
		t.Errorf("completions row = %+v, want unlimited (Limit=nil, Used=0)", completions)
	}
	if completions.Message != "unlimited" {
		t.Errorf("completions.Message = %q, want %q", completions.Message, "unlimited")
	}
}

func TestBuildCopilotQuotas_FreeLimitedPlan(t *testing.T) {
	a := account.Account{Provider: account.ProviderCopilot, Name: "personal"}
	body := copilotUserResponse{
		MonthlyQuotas:        map[string]int64{"chat": 50, "completions": 2000},
		LimitedUserQuotas:    map[string]int64{"chat": 10, "completions": 500},
		LimitedUserResetDate: "2026-09-01",
	}

	got := buildCopilotQuotas(a, body, "")
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	byName := map[string]int64{}
	for _, q := range got {
		if q.Limit == nil {
			t.Errorf("row %q: Limit is nil, want a value", q.Name)
			continue
		}
		byName[q.Name] = *q.Limit
	}
	if byName["completions"] != 2000 {
		t.Errorf("completions Limit = %d, want 2000 (monthly_quotas total, not limited_user_quotas remaining)", byName["completions"])
	}
	if byName["chat"] != 50 {
		t.Errorf("chat Limit = %d, want 50", byName["chat"])
	}
}
