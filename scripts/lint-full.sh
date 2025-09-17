#!/bin/bash

# 完整的代码质量检查脚本
# 包含多种 lint 工具和检查

set -e

echo "🔍 正在运行完整的代码质量检查..."

# 错误计数
ERRORS=0

# 1. 检查代码格式
echo "📝 检查代码格式..."
if [ "$(gofmt -l . | wc -l)" -ne 0 ]; then
    echo "❌ 代码格式错误:"
    gofmt -l .
    ERRORS=$((ERRORS + 1))
else
    echo "✅ 代码格式检查通过"
fi

# 2. 运行 go vet
echo "🔍 运行 go vet..."
if go vet ./...; then
    echo "✅ go vet 检查通过"
else
    echo "❌ go vet 检查失败"
    ERRORS=$((ERRORS + 1))
fi

# 3. 检查依赖文件
echo "📦 检查依赖文件..."
go mod tidy
if [ -n "$(git status --porcelain go.mod go.sum 2>/dev/null)" ]; then
    echo "❌ 依赖文件有未提交的更改"
    ERRORS=$((ERRORS + 1))
else
    echo "✅ 依赖文件检查通过"
fi

# 4. 运行 golangci-lint
echo "🔬 运行 golangci-lint..."
if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "📦 安装 golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

if golangci-lint run ./...; then
    echo "✅ golangci-lint 检查通过"
else
    echo "❌ golangci-lint 检查失败"
    ERRORS=$((ERRORS + 1))
fi

# 5. 运行 revive (如果安装了)
echo "🔍 运行 revive (如果可用)..."
if command -v revive >/dev/null 2>&1; then
    if revive ./...; then
        echo "✅ revive 检查通过"
    else
        echo "❌ revive 检查失败"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo "⚠️  revive 未安装，跳过 (安装: go install github.com/mgechev/revive@latest)"
fi

# 6. 运行 errcheck (如果安装了)
echo "🔍 运行 errcheck (如果可用)..."
if command -v errcheck >/dev/null 2>&1; then
    if errcheck ./...; then
        echo "✅ errcheck 检查通过"
    else
        echo "❌ errcheck 检查失败"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo "⚠️  errcheck 未安装，跳过 (安装: go install github.com/kisielk/errcheck@latest)"
fi

# 总结
echo ""
if [ $ERRORS -eq 0 ]; then
    echo "🎉 所有检查通过！"
    exit 0
else
    echo "❌ 发现 $ERRORS 个错误，请修复后重试"
    exit 1
fi