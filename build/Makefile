# GitLab Task Hook Makefile
#
# 常用命令：
#   make build          # 构建二进制文件
#   make build-all      # 构建所有平台
#   make build-optimized # 构建优化版本
#   make build-static   # 构建静态链接版本
#   make test           # 运行测试
#   make lint           # 代码检查
#   make fmt            # 代码格式化
#   make clean          # 清理构建产物
#   make help           # 显示帮助
#

.PHONY: help build build-all build-optimized build-static test lint fmt clean install version

# 项目信息
PROJECT_NAME := gitlab-task-hook
GO := go
GOFLAGS := -v
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d %H:%M:%S')

# 构建输出目录
OUTPUT_DIR ?= .
BIN_NAME := $(PROJECT_NAME)

# LDFLAGS 用于注入版本信息
LDFLAGS := -X 'main.Version=$(VERSION)' \
           -X 'main.Commit=$(COMMIT)' \
           -X 'main.BuildTime=$(BUILD_TIME)'

# 默认目标
help:
	@echo "GitLab Task Hook 构建工具"
	@echo ""
	@echo "可用命令："
	@echo "  make build              - 构建当前平台的二进制文件"
	@echo "  make build-all          - 构建所有平台 (linux-amd64, linux-arm64)"
	@echo "  make build-optimized    - 构建优化版本 (去掉调试符号)"
	@echo "  make build-static       - 静态链接构建 (无 glibc 依赖)"
	@echo "  make build-docker       - 构建 Docker 镜像（需要 Dockerfile）"
	@echo ""
	@echo "测试和检查："
	@echo "  make test               - 运行所有测试"
	@echo "  make test-verbose       - 详细的测试输出"
	@echo "  make test-coverage      - 生成覆盖率报告"
	@echo "  make lint               - 代码检查"
	@echo "  make fmt                - 代码格式化"
	@echo "  make vet                - go vet 检查"
	@echo ""
	@echo "其他命令："
	@echo "  make clean              - 清理构建产物"
	@echo "  make install            - 安装二进制文件到 \$$GOPATH/bin"
	@echo "  make version            - 显示版本信息"
	@echo "  make help               - 显示此帮助信息"
	@echo ""
	@echo "自定义变量："
	@echo "  OUTPUT_DIR=path         - 输出目录 (默认: 当前目录)"
	@echo "  VERSION=v1.0.0          - 版本号 (默认: dev)"
	@echo ""
	@echo "示例："
	@echo "  make build"
	@echo "  make build OUTPUT_DIR=/tmp/builds"
	@echo "  make build-all VERSION=v1.0.0 OUTPUT_DIR=/tmp/releases"
	@echo "  make build-optimized"
	@echo "  make test && make build"

# 简单构建
build: check-go
	@echo "[构建] 当前平台"
	@$(GO) build $(GOFLAGS) \
		-ldflags "$(LDFLAGS)" \
		-o "$(OUTPUT_DIR)/$(BIN_NAME)" .
	@echo "[成功] 输出: $(OUTPUT_DIR)/$(BIN_NAME)"
	@ls -lh "$(OUTPUT_DIR)/$(BIN_NAME)"

# 多平台构建
build-all: check-go
	@echo "[构建] 所有平台 (linux-amd64, linux-arm64)"
	@mkdir -p "$(OUTPUT_DIR)"
	@for arch in amd64 arm64; do \
		echo "[构建] linux-$$arch"; \
		GOOS=linux GOARCH=$$arch $(GO) build $(GOFLAGS) \
			-ldflags "$(LDFLAGS)" \
			-o "$(OUTPUT_DIR)/$(BIN_NAME)-linux-$$arch" . || exit 1; \
		ls -lh "$(OUTPUT_DIR)/$(BIN_NAME)-linux-$$arch"; \
	done
	@echo "[成功] 所有平台构建完成"

