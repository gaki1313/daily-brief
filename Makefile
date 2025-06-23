# Makefile for Daily Brief Generator
# 提供常用的开发、构建、测试命令

# 应用信息
APP_NAME := daily-brief
VERSION := 1.0.0
BUILD_TIME := $(shell date +%Y-%m-%d\ %H:%M:%S)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 相关变量
GO_VERSION := 1.21
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# 构建标签
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# 默认目标
.DEFAULT_GOAL := help

## 帮助信息
.PHONY: help
help: ## 显示帮助信息
	@echo "Daily Brief Generator - Development Commands"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## 开发相关命令
.PHONY: dev
dev: ## 启动开发服务器（热重载）
	@echo "Starting development server..."
	go run cmd/main.go

.PHONY: build
build: ## 构建应用程序
	@echo "Building $(APP_NAME)..."
	go build $(LDFLAGS) -o bin/$(APP_NAME) cmd/main.go

.PHONY: build-all
build-all: ## 构建所有平台的二进制文件
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 cmd/main.go
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-amd64 cmd/main.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-arm64 cmd/main.go
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-windows-amd64.exe cmd/main.go

## 依赖管理
.PHONY: deps
deps: ## 下载依赖
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

.PHONY: deps-update
deps-update: ## 更新依赖
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

## 测试相关
.PHONY: test
test: ## 运行测试
	@echo "Running tests..."
	go test -v ./...

.PHONY: test-cover
test-cover: ## 运行测试并生成覆盖率报告
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-race
test-race: ## 运行竞态检测测试
	@echo "Running race detection tests..."
	go test -race -v ./...

## 代码质量
.PHONY: lint
lint: ## 运行代码检查
	@echo "Running linter..."
	golangci-lint run

.PHONY: fmt
fmt: ## 格式化代码
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

.PHONY: vet
vet: ## 运行go vet
	@echo "Running go vet..."
	go vet ./...

## 数据库相关
.PHONY: db-migrate
db-migrate: ## 运行数据库迁移
	@echo "Running database migrations..."
	go run cmd/main.go -migrate

.PHONY: db-reset
db-reset: ## 重置数据库
	@echo "Resetting database..."
	rm -f data/daily-brief.db
	@$(MAKE) db-migrate

## 清理
.PHONY: clean
clean: ## 清理构建文件
	@echo "Cleaning up..."
	rm -rf bin/
	rm -rf dist/
	rm -f coverage.out coverage.html
	rm -rf logs/
	go clean

.PHONY: clean-all
clean-all: clean ## 深度清理（包括数据和缓存）
	@echo "Deep cleaning..."
	rm -rf data/
	go clean -cache
	go clean -modcache

## 运行相关
.PHONY: run
run: ## 运行应用程序
	@echo "Running $(APP_NAME)..."
	./bin/$(APP_NAME)

.PHONY: install
install: build ## 安装到系统路径
	@echo "Installing $(APP_NAME)..."
	cp bin/$(APP_NAME) /usr/local/bin/

## Docker相关
.PHONY: docker-build
docker-build: ## 构建Docker镜像
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

.PHONY: docker-run
docker-run: ## 运行Docker容器
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 -v $(PWD)/data:/app/data $(APP_NAME):latest

## 初始化项目
.PHONY: init
init: ## 初始化项目环境
	@echo "Initializing project..."
	mkdir -p bin data logs
	@$(MAKE) deps
	@echo "Project initialized successfully!"

## 完整构建流程
.PHONY: all
all: clean deps fmt vet test build ## 完整的构建流程

## 开发检查
.PHONY: check
check: fmt vet lint test ## 运行所有检查

## 版本信息
.PHONY: version
version: ## 显示版本信息
	@echo "App Name: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "OS/Arch: $(GOOS)/$(GOARCH)" 