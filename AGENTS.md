# Repository Guidelines

This repo is a Go CLI for the Google Search Console API.

## Setup And Common Commands

- Run tests: `go test ./...`
- Format: `gofmt -w .`
- Vet: `go vet ./...`
- Build: `go build ./...`
- Run locally: `go run . --help`

Go version is set in `go.mod` (currently `go 1.26.0`).

## Code Style

- UNIX philosophy: small, focused, correct changes.
- Idiomatic Go. Prefer straightforward code over cleverness.
- Comments explain why, not what.
- Be conservative with new dependencies. Prefer the standard library and existing deps.

## Running The CLI (Local Dev)

Credentials are stored on disk (see Security below). Typical flow:

- Write OAuth client credentials:
  - `go run . auth credentials ~/Downloads/client_secret_....json`
- Run OAuth installed-app flow and save tokens:
  - `go run . auth add`
  - Read-only scope: `go run . auth add --readonly`
- List configured accounts:
  - `go run . auth list`

Useful global flags:

- `--account <name>` (default: `default`)
- `--json` (see Output Rules)
- `--plain` (see Output Rules)
- `--no-input` (disable prompts; fail fast if required inputs are missing)
- `--verbose` (logs to stderr)
- `--color` (force color when output is a terminal; auto-detected otherwise)

## Project Structure

- `main.go`: entry point, calls `cmd.Execute()`.
- `cmd/`: Cobra commands and CLI wiring.
  - `cmd/root.go`: global flags, app context initialization, output mode, version printing.
- `internal/auth/`: OAuth flow + local credential store.
- `internal/cli/`: output formatting (`Printer`), including JSON/table/plain modes.

## Conventions For Changes

- Prefer small, verifiable diffs. Run the Preflight checks before finishing.
- Tests should verify behavior, not implementation details.
- For bug fixes, decide whether a regression test is worthwhile; prefer covering the failure mode when practical.
- When adding a new command:
  - Implement under `cmd/`.
  - Wire it into `cmd/root.go` via `rootCmd.AddCommand(...)`.
  - Start `RunE` with `app, err := appFrom(cmd)` and use `app.Printer` for output.
- Keep `--help` / `--version` fast and side-effect free.
  - `cmd/root.go` intentionally avoids creating `~/.gsc` during help/version invocations.

## Output Rules

- Default output is a table; `--plain` is tab-separated; `--json` prints structured JSON.
- Keep JSON stable and intentionally scoped for scripting.
- Never print OAuth token values. Avoid logging secrets even with `--verbose`.

## Security And Local State

The CLI writes credentials and tokens to the user’s home directory:

- Base dir: `~/.gsc/credentials/`
- Per account:
  - `~/.gsc/credentials/<account>/client.json`
  - `~/.gsc/credentials/<account>/token.json`

Rules:

- Do not commit real `client.json` or `token.json` files.
- Do not add tests that require live Google APIs or real credentials.
  - Use `httptest` for OAuth flow behavior (see `internal/auth/oauth_test.go`).
- Be careful when changing account naming/paths; account names are validated and directories are permissioned.

## Git Conventions

- PR titles: Conventional Commits `type(scope): description` (lowercase).
  - Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`
- Commits: plain lowercase; no conventional format requirement.
- Keep PRs atomic: one concern per PR.

## Changelog

Use Keep a Changelog format. User-facing changes only.

Sections: Added, Changed, Deprecated, Removed, Fixed, Security.

## Preflight

Before finishing (and before opening a PR or asking for review), run the following commands:

- `gofmt -w .`
- `go vet ./...`
- `go test ./...`