# 优化版本 (去掉调试符号)
build-optimized: check-go
	@echo "[构建] 优化版本 (移除调试符号)"
	@$(GO) build $(GOFLAGS) \
		-ldflags "$(LDFLAGS) -s -w" \
		-o "$(OUTPUT_DIR)/$(BIN_NAME)" .
	@echo "[成功] 输出: $(OUTPUT_DIR)/$(BIN_NAME)"
	@ls -lh "$(OUTPUT_DIR)/$(BIN_NAME)"

# 静态链接 (无 glibc 依赖)
build-static: check-go
	@echo "[构建] 静态链接版本"
	@CGO_ENABLED=0 $(GO) build $(GOFLAGS) \
		-ldflags "$(LDFLAGS) -s -w" \
		-o "$(OUTPUT_DIR)/$(BIN_NAME)" .
	@echo "[成功] 输出: $(OUTPUT_DIR)/$(BIN_NAME)"
	@ls -lh "$(OUTPUT_DIR)/$(BIN_NAME)"

# Docker 构建 (需要 Dockerfile)
build-docker: check-docker
	@echo "[构建] Docker 镜像"
	docker build -t $(PROJECT_NAME):$(VERSION) .
	@echo "[成功] Docker 镜像: $(PROJECT_NAME):$(VERSION)"
	docker images | grep $(PROJECT_NAME)

# 运行测试
test: check-go
	@echo "[测试] 运行单元测试"
	@$(GO) test -v ./...

# 详细的测试输出
test-verbose: check-go
	@echo "[测试] 运行单元测试（详细输出）"
	@$(GO) test -v -race -coverprofile=coverage.out ./...
	@$(GO) tool cover -func=coverage.out

# 生成覆盖率报告
test-coverage: check-go
	@echo "[测试] 生成覆盖率报告"
	@$(GO) test -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "[成功] 覆盖率报告: coverage.html"

# 代码检查 (go vet)
vet: check-go
	@echo "[检查] go vet"
	@$(GO) vet ./...

# 代码格式化
fmt: check-go
	@echo "[格式化] Go 代码"
	@$(GO) fmt ./...
	@echo "[成功] 代码格式化完成"

# 代码检查 (golangci-lint)
lint: check-golangci-lint
	@echo "[检查] golangci-lint"
	@golangci-lint run

# 清理构建产物
clean:
	@echo "[清理] 构建产物"
	@rm -f "$(OUTPUT_DIR)/$(BIN_NAME)"
	@rm -f "$(OUTPUT_DIR)/$(BIN_NAME)-linux-amd64"
	@rm -f "$(OUTPUT_DIR)/$(BIN_NAME)-linux-arm64"
	@rm -f coverage.out coverage.html
	@echo "[成功] 清理完成"

# 安装到 GOPATH/bin
install: build
	@echo "[安装] 复制到 \$$GOPATH/bin"
	@mkdir -p "$(GOPATH)/bin"
	@cp "$(OUTPUT_DIR)/$(BIN_NAME)" "$(GOPATH)/bin/"
	@echo "[成功] 已安装到: $(GOPATH)/bin/$(BIN_NAME)"

# 显示版本信息
version:
	@echo "GitLab Task Hook 版本信息"
	@echo "  VERSION:     $(VERSION)"
	@echo "  COMMIT:      $(COMMIT)"
	@echo "  BUILD_TIME:  $(BUILD_TIME)"

# 前置条件检查
check-go:
	@command -v go >/dev/null 2>&1 || { echo "[错误] Go 未安装"; exit 1; }

check-docker:
	@command -v docker >/dev/null 2>&1 || { echo "[错误] Docker 未安装"; exit 1; }

check-golangci-lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "[错误] golangci-lint 未安装"; \
		echo "请执行: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	}

# 完整的 CI 流程
ci: test lint build
	@echo "[成功] CI 流程完成"

.DEFAULT_GOAL := help
