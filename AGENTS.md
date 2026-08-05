---
description:
globs:
alwaysApply: true
---
# Instructions

The project is designed with an **AI-First** philosophy, meaning its architecture and code are structured to be easily understood, modified, and tested by an AI. Modern LLMs read and infer code structure directly, so doc comments should not restate what is already obvious from the code (types, parameter names, control flow). Instead they should carry the information an AI (or human) cannot infer just by reading the signature: the business rationale, references to spec/change docs, and non-obvious side effects.

## Doc Comments & Conventions (Mandatory)

Doc comments should follow Go idioms (GoDoc: `//` comments starting with the identifier name). They are mandatory on exported functions, types, and non-obvious logical blocks. Focus on **WHY** (purpose, business rationale, spec/change-doc references) and **non-obvious side effects** (file I/O, network calls, env-var reads, global/state mutation, process exit) — avoid redundant parameter/return type wrappers already covered by the code signature.

- Add or update these comments for every new or modified exported function, type, or non-obvious logical block. When modifying existing code, read and understand the existing doc comment first, and make sure your change doesn't silently break the contract it documents. If there is no such comment yet, add one as you touch the code.
- Comments must be in English and use Go's native `//` doc-comment format — do **not** use XML-like wrapper tags (`<purpose-start>`, `<inputs-start>`, `<outputs-start>`, `<side-effects-start>`, etc.) and do not restate parameter/return types that are already self-evident from the function signature.
- Reference the relevant change summary doc (`docs/<issue-id>.md`, see below) when the rationale is non-trivial or tied to a specific request.

Example:
```go
// isUserAllowed checks id against the SERBOROBOT_ALLOWED_USERS allowlist.
//
// Side effects: increments the allowed/denied access counter metrics.
func isUserAllowed(username string) bool {
	...
}
```

## Change Summary Docs

Track all changes in the `docs/` folder. It should contain a summary document with the changes made and the reasoning behind those changes. The file should be named using the GitHub issue id, e.g. `docs/582.md`. Check the change summary doc as a last step in the implementation of a change — confirm it exists and is aligned with the actual implementation. Reference it from relevant doc comments (`Ref: docs/582.md`) when applicable.

## Extensive Logging

Another important aspect is the use of extensive logging via `log/slog`. Each log message includes the package name and context fields to help trace the execution flow. Logging always writes to `stderr` so it never corrupts the table or JSON output on `stdout`.

IMPORTANT: Logging should be used extensively throughout the codebase. Every significant action, decision point, or error condition should be logged with sufficient detail to allow for effective debugging and tracing of the program's execution flow.

Use `TKNCAP_LOG_LEVEL=debug` (or `--log-level debug`) to enable verbose tracing.

## Development Workflow

1. **Understand the Context**: Before starting work on a task, read `./docs/architecture.md` and any other relevant docs in the `./docs` directory to understand the overall system design, the provider registry pattern, and the env-var account convention.

2. **Identify the Target**: Locate the function or system you want to work on. Read its doc comment to understand the existing rationale and constraints.

3. **Implement the Code**: Implement the change, adding verbose logging for each step — this is crucial for debugging and tracing execution. Make sure the implementation doesn't silently break the contract described by any existing doc comment.

4. **Update Doc Comments**: Update the doc comment to reflect the new behaviour if needed. If the change alters the exported contract (signature, inputs/outputs), update all call sites to match. Contract changes should be avoided when possible since they are risky and can break other parts of the system.

5. **Write or Update Tests**: After implementing the code, write or update tests so expected inputs produce expected outputs, using the doc comment as the reference contract. This is crucial for validating that your implementation meets the requirements.

6. **Run and Verify**: Execute the tests to verify that your implementation works as intended. If any tests fail, debug the issues, make necessary corrections, and re-run the tests until they all pass.

## MCP Docs Server

Always use Context7 when you need code generation, setup or configuration steps, or library/API documentation. This means you should automatically use the Context7 MCP tools to resolve the library ID and get library docs **without the user having to explicitly ask**. This applies to all libraries and frameworks used in this project: `cobra`, `slog`, future provider SDKs (Anthropic, Google Gemini), and any new dependency.

Workflow:
1. Call `mcp__plugin_context7_context7__resolve-library-id` to get the library ID.
2. Call `mcp__plugin_context7_context7__query-docs` with the resolved ID and a specific query.

## Git Operations

**Only git read operations are allowed.** Do not run `git commit`, `git push`, `git merge`, `git rebase`, or any other write operation without explicit user confirmation for each individual action. Allowed git operations include: `git status`, `git diff`, `git log`, `git show`, `git branch`, `git remote -v`.

## Project-Specific Notes

### Module

```
github.com/hieropold/tkncap
```

### Providers

| Provider    | Constant              | Required Field        | Env-Var Example                                    |
|-------------|----------------------|-----------------------|----------------------------------------------------|
| claude      | `ProviderClaude`      | `CREDENTIALS_PATH`    | `TKNCAP_CLAUDE_WORK_CREDENTIALS_PATH=~/.claude/...` |
| gemini      | `ProviderGemini`      | `API_KEY`             | `TKNCAP_GEMINI_MAIN_API_KEY=AIza...`                |

### Adding a New Provider

1. Add a constant to `internal/account/account.go` (`knownProviders` map).
2. Create `internal/provider/<name>.go` with a struct implementing `Provider`.
3. Register via `func init() { Register(&YourProvider{}) }`.
4. Update this file and `docs/architecture.md`.
5. Add `account_test.go` cases for the new provider segment.

### Account Discovery

Accounts are discovered from environment variables at runtime. No config file is used. See `internal/account/account.go` and `docs/architecture.md` for the full env-var convention.

### Output

- Default: aligned table via `text/tabwriter` (no external dep).
- With `--json`: JSON array via `encoding/json`, pipe-friendly (`tkncap show --json | jq .`).
