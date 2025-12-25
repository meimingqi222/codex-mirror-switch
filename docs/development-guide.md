# Development Guide

This guide provides comprehensive information about the project architecture, development workflow, and best practices for contributing to codex-mirror-switch.

> **Note**: This file was originally created as `CLAUDE.md` for Claude Code integration, but has been moved to the docs directory to serve as general development documentation for all contributors.

## Project Overview

A Go-based CLI tool for managing and switching between different API mirror sources for Claude Code, Codex CLI, and VS Code extensions. It allows users to add, remove, list, and switch between different API endpoints while automatically updating configuration files and environment variables.

The tool supports two tool types:
- **Claude** (`claude`): Sets environment variables only (`ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, optional `ANTHROPIC_MODEL`), no config file modifications
- **Codex** (`codex`): Updates Codex CLI config files, VS Code settings.json, and environment variables

## Build & Development Commands

### Common Commands
- `make build` - Build for current platform (runs lint, deps, then build)
- `make build-fast` - Quick build skipping lint checks
- `make build-all` - Cross-compile for all platforms (Windows amd64/arm64, Linux amd64/arm64, macOS amd64/arm64)
- `make test` - Run tests with race detection and coverage
- `make fmt` - Format code with gofmt
- `make vet` - Run go vet for static analysis
- `make lint` - Run golangci-lint via scripts/lint.sh
- `make lint-ci` - Run CI-friendly checks (vet, fmt, mod tidy)
- `make install` - Install binary to $GOPATH/bin
- `make clean` - Clean build artifacts
- `make package` - Create release packages (zip for Windows, tar.gz for Unix)

### Testing & Quality
- `make coverage` - Generate HTML coverage report (coverage.html)
- `make security` - Check for vulnerabilities with govulncheck
- `go test ./...` - Run all tests
- `go test -run TestSwitch ./cmd` - Run specific test
- `go test -run TestMirrorManager ./internal` - Run specific internal package test
- `go run main.go --help` - Run locally without building

### Version Information
Version info is injected at build time via ldflags:
- `cmd.Version` - Git tag or "dev"
- `cmd.BuildTime` - Build timestamp
- `cmd.GitCommit` - Short commit hash
- View with: `./build/codex-mirror version` or `make version`

## Architecture Overview

### Core Components (internal/)
- **Mirror Management** (`mirror.go`):
  - Manages mirror source configurations stored in TOML format at `~/.codex-mirror/mirrors.toml`
  - `MirrorManager` struct provides methods: `AddMirror()`, `RemoveMirror()`, `ListMirrors()`, `GetMirror()`, `SetCurrentMirror()`
  - Supports environment variable discovery on first run
  - Includes soft-delete support (Deleted flag + DeletedAt timestamp)

- **Configuration Handlers**:
  - `codex.go`: Handles Codex CLI config files (`~/.codex/config.toml`, `~/.codex/auth.json`)
  - `vscode.go`: Manages VS Code settings.json with platform-specific paths, preserves existing fields when updating chatgpt.apiBase
  - `env.go`: Cross-platform environment variable persistence
    - Windows: Registry via `setx` command (user-level)
    - macOS/Linux: Shell profile files (~/.bashrc, ~/.zshrc, etc.)

- **Sync System** (`sync*.go`, `sync_gist.go`, `conflict.go`):
  - Cloud sync via Gist, WebDAV, or custom providers
  - Optional API key encryption using `crypto.go`
  - Conflict resolution for multi-device configs
  - Selective API key syncing (SyncAPIKeys flag)

- **Platform Support** (`platform.go`): Cross-platform path resolution for config directories
- **Type Definitions** (`types.go`):
  - Centralized structs: `MirrorConfig`, `SystemConfig`, `CodexConfig`, `VSCodeSettings`, `SyncConfig`, etc.
  - Platform and shell constants: `WindowsOS`, `MacOS`, `LinuxOS`, `BashShell`, `ZshShell`, etc.
  - Environment variable constants: `AnthropicBaseURLEnv`, `AnthropicAuthTokenEnv`, `CodexSwitchAPIKeyEnv`, etc.

### Command Structure (cmd/)
Each command is a separate file using Cobra framework:
- `root.go`: Root command with global helpers (`maskAPIKey()` shows first/last 4 chars only)
- `add.go`: Add new mirror sources with validation (URL format, required fields)
- `list.go`: List available mirrors with current selection marker (separate tracking for claude/codex types)
- `remove.go`: Delete mirror sources with confirmation
- `switch.go`: Switch active mirror
  - Optional flags: `--codex-only`, `--vscode-only`, `--no-backup`, `--shell`
  - Shell integration support for immediate env var export in current session
- `status.go`: Show current mirror status across all tools (Claude Code, Codex CLI, VS Code)
- `init.go` / `uninit.go`: Shell integration setup/removal
  - Adds/removes wrapper functions to shell profile files
  - Supports bash, zsh, fish, PowerShell (with OneDrive path detection on Windows)
- `sync*.go`: Cloud sync commands (`sync push`, `sync pull`, `sync status`, `sync init`)
- `sync_resolve.go`: Interactive conflict resolution for sync conflicts
- `sync_help.go`: Detailed sync feature documentation
- `doctor.go`: Health check command that diagnoses configuration issues
  - Checks config file integrity, environment variables, VS Code/Codex config
  - Optional mirror connectivity testing with `--skip-test` flag
  - Returns `CheckResult` structs with status (ok/warning/error/skipped)
- `test.go`: Test mirror connectivity to validate API endpoints
- `update.go`: Update mirror configuration (API keys, URLs, etc.)
- `version.go`: Display version, build time, and git commit info
- `clean.go`: Clean up orphaned/invalid configurations

### Configuration Management
The tool manages three types of configuration:

1. **Mirror config** (`~/.codex-mirror/mirrors.toml`):
   - TOML format with `current_codex`, `current_claude`, and `[[mirrors]]` array
   - Each mirror has: name, base_url, api_key, tool_type (claude/codex), optional model_name
   - Includes timestamps (CreatedAt, LastModified) and soft-delete fields (Deleted, DeletedAt)
   - Optional sync section for cloud sync configuration

2. **Codex CLI config**:
   - `~/.codex/config.toml`: model_provider, model, model_providers map, etc.
   - `~/.codex/auth.json`: Contains OPENAI_API_KEY
   - Preserves unknown fields via OtherFields map for forward compatibility

3. **VS Code settings** (platform-specific paths):
   - Windows: `%APPDATA%\Code\User\settings.json`
   - macOS: `~/Library/Application Support/Code/User/settings.json`
   - Linux: `~/.config/Code/User/settings.json`
   - Updates chatgpt.apiBase while preserving all other settings via OtherSettings map

### Environment Variables
**Claude type mirrors** set:
- `ANTHROPIC_BASE_URL`: API base URL
- `ANTHROPIC_AUTH_TOKEN`: Claude API token
- `ANTHROPIC_MODEL`: Model name (optional, if specified in mirror config)
- Extra environment variables via `ExtraEnv` map in mirror config (e.g., `ANTHROPIC_DEFAULT_HAIKU_MODEL`)

**Codex type mirrors** set:
- `CODEX_SWITCH_OPENAI_API_KEY`: Tool's own env var for API key

**Configuration discovery:**
- `CODEX_MIRROR_CONFIG_PATH`: Override default config location (`~/.codex-mirror/mirrors.toml`)

**Persistence mechanisms:**
- **Windows**: Registry via `setx` command (user-level, permanent)
- **macOS**: Writes to `~/.zshrc` and `~/.bash_profile`
- **Linux**: Writes to `~/.bashrc` and `~/.profile`
- **Shell integration**: `init` command adds wrapper functions for immediate effect in current session

## Development Guidelines

### Code Style
- Follow idiomatic Go patterns
- Use exported CamelCase for public APIs, camelCase for private
- Keep command implementations in separate files under `cmd/`
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Use TOML for configuration files
- Place test files (`*_test.go`) beside implementation
- Use table-driven tests and mock file system interactions in internal/ packages

### Testing Strategy
- Unit tests for `internal/` packages with mocked file I/O (see `mirror_test.go`, `codex_test.go`, `vscode_test.go`)
- Command handler tests in `cmd/cmd_test.go` using test doubles
- Test cross-platform path resolution logic in `platform_test.go`
- Use `internal.Test*` constants (defined in `types.go`) for test data consistency
- Run `make test` before committing (includes race detection and coverage)

### Configuration Security
- Never commit real API keys (use placeholders like "sk-test-key")
- All API keys masked in output via `maskAPIKey()` in [root.go:67](cmd/root.go#L67) (shows first/last 4 chars only)
- Optional API key encryption for cloud sync via [internal/crypto.go](internal/crypto.go)
- Configuration stored in user directories, not system-wide
- Gist sync uses device-specific encryption keys
- Use `CODEX_MIRROR_CONFIG_PATH` for testing with isolated config files

### Dependencies
- `github.com/spf13/cobra` - CLI framework for command structure
- `github.com/BurntSushi/toml` - TOML configuration parsing
- Go 1.23+ required (Go 1.23.12+ recommended for golangci-lint compatibility)

### Key Implementation Details

**Atomic Configuration Writes:**
- Configuration saves use atomic writes via temp file + rename pattern (see [mirror.go:68-110](internal/mirror.go#L68-L110))
- Prevents corruption if write operation is interrupted
- Cross-platform safe (handles Windows file locking)

**Forward Compatibility:**
- `CodexConfig.OtherFields` map preserves unknown TOML fields when reading/writing config
- `VSCodeSettings.OtherSettings` map preserves all settings except chatgpt.apiBase

**Soft Delete:**
- Mirrors are soft-deleted with `Deleted: true` flag and `DeletedAt` timestamp
- Sync system tracks deleted mirrors in `SyncData.DeletedMirrors` for proper multi-device sync

**Shell Integration:**
- `init`/`uninit` commands add/remove marked blocks: `# >>> codex-mirror init >>>` ... `# <<< codex-mirror init <<<`
- Fish shell uses separate function file: `~/.config/fish/functions/codex-mirror.fish`
- PowerShell checks for OneDrive redirected Documents folder on Windows

**Cross-Platform Paths:**
- Use `internal.GetPathConfig()` for platform-specific config directories
- All path operations use `filepath.Join()` for cross-platform compatibility

**Data Validation:**
- Mirror configs include data checksums to detect corruption
- Version field tracks config format changes for migrations

### Adding New Features

**Add new command:**
1. Create new file in `cmd/` (e.g., `cmd/mynewcmd.go`)
2. Define command with Cobra: `var mynewcmdCmd = &cobra.Command{...}`
3. Register in `cmd/root.go` init(): `rootCmd.AddCommand(mynewcmdCmd)`
4. Add tests in `cmd/cmd_test.go`

**Modify configuration format:**
1. Update structs in [internal/types.go](internal/types.go)
2. Update `MirrorManager.loadConfig()` and `saveConfig()` in [internal/mirror.go](internal/mirror.go)
3. Add migration logic if changing existing fields
4. Update tests to verify backward compatibility

**Add new platform:**
1. Add platform constant to [internal/types.go](internal/types.go) (e.g., `PlatformFreeBSD`)
2. Extend `GetPathConfig()` in [internal/platform.go](internal/platform.go)
3. Update `SetEnvVar()` and `GetEnvVar()` in [internal/env.go](internal/env.go) if needed
4. Add platform-specific tests in [internal/platform_test.go](internal/platform_test.go)

**Troubleshooting Configuration Issues:**
1. Use `codex-mirror doctor` to diagnose common problems
2. Use `codex-mirror doctor --verbose` for detailed output
3. Use `codex-mirror doctor --skip-test` to skip connectivity checks
4. Check `~/.codex-mirror/mirrors.toml` for corruption
5. Use `CODEX_MIRROR_CONFIG_PATH` env var for testing with isolated configs