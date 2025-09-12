# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based CLI tool for managing and switching between different API mirror sources for Codex CLI and VS Code extensions. It allows users to add, remove, list, and switch between different API endpoints while automatically updating configuration files and environment variables.

## Build & Development Commands

### Common Commands
- `make build` - Build for current platform
- `make test` - Run tests with coverage
- `make fmt` - Format code with gofmt
- `make vet` - Run go vet for static analysis
- `make lint` - Run golangci-lint (if installed)
- `make install` - Install binary locally
- `make clean` - Clean build artifacts

### Testing & Quality
- `make coverage` - Generate coverage report
- `make security` - Check for vulnerabilities with govulncheck
- `go test ./...` - Alternative test command
- `go run . --help` - Run locally without building

## Architecture Overview

### Core Components
- **Mirror Management** (`internal/mirror.go`): Manages mirror source configurations stored in TOML format
- **Configuration Handlers**: 
  - `internal/codex.go`: Handles Codex CLI config files (~/.codex/config.toml, ~/.codex/auth.json)
  - `internal/vscode.go`: Manages VS Code settings (platform-specific paths)
- **Platform Support** (`internal/platform.go`): Cross-platform path resolution and environment variable persistence
- **Type Definitions** (`internal/types.go`): Shared structs for configs and platform types

### Command Structure
Each command is implemented as a separate file in `cmd/`:
- `root.go`: Root command with global helpers
- `add.go`: Add new mirror sources
- `list.go`: List available mirrors
- `remove.go`: Delete mirror sources
- `switch.go`: Switch active mirror (with --codex-only, --vscode-only flags)
- `status.go`: Show current mirror status

### Configuration Management
The tool manages three types of configuration:
1. **Mirror config**: ~/.codex-mirror/mirrors.toml (app's own config)
2. **Codex CLI config**: ~/.codex/config.toml + auth.json
3. **VS Code settings**: Platform-specific settings.json files

Environment variables are persisted across platforms:
- Windows: Registry via `setx`
- macOS: ~/.zshrc and ~/.bash_profile
- Linux: ~/.bashrc and ~/.profile

## Development Guidelines

### Code Style
- Follow idiomatic Go patterns
- Use exported CamelCase for public APIs, camelCase for private
- Keep command implementations in separate files
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Use TOML for configuration files

### Testing Strategy
- Place `*_test.go` files beside implementation
- Use table-driven tests for command handlers
- Mock file system interactions for `internal/` package tests
- Test cross-platform path resolution logic

### Configuration Security
- Never commit real API keys (use placeholders)
- All API keys are masked in output (show only first/last 4 chars)
- Configuration stored in user directories, not system-wide
- Environment variables use `CODEX_<MIRROR_NAME>_API_KEY` format

### Dependencies
- `github.com/spf13/cobra` - CLI framework
- `github.com/BurntSushi/toml` - TOML parsing
- Go 1.23.12+ required (to match golangci-lint compatibility)

### Common Development Tasks
1. Add new command: Create file in `cmd/`, register in `cmd/root.go`
2. Modify configuration: Update structs in `internal/types.go`
3. Add platform support: Extend `internal/platform.go`
4. Test changes: Use `make test` and cross-reference with `AGENTS.md`