# tkncap — Architecture

## Overview

`tkncap` is a CLI tool that reads token quota information for user accounts across multiple AI provider services: Claude Code, Gemini, and Antigravity. Each service call is isolated behind a `Provider` interface so implementations can be added independently without changing the command layer.

## Repository Layout

```
tkncap/
├── main.go                       # Thin entrypoint — calls cmd.Execute()
├── cmd/
│   ├── root.go                   # Root cobra command; global --json, --log-level flags
│   ├── show.go                   # `tkncap show` — discovery → provider dispatch → render
│   └── version.go                # `tkncap version`
└── internal/
    ├── account/
    │   ├── account.go            # Account type + Discover(env []string)
    │   └── account_test.go
    ├── provider/
    │   ├── provider.go           # Provider interface, Quota type, registry
    │   ├── claude.go             # ClaudeProvider (stub)
    │   ├── gemini.go             # GeminiProvider (stub)
    │   └── antigravity.go        # AntigravityProvider (stub)
    ├── logging/
    │   └── logging.go            # slog initialisation from TKNCAP_LOG_LEVEL
    └── output/
        ├── output.go             # Renderer interface
        ├── table.go              # text/tabwriter table renderer
        └── json.go               # encoding/json renderer
```

## Data Flow

```
os.Environ()
    │
    ▼
account.Discover()          ← groups TKNCAP_* vars into []Account
    │
    ▼
for each Account:
    provider.For(kind)      ← registry lookup (registered via init())
    p.Fetch(ctx, account)   ← returns Quota (status + data or error)
    │
    ▼
output.Renderer.Render()    ← table (default) or JSON (--json flag)
    │
    ▼
os.Stdout
```

## Account Discovery

All accounts are configured via environment variables. No config file is used. The naming convention:

```
TKNCAP_<PROVIDER>_<ACCOUNT>_<FIELD>=<value>
```

- **PROVIDER**: `CLAUDE`, `GEMINI`, or `ANTIGRAVITY` (case-sensitive uppercase).
- **ACCOUNT**: A user-chosen label (uppercase, single token with no underscores), e.g. `WORK`, `PERSONAL`, `MAIN`.
- **FIELD**: A provider-specific key, e.g. `CREDENTIALS_PATH`, `API_KEY`, `TOKEN`.

Multiple accounts of the same provider are supported by using different ACCOUNT labels:

```bash
TKNCAP_CLAUDE_WORK_CREDENTIALS_PATH=/home/user/.claude/.credentials.json
TKNCAP_CLAUDE_PERSONAL_CREDENTIALS_PATH=/home/user/.claude-personal/.credentials.json
TKNCAP_GEMINI_MAIN_API_KEY=AIzaFake
TKNCAP_ANTIGRAVITY_DEFAULT_TOKEN=tok123
```

## Provider Interface

```go
type Provider interface {
    Kind() account.Provider
    Fetch(ctx context.Context, a account.Account) Quota
}
```

Implementations live in `internal/provider/`. Each file registers itself in `init()`:

```go
func init() {
    Register(&ClaudeProvider{})
}
```

The registry is populated before `main()` runs, so `cmd/show.go` can call `provider.For(kind)` without knowing which concrete types exist.

## Adding a New Provider

1. Create `internal/provider/<name>.go`.
2. Define a struct implementing `Provider`.
3. Add a `func init() { Register(&YourProvider{}) }`.
4. Add the provider string constant to `internal/account/account.go` (`knownProviders` map).
5. Update `docs/architecture.md` and `CLAUDE.md` to document the new required fields.
6. Add tests in `internal/account/account_test.go` for the new provider segment.

## Output Formats

| Format | Flag | Implementation |
|--------|------|----------------|
| Table  | (default) | `output.TableRenderer` — `text/tabwriter` |
| JSON   | `--json` | `output.JSONRenderer` — `encoding/json` |

## Logging

All logging uses `log/slog` writing to `stderr` (never stdout, so it does not corrupt table/JSON output). The level is controlled by the `TKNCAP_LOG_LEVEL` env-var (or `--log-level` flag), defaulting to `info`. Set `TKNCAP_LOG_LEVEL=debug` to trace the full discovery and fetch flow.

## Build & Release

```bash
# Development build
go build -o tkncap .

# Release build with version metadata
go build -ldflags "
  -X github.com/hieropold/tkncap/cmd.version=1.0.0
  -X github.com/hieropold/tkncap/cmd.commit=$(git rev-parse --short HEAD)
  -X github.com/hieropold/tkncap/cmd.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)
" -o tkncap .
```
