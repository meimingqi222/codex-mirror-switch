#!/bin/bash

# Lint 检查脚本
# 运行代码质量检查

set -e

echo "🔍 正在进行代码质量检查..."

# 检查是否安装了 golangci-lint
if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "📦 安装 golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

# 运行 golangci-lint
echo "🔬 运行 golangci-lint..."
if golangci-lint run ./...; then
    echo "✅ golangci-lint 检查通过"
    exit 0
else
    echo "❌ golangci-lint 检查失败"
    exit 1
fi