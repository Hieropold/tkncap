---
description:
globs:
alwaysApply: true
---
# Instructions

The project is designed with an **AI-First** philosophy, meaning its architecture and code are structured to be easily understood, modified, and tested by an AI. The core of this philosophy is a system of structured, machine-readable comments called **semantic markup blocks**.

A semantic markup block is a formal, XML-like specification placed in a documentation comment (docstring) directly above a function or class. It serves as a detailed "brief" for the AI, providing a clear contract for what the code should do, how it should do it, and how to verify its correctness.

This structured approach is more robust than informal instructions, enabling the AI to generate higher-quality, context-aware code with greater reliability.

## Semantic Markup Rules

- The comments describing the purpose of each changed or updated logical block of code, function or class should be added when applying those changes. These comments should focus on documenting the reasons behind the code, not the implementation details - i.e. focus should be on WHY, not HOW. The documenting comment should contain sections: purpose, inputs, outputs, and side effects.
- These comments should be added to all new or modified code blocks, ensuring clarity and maintainability for future developers. When modifying existing code, please ensure to read and understand the existing doc comments, and make sure that updates won't break the contract established by those comments. If there are no such comments in the existing code, please add them as you are applying changes.
- Sections should be clearly formatted with XML-like tags or sections for easier reading and parsing. Best practice is to add such comment to each logical block of the code, like class, function or component. This structure provides better anchors by using clear unique (in the scope of each doc comment) delimiters that improves readability and provides additional embedded context to help with understanding the implementation and making changes that will take into account existing architecture, implementation and any potential side-effects of changes in a particular code block.

Example of the semantic markup block:
# Go
```go
/**
 * isUserAllowed
 *
 * <purpose-start>
 * This function checks if a user is in the list of allowed users.
 * The list of allowed users is read from the SERBOROBOT_ALLOWED_USERS environment variable.
 * <purpose-end>
 *
 * <inputs-start>
 * - username: The username of the user to check.
 * <inputs-end>
 *
 * <outputs-start>
 * - bool: True if the user is allowed, false otherwise.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Increases the counter metrics for allowed and denied access.
 * <side-effects-end>
 */
```

## Extensive Logging

Another important aspect is the use of extensive logging via `log/slog`. Each log message includes the package name and context fields to help trace the execution flow. Logging always writes to `stderr` so it never corrupts the table or JSON output on `stdout`.

IMPORTANT: Logging should be used extensively throughout the codebase. Every significant action, decision point, or error condition should be logged with sufficient detail to allow for effective debugging and tracing of the program's execution flow.

Use `TKNCAP_LOG_LEVEL=debug` (or `--log-level debug`) to enable verbose tracing.

## Development Workflow

1. **Understand the Context**: Before starting work on a task, read `./docs/architecture.md` and any other relevant docs in the `./docs` directory to understand the overall system design, the provider registry pattern, and the env-var account convention.

2. **Identify the Target**: Locate the function or system you want to work on. Read its semantic markup block to understand the requirements and constraints.

3. **Implement the Code**: Use the information in the semantic markup block to guide your implementation. Ensure that your code adheres to the specified constraints and fulfils the goal. Make sure that there is verbose logging for each step in the code. This is CRUCIAL for debugging and understanding the code. Check that the contract specified by inputs and outputs of the semantic markup is not violated.

4. **Align with Semantic Markup Block**: Update the semantic markup block to match the new requirements, if needed. If new requirements absolutely and directly require a contract change — all calling code should be updated to match the new implementation. Contract updates should be avoided if possible because it is a risky operation which will potentially break other parts of the system.

5. **Write or Update Tests**: After implementing the code, write or update the tests to check that expected inputs produce expected outputs using the semantic markup block for reference. This is crucial for validating that your implementation meets the requirements.

6. **Run and Verify**: Execute the tests to verify that your implementation works as intended. If any tests fail, debug the issues, make necessary corrections, and re-run the tests until they all pass.

## MCP Docs Server

Always use Context7 when you need code generation, setup or configuration steps, or library/API documentation. This means you should automatically use the Context7 MCP tools to resolve the library ID and get library docs **without the user having to explicitly ask**. This applies to all libraries and frameworks used in this project: `cobra`, `slog`, future provider SDKs (Anthropic, Google Gemini, Antigravity), and any new dependency.

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
| antigravity | `ProviderAntigravity` | `TOKEN`               | `TKNCAP_ANTIGRAVITY_DEFAULT_TOKEN=tok...`           |

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
