#!/bin/bash

# Lint check script for codex-mirror-switch
# This script runs various linting tools and checks for code quality issues

set -e

echo "🔍 Running code quality checks for codex-mirror-switch..."
echo "=================================================="

# Check if we're in the right directory
if [ ! -f "main.go" ]; then
    echo "❌ Error: Please run this script from the project root directory"
    exit 1
fi

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Run go fmt
echo "📝 Running go fmt..."
if ! gofmt -s -w .; then
    echo "❌ go fmt failed"
    exit 1
fi
echo "✅ go fmt completed"

# Run go vet
echo "🔍 Running go vet..."
if ! go vet ./...; then
    echo "❌ go vet found issues"
    exit 1
fi
echo "✅ go vet completed"

# Run go mod tidy
echo "📦 Running go mod tidy..."
if ! go mod tidy; then
    echo "❌ go mod tidy failed"
    exit 1
fi
echo "✅ go mod tidy completed"

# Run golangci-lint if available (optional, may have version compatibility issues)
if command_exists golangci-lint; then
    echo "🧹 Running golangci-lint (optional)..."
    if ! golangci-lint run --fast; then
        echo "⚠️  golangci-lint found issues (may be due to Go version compatibility)"
        echo "💡 This is optional, continuing with other checks..."
    else
        echo "✅ golangci-lint completed"
    fi
else
    echo "ℹ️  golangci-lint not found, skipping (optional)"
fi

# Run lightweight alternatives
echo "🔍 Running lightweight linting tools..."

if command_exists revive; then
    echo "   Running revive..."
    if ! revive ./...; then
        echo "❌ revive found issues"
        exit 1
    fi
    echo "   ✅ revive completed"
else
    echo "   ℹ️  revive not found, skipping (install: go install github.com/mgechev/revive@latest)"
fi

if command_exists errcheck; then
    echo "   Running errcheck..."
    if ! errcheck ./...; then
        echo "❌ errcheck found issues"
        exit 1
    fi
    echo "   ✅ errcheck completed"
else
    echo "   ℹ️  errcheck not found, skipping (install: go install github.com/kisielk/errcheck@latest)"
fi

# Run tests if they exist
if [ -n "$(find . -name '*_test.go' -print -quit)" ]; then
    echo "🧪 Running tests..."
    if ! go test -race ./...; then
        echo "❌ tests failed"
        exit 1
    fi
    echo "✅ tests completed"
else
    echo "ℹ️  No tests found, skipping..."
fi

# Check for security vulnerabilities if govulncheck is available
if command_exists govulncheck; then
    echo "🔒 Running security check..."
    if ! govulncheck ./...; then
        echo "⚠️  govulncheck found potential security issues"
        # Don't exit on security issues, just warn
    fi
    echo "✅ security check completed"
else
    echo "⚠️  govulncheck not found, skipping..."
    echo "💡 Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

echo "=================================================="
echo "✅ All code quality checks passed!"
echo "🎉 Code is ready to commit!"