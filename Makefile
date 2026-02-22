# Makefile for codex-mirror-switch

# 变量定义
APP_NAME := codex-mirror
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 相关变量
GO := go
GOFLAGS := -ldflags="-s -w -X codex-mirror/cmd.Version=$(VERSION) -X codex-mirror/cmd.BuildTime=$(BUILD_TIME) -X codex-mirror/cmd.GitCommit=$(GIT_COMMIT)"
GOMOD := $(GO) mod
GOBUILD := $(GO) build
GOTEST := $(GO) test
GOVET := $(GO) vet
GOFMT := gofmt

# 平台和架构
PLATFORMS := windows linux darwin
ARCHITECTURES := amd64 arm64

# 输出目录
BUILD_DIR := build
DIST_DIR := dist

# 默认目标
.PHONY: all
all: clean lint test build

# 安装依赖
.PHONY: deps
deps:
	@echo "正在下载依赖..."
	$(GOMOD) download
	$(GOMOD) tidy

# 代码格式化
.PHONY: fmt
fmt:
	@echo "正在格式化代码..."
	$(GOFMT) -s -w .

# 代码检查
.PHONY: vet
vet:
	@echo "正在进行代码检查..."
	$(GOVET) ./...

# 运行测试
.PHONY: test
test:
	@echo "正在运行测试..."
	$(GOTEST) -race -coverprofile=coverage.out ./...

# 查看测试覆盖率
.PHONY: coverage
coverage: test
	@echo "正在生成覆盖率报告..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 代码质量检查（使用golangci-lint）
.PHONY: lint
lint:
	@./scripts/lint.sh

# 运行轻量级 lint 工具（可选）
.PHONY: lint-light
lint-light:
	@echo "运行轻量级代码质量检查..."
	@if command -v revive >/dev/null 2>&1; then \
		echo "运行 revive..."; \
		revive ./...; \
	else \
		echo "revive 未安装，跳过"; \
		echo "安装: go install github.com/mgechev/revive@latest"; \
	fi
	@if command -v errcheck >/dev/null 2>&1; then \
		echo "运行 errcheck..."; \
		errcheck ./...; \
	else \
		echo "errcheck 未安装，跳过"; \
		echo "安装: go install github.com/kisielk/errcheck@latest"; \
	fi

# 运行完整的代码质量检查脚本（包含golangci-lint）
.PHONY: lint-check
lint-check:
	@echo "正在运行完整的代码质量检查..."
	@./scripts/lint-full.sh

# 运行基础检查（CI友好）
.PHONY: lint-ci
lint-ci:
	@echo "运行CI代码质量检查..."
	@$(GOVET) ./...
	@if [ "$$(gofmt -l . | wc -l)" -ne 0 ]; then \
		echo "代码格式错误:"; \
		gofmt -l .; \
		exit 1; \
	fi
	@$(GOMOD) tidy
	@if [ -n "$$(git status --porcelain go.mod go.sum 2>/dev/null)" ]; then \
		echo "依赖文件有未提交的更改"; \
		exit 1; \
	fi

# 构建 CLI 版本
.PHONY: build-cli
build-cli: lint deps
	@echo "正在构建 $(APP_NAME) CLI 版本..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -tags cli $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-cli ./main.go

# 快速构建 CLI 版本（跳过lint检查，用于测试）
.PHONY: build-cli-fast
build-cli-fast: deps
	@echo "正在快速构建 $(APP_NAME) CLI 版本..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -tags cli $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-cli ./main.go

# 构建 GUI 版本
.PHONY: build-gui
build-gui: lint deps
	@echo "正在构建 $(APP_NAME) GUI 版本..."
	@mkdir -p $(BUILD_DIR)
	wails build -o $(BUILD_DIR)/$(APP_NAME)-gui

# 本地构建（默认构建 CLI 版本）
.PHONY: build
build: build-cli

# 快速构建（跳过lint检查，用于测试，默认构建 CLI 版本）
.PHONY: build-fast
build-fast: build-cli-fast

# 构建当前平台的可执行文件（默认构建 CLI 版本）
.PHONY: build-local
build-local: build

