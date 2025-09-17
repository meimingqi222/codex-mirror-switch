# Scripts 目录

这个目录包含了项目的构建和代码质量检查脚本。

## 脚本说明

### lint.sh
基础的代码质量检查脚本，运行 golangci-lint。

**用途：**
- 日常开发中的快速检查
- CI/CD 流程中的代码质量门禁

**使用方法：**
```bash
# 直接运行
./scripts/lint.sh

# 通过 Makefile 运行
make lint
```

### lint-full.sh
完整的代码质量检查脚本，包含多种检查工具。

**检查内容：**
- 代码格式 (gofmt)
- 静态分析 (go vet)
- 依赖文件检查 (go mod tidy)
- 高级 lint (golangci-lint)
- 可选工具 (revive, errcheck)

**用途：**
- 代码提交前的全面检查
- 发布前的质量保证

**使用方法：**
```bash
# 直接运行
./scripts/lint-full.sh

# 通过 Makefile 运行
make lint-check
```

## 构建流程

### 标准构建
```bash
# 完整流程：lint + 下载依赖 + 构建
make build
```

### 快速构建
```bash
# 跳过 lint 检查，仅下载依赖和构建
make build-fast
```

### 仅检查代码质量
```bash
# 基础检查
make lint

# 完整检查
make lint-check
```

## 输出目录

- **构建输出**: `build/codex-mirror`
- **发布包**: `dist/` (交叉编译时使用)