# Repository Guidelines

## Project Structure & Modules
- `main.go`: CLI entry point.
- `cmd/`: Cobra subcommands (`add`, `list`, `remove`, `status`, `switch`, `root`).
- `internal/`: Core logic: mirrors, Codex/VS Code config, platform paths, shared types (`mirror.go`, `codex.go`, `vscode.go`, `platform.go`, `types.go`).
- `config/`: Sample/default configuration (if added later).
- `pkg/`: Reserved for exported libraries (currently unused).
- Artifacts: `codex-mirror.exe` in repo is a sample build.

## Build, Test, and Development
- Build: `go build -o codex-mirror.exe` (Windows) or `go build -o codex-mirror` (Unix).
- Run locally: `go run . --help` or `./codex-mirror.exe status`.
- Tests: `go test ./...` (add `-v` for verbose). No tests exist yet; add them near code.
- Format & vet: `go fmt ./...` and `go vet ./...` before pushing.

## Coding Style & Naming
- Language: Go modules (Go 1.24.x). Prefer idiomatic Go and small packages.
- Packages/files: lower-case, short names; avoid underscores.
- Identifiers: exported use CamelCase; unexported start lower-case.
- Commands: one file per verb in `cmd/` (e.g., `switch.go`) and keep flag names kebab-case.
- Errors: wrap with context (`fmt.Errorf("switch failed: %w", err)`).

## Testing Guidelines
- Place `*_test.go` beside implementation in the same package.
- Use table-driven tests; mock or isolate file/system interactions in `internal/`.
- Aim for coverage on command handlers and mirror/config logic. Run `go test ./...` locally.

## Commit & Pull Request Guidelines
- Commits: imperative mood, concise subject (~72 chars), meaningful body (why + what). Reference issues (`Fixes #12`).
- PRs: include a clear description, linked issues, steps to verify (sample commands/output), and note any config path changes.
- Keep diffs focused; update `README.md` and this guide when flags/paths or behavior change.

## Security & Configuration Tips
- Do not commit real API keys; use placeholders in examples.
- Config is user-scoped; typical paths include `%USERPROFILE%\.codex-mirror\config.toml` (Windows) and `~/.codex-mirror/config.toml` (Unix). Use helpers in `internal/platform.go` for cross‑platform paths.