# 交叉编译所有平台
.PHONY: build-all
build-all: clean deps
	@echo "正在进行交叉编译..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		for arch in $(ARCHITECTURES); do \
			if [ "$$platform" = "windows" ] && [ "$$arch" = "arm64" ]; then \
				continue; \
			fi; \
			echo "构建 $$platform/$$arch..."; \
			if [ "$$platform" = "windows" ]; then \
				CGO_ENABLED=0 GOOS=$$platform GOARCH=$$arch $(GOBUILD) $(GOFLAGS) -o $(DIST_DIR)/$(APP_NAME)-$$platform-$$arch.exe ./main.go; \
			else \
				CGO_ENABLED=0 GOOS=$$platform GOARCH=$$arch $(GOBUILD) $(GOFLAGS) -o $(DIST_DIR)/$(APP_NAME)-$$platform-$$arch ./main.go; \
			fi; \
		done; \
	done
	@echo "交叉编译完成，输出目录: $(DIST_DIR)"

# 创建发布包
.PHONY: package
package: build-all
	@echo "正在创建发布包..."
	@cd $(DIST_DIR) && \
	for file in $(APP_NAME)-*; do \
		if [[ "$$file" == *".exe" ]]; then \
			zip "$${file%.exe}.zip" "$$file" ../README.md ../LICENSE 2>/dev/null || zip "$${file%.exe}.zip" "$$file" ../README.md; \
		else \
			tar -czf "$$file.tar.gz" "$$file" ../README.md ../LICENSE 2>/dev/null || tar -czf "$$file.tar.gz" "$$file" ../README.md; \
		fi; \
	done
	@echo "发布包创建完成"

# 安装到本地
.PHONY: install
install: build
	@echo "正在安装 $(APP_NAME)..."
	$(GO) install $(GOFLAGS) ./main.go

# 运行程序
.PHONY: run
run: build
	@echo "正在运行 $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

# 清理构建文件
.PHONY: clean
clean:
	@echo "正在清理构建文件..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f coverage.out coverage.html

# 深度清理（包括依赖缓存）
.PHONY: clean-all
clean-all: clean
	@echo "正在清理依赖缓存..."
	$(GO) clean -modcache

# 更新依赖
.PHONY: update
update:
	@echo "正在更新依赖..."
	$(GO) get -u ./...
	$(GOMOD) tidy

# 检查依赖安全性
.PHONY: security
security:
	@echo "正在检查依赖安全性..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck 未安装，跳过安全检查"; \
		echo "安装命令: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

# 开发环境设置
.PHONY: dev-setup
dev-setup:
	@echo "正在设置开发环境..."
	@echo "安装开发工具..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	@if command -v pre-commit >/dev/null 2>&1; then \
		echo "安装 pre-commit hooks..."; \
		pre-commit install; \
	else \
		echo "pre-commit 未安装，跳过 hooks 安装"; \
		echo "安装命令: pip install pre-commit"; \
	fi
	@echo "开发环境设置完成"

# 显示帮助信息
.PHONY: help
help:
	@echo "可用的 make 命令:"
	@echo "  all         - 运行完整的构建流程 (clean + deps + test + build)"
	@echo "  deps        - 下载并整理依赖"
	@echo "  fmt         - 格式化代码"
	@echo "  vet         - 代码检查"
	@echo "  test        - 运行测试"
	@echo "  coverage    - 生成测试覆盖率报告"
	@echo "  lint        - 代码质量检查（轻量级工具，版本兼容性好）"
	@echo "  lint-light  - 运行轻量级 lint 工具（revive、errcheck）"
	@echo "  lint-check  - 运行完整的代码质量检查脚本"
	@echo "  lint-ci     - 运行CI友好的基础检查"
	@echo "  build       - 构建当前平台的可执行文件"
	@echo "  build-all   - 交叉编译所有平台"
	@echo "  package     - 创建发布包"
	@echo "  install     - 安装到本地"
	@echo "  run         - 构建并运行程序"
	@echo "  clean       - 清理构建文件"
	@echo "  clean-all   - 深度清理（包括依赖缓存）"
	@echo "  update      - 更新依赖"
	@echo "  security    - 检查依赖安全性"
	@echo "  dev-setup   - 设置开发环境"
	@echo "  help        - 显示此帮助信息"

# 显示版本信息
.PHONY: version
version:
	@echo "应用名称: $(APP_NAME)"
	@echo "版本: $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)"
	@echo "Git提交: $(GIT_COMMIT)"