// Package account tests cover Discover: single account, multiple accounts of
// the same provider, multiple providers, malformed variables (too few
// segments, wrong prefix, unknown provider), and empty input.
package account

import (
	"testing"
)

func TestDiscover(t *testing.T) {
	tests := []struct {
		name     string
		env      []string
		wantLen  int
		wantAccs []Account // check subset of fields; use nil to skip detail check
	}{
		{
			name:    "empty env returns no accounts",
			env:     []string{},
			wantLen: 0,
		},
		{
			name: "unrelated vars are ignored",
			env: []string{
				"PATH=/usr/bin",
				"HOME=/home/user",
				"CLAUDE_API_KEY=abc", // missing TKNCAP_ prefix
			},
			wantLen: 0,
		},
		{
			name: "single claude account",
			env: []string{
				"TKNCAP_CLAUDE_WORK_CREDENTIALS_PATH=/home/user/.claude/.credentials.json",
			},
			wantLen: 1,
			wantAccs: []Account{
				{Provider: ProviderClaude, Name: "work", Fields: map[string]string{
					"CREDENTIALS_PATH": "/home/user/.claude/.credentials.json",
				}},
			},
		},

		{
			name: "two claude accounts accumulate separately",
			env: []string{
				"TKNCAP_CLAUDE_WORK_CREDENTIALS_PATH=/home/user/.claude-work/.credentials.json",
				"TKNCAP_CLAUDE_PERSONAL_CREDENTIALS_PATH=/home/user/.claude-personal/.credentials.json",
			},
			wantLen: 2,
		},
		{
			name: "mixed providers each produce their own account",
			env: []string{
				"TKNCAP_CLAUDE_WORK_CREDENTIALS_PATH=/some/path",
				"TKNCAP_GEMINI_MAIN_API_KEY=AIzaFake",
			},
			wantLen: 2,
		},
		{
			name: "multiple fields for same account merged into one Account",
			env: []string{
				"TKNCAP_GEMINI_MAIN_API_KEY=AIzaFake",
				"TKNCAP_GEMINI_MAIN_PROJECT_ID=my-project",
			},
			wantLen: 1,
			wantAccs: []Account{
				{Provider: ProviderGemini, Name: "main", Fields: map[string]string{
					"API_KEY":    "AIzaFake",
					"PROJECT_ID": "my-project",
				}},
			},
		},
		{
			name: "unknown provider segment is skipped",
			env: []string{
				"TKNCAP_OPENAI_WORK_API_KEY=sk-fake",
			},
			wantLen: 0,
		},
		{
			name: "too few segments after prefix are skipped",
			env: []string{
				"TKNCAP_CLAUDE=bad",       // only 1 segment after TKNCAP_
				"TKNCAP_CLAUDE_WORK=bad",  // only 2 segments (no field)
			},
			wantLen: 0,
		},
		{
			name: "field name with underscores is preserved correctly",
			env: []string{
				"TKNCAP_CLAUDE_WORK_CREDENTIALS_PATH=/path/to/creds",
			},
			wantLen: 1,
			wantAccs: []Account{
				{Provider: ProviderClaude, Name: "work", Fields: map[string]string{
					"CREDENTIALS_PATH": "/path/to/creds",
				}},
			},
		},
		{
			name: "account name is lowercased",
			env: []string{
				"TKNCAP_GEMINI_MYACCOUNT_API_KEY=key",
			},
			wantLen: 1,
			wantAccs: []Account{
				{Provider: ProviderGemini, Name: "myaccount"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Discover(tt.env)
			if err != nil {
				t.Fatalf("Discover returned unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Errorf("got %d accounts, want %d; accounts: %+v", len(got), tt.wantLen, got)
			}
			for i, want := range tt.wantAccs {
				if i >= len(got) {
					t.Errorf("missing account at index %d: want %+v", i, want)
					continue
				}
				g := got[i]
				if g.Provider != want.Provider {
					t.Errorf("[%d] provider: got %q, want %q", i, g.Provider, want.Provider)
				}
				if g.Name != want.Name {
					t.Errorf("[%d] name: got %q, want %q", i, g.Name, want.Name)
				}
				for field, wantVal := range want.Fields {
					if gotVal, ok := g.Fields[field]; !ok {
						t.Errorf("[%d] missing field %q", i, field)
					} else if gotVal != wantVal {
						t.Errorf("[%d] field %q: got %q, want %q", i, field, gotVal, wantVal)
					}
				}
			}
		})
	}
}
