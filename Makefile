# Makefile for codex-mirror-switch

# 变量定义
APP_NAME := codex-mirror
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 相关变量
GO := go
GOFLAGS := -ldflags="-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"
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
all: clean deps test build

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

# 代码质量检查
.PHONY: lint
lint:
	@echo "正在进行代码质量检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint 未安装，跳过代码质量检查"; \
		echo "安装命令: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# 本地构建
.PHONY: build
build: deps
	@echo "正在构建 $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./main.go

# 构建当前平台的可执行文件
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
	@echo "  lint        - 代码质量检查"
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