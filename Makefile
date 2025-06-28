# ==============================================================================
# Payer Status IO - High-Performance WebSocket Health Monitor
#
# Available targets:
#   - build           Build the application
#   - test            Run tests with coverage
#   - docker-build    Build Docker image
#   - release         Create release artifacts
#   - help            Show this help message
# ==============================================================================

# Project metadata
PROJECT_NAME := payer-status-io
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo v0.0.0)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

# Build configuration
GO := go
GOFLAGS := -mod=mod
GOPRIVATE := github.com/copyleftdev/*

# Build flags
LDFLAGS := -ldflags '\
    -X main.version=$(VERSION) \
    -X main.commit=$(GIT_COMMIT) \
    -X main.date=$(BUILD_TIME) \
    -w -s -extldflags "-static"'

# Directories
BIN_DIR := bin
DIST_DIR := dist
COVERAGE_DIR := coverage
MIGRATIONS_DIR := migrations
WEB_DIR := web

# File names
BINARY_NAME := $(PROJECT_NAME)
DOCKER_IMAGE := ghcr.io/copyleftdev/$(PROJECT_NAME)
DOCKER_TAG := $(VERSION)

# Tools
DOCKER := docker
DOCKER_COMPOSE := docker-compose
GOLANGCI_LINT := golangci-lint
GORELEASER := goreleaser
MIGRATE := migrate
PROTOC := protoc
RICHGO := richgo
GOTESTSUM := gotest.tools/gotestsum

# Go parameters
GO_PACKAGES = $(shell $(GO) list ./... | grep -v /vendor/)
GO_FILES = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

# Build tags
BUILD_TAGS = netgo

# Default target
.PHONY: default
default: help

# Print the current version
.PHONY: version
version:
	@echo "$(VERSION)"

# Print build information
.PHONY: info
info:
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Git Branch: $(GIT_BRANCH)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(shell $(GO) version)"
	@echo "OS/Arch:    $(shell uname -s)/$(shell uname -m)"

# ==============================================================================
# Build Targets
# ==============================================================================

# Build the application
.PHONY: build
build: $(BIN_DIR)/$(BINARY_NAME) ## Build the application

$(BIN_DIR)/$(BINARY_NAME): $(GO_FILES)
	@echo "🚀 Building $(PROJECT_NAME) $(VERSION)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -v -tags "$(BUILD_TAGS)" $(LDFLAGS) -o $@ ./cmd/server

# Build for specific platforms
.PHONY: build-linux build-darwin build-windows build-all

build-linux: ## Build for Linux (amd64)
	@$(MAKE) build GOOS=linux GOARCH=amd64 BIN_SUFFIX=-linux-amd64

build-darwin: ## Build for macOS (arm64)
	@$(MAKE) build GOOS=darwin GOARCH=arm64 BIN_SUFFIX=-darwin-arm64

build-windows: ## Build for Windows (amd64)
	@$(MAKE) build GOOS=windows GOARCH=amd64 BIN_SUFFIX=-windows-amd64.exe

build-all: clean ## Build for all platforms
	@echo "🔨 Building for all platforms..."
	@$(MAKE) build-linux
	@$(MAKE) build-darwin
	@$(MAKE) build-windows

# Install the application
.PHONY: install
install: build ## Install the application
	@echo "📦 Installing $(PROJECT_NAME)..."
	@cp $(BIN_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

# Build with debug information
.PHONY: debug
debug: LDFLAGS = -ldflags "-X main.version=$(VERSION)-debug -X main.commit=$(GIT_COMMIT) -X main.date=$(BUILD_TIME)"
debug: build ## Build with debug information

# Clean build artifacts
.PHONY: clean
clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	@rm -rf $(BIN_DIR) $(DIST_DIR) $(COVERAGE_DIR)
	@find . -name "*.test" -delete
	@find . -name "*.out" -delete
	@find . -name "coverage.*" -delete

# Check for required tools
.PHONY: check-tools
check-tools: ## Check for required tools
	@command -v $(GOLANGCI_LINT) >/dev/null || (echo "❌ $(GOLANGCI_LINT) not found. Run 'make deps' to install." && exit 1)
	@command -v $(GORELEASER) >/dev/null || (echo "❌ $(GORELEASER) not found. Run 'make deps' to install." && exit 1)
	@command -v $(DOCKER) >/dev/null || (echo "❌ $(DOCKER) not found. Please install Docker." && exit 1)
	@echo "✅ All required tools are installed"

# ==============================================================================
# Run Targets
# ==============================================================================

# Run the application
.PHONY: run
run: build ## Run the application
	@echo "🚀 Starting $(PROJECT_NAME) $(VERSION)..."
	@$(BIN_DIR)/$(BINARY_NAME) --config ./docs/payer_status.yaml

# Run with hot-reloading (requires air)
.PHONY: dev
run-dev: ## Run with hot-reloading (requires air)
	@echo "🔥 Starting development server with hot-reloading..."
	air -c .air.toml

# Run with race detector
.PHONY: run-race
run-race: ## Run with race detector
	@echo "🏃 Running with race detector..."
	$(GO) run -race ./cmd/server --config ./docs/payer_status.yaml

# Run with pprof (http://localhost:6060/debug/pprof/)
.PHONY: pprof
pprof: ## Run with pprof enabled
	@echo "📊 Starting with pprof on :6060..."
	$(GO) run -tags=pprof ./cmd/server --config ./docs/payer_status.yaml

# Generate sample configuration
.PHONY: config
docs/payer_status.yaml: ## Generate sample configuration
	@echo "⚙️  Generating sample configuration..."
	@mkdir -p docs
	@echo '# Payer Status Configuration' > $@
	@echo '# Generated: $(shell date)' >> $@
	@echo '---' >> $@
	@echo 'server:' >> $@
	@echo '  port: 8080' >> $@
	@echo '  metrics_port: 9090' >> $@
	@echo '  log_level: "info"' >> $@
	@echo '  environment: "development"' >> $@

# Install development tools
.PHONY: tools
tools: ## Install development tools
	@echo "🔧 Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/goreleaser/goreleaser@latest
	@go install github.com/kyoh86/richgo@latest
	@go install gotest.tools/gotestsum@latest
	@go install github.com/cosmtrek/air@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install github.com/vektra/mockery/v2@latest

# Test targets
.PHONY: test
TEST_PACKAGES ?= ./...
test: ## Run tests with coverage
	@echo "🧪 Running tests..."
	@mkdir -p $(COVERAGE_DIR)
	@$(GO) test -v -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic $(TEST_PACKAGES)
	@$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "✅ Test coverage report: file://$(shell pwd)/$(COVERAGE_DIR)/coverage.html"

# Run tests with race detector
.PHONY: test-race
test-race: ## Run tests with race detector
	@echo "🏃 Running tests with race detector..."
	@$(GO) test -race -v $(TEST_PACKAGES)

# Run tests with rich output
.PHONY: test-rich
test-rich: ## Run tests with rich output (requires richgo)
	@$(GO) install github.com/kyoh86/richgo@latest
	@richgo test -v $(TEST_PACKAGES)

# Run tests with verbose output
.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	@$(GO) test -v -cover -coverprofile=$(COVERAGE_DIR)/coverage.out $(TEST_PACKAGES)

# Run benchmark tests
.PHONY: benchmark
benchmark: ## Run benchmark tests
	@echo "🏋️  Running benchmarks..."
	@$(GO) test -run=^$$ -bench=. -benchmem -count=5 $(TEST_PACKAGES) | tee benchmark.txt

# Run integration tests
.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "🧪 Running integration tests..."
	@$(GO) test -tags=integration -v $(TEST_PACKAGES)

# Run tests with coverage and open in browser
.PHONY: coverage
coverage: test ## Run tests and open coverage in browser
	@xdg-open $(COVERAGE_DIR)/coverage.html 2>/dev/null || open $(COVERAGE_DIR)/coverage.html 2>/dev/null || echo "Open $(COVERAGE_DIR)/coverage.html in your browser"

# Check for race conditions
.PHONY: race
race: ## Run tests with race detector
	@echo "🔍 Checking for race conditions..."
	@$(GO) test -race -short $(TEST_PACKAGES)

# ==============================================================================
# Lint & Static Analysis
# ==============================================================================

# Run all linters
.PHONY: lint
lint: lint-go lint-yaml lint-dockerfile ## Run all linters

# Run Go linters
.PHONY: lint-go
lint-go: check-tools ## Run Go linters
	@echo "🔍 Running Go linters..."
	@$(GOLANGCI_LINT) run --timeout 5m --enable-all

# Fix linting issues
.PHONY: lint-fix
lint-fix: check-tools ## Fix linting issues
	@echo "🔧 Fixing linting issues..."
	@$(GOLANGCI_LINT) run --fix --timeout 5m

# Lint YAML files
.PHONY: lint-yaml
lint-yaml: ## Lint YAML files
	@echo "🔍 Linting YAML files..."
	@command -v yamllint >/dev/null || (echo "❌ yamllint not found. Install with 'pip install yamllint'" && exit 1)
	@yamllint .

# Lint Dockerfile
.PHONY: lint-dockerfile
lint-dockerfile: ## Lint Dockerfile
	@echo "🔍 Linting Dockerfile..."
	@command -v hadolint >/dev/null || (echo "❌ hadolint not found. Install with 'brew install hadolint'" && exit 1)
	@hadolint Dockerfile

# Run security scanner
.PHONY: security
security: ## Run security scanner
	@echo "🔒 Running security scan..."
	@$(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	@gosec -exclude=G104 ./...

# Run static analysis
.PHONY: analyze
analyze: ## Run static analysis
	@echo "🔍 Running static analysis..."
	@$(GO) vet ./...
	@$(GO) mod verify
	@$(GO) list -json -m all | jq -r '. | select(.Indirect == false) | .Path + "@" + .Version'

# ==============================================================================
# Clean Targets
# ==============================================================================

# Clean build artifacts
.PHONY: clean
clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(DIST_DIR) $(COVERAGE_DIR)
	@find . -name "*.test" -delete
	@find . -name "*.out" -delete
	@find . -name "coverage.*" -delete

# Clean Docker artifacts
.PHONY: clean-docker
clean-docker: ## Clean Docker artifacts
	@echo "🧹 Cleaning Docker artifacts..."
	@$(DOCKER) system prune -f
	@$(DOCKER) images -q $(DOCKER_IMAGE) | xargs -r $(DOCKER) rmi -f

# Clean everything
.PHONY: clean-all
clean-all: clean clean-docker ## Clean everything
	@echo "🧹 Cleaning everything..."
	@$(GO) clean -cache -modcache -i -r
	@rm -rf vendor/

# ==============================================================================
# Docker Targets
# ==============================================================================

# Build Docker image
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@$(DOCKER) build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		.

# Run Docker container
.PHONY: docker-run
docker-run: docker-build ## Run Docker container
	@echo "🚀 Starting Docker container..."
	@$(DOCKER) run -it --rm \
		-p 8080:8080 \
		-p 9090:9090 \
		-v $(PWD)/docs:/app/docs:ro \
		-e CONFIG_PATH=/app/docs/payer_status.yaml \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

# Push Docker image
.PHONY: docker-push
docker-push: docker-build ## Push Docker image to registry
	@echo "📤 Pushing Docker image..."
	@$(DOCKER) push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@$(DOCKER) push $(DOCKER_IMAGE):latest

# Scan Docker image for vulnerabilities
.PHONY: docker-scan
docker-scan: docker-build ## Scan Docker image for vulnerabilities
	@echo "🔍 Scanning Docker image for vulnerabilities..."
	@$(DOCKER) scan $(DOCKER_IMAGE):$(DOCKER_TAG)

# ==============================================================================
# Docker Compose Targets
# ==============================================================================

# Start services with Docker Compose
.PHONY: compose-up
compose-up: ## Start services with Docker Compose
	@echo "🚀 Starting services with Docker Compose..."
	@$(DOCKER_COMPOSE) up -d --build

# Stop services
.PHONY: compose-down
compose-down: ## Stop services
	@echo "🛑 Stopping services..."
	@$(DOCKER_COMPOSE) down --remove-orphans

# View service logs
.PHONY: compose-logs
compose-logs: ## View service logs
	@$(DOCKER_COMPOSE) logs -f

# Rebuild and restart services
.PHONY: compose-restart
compose-restart: compose-down compose-up ## Rebuild and restart services

# Show service status
.PHONY: compose-ps
compose-ps: ## Show service status
	@$(DOCKER_COMPOSE) ps

# Run tests in Docker
.PHONY: compose-test
compose-test: ## Run tests in Docker
	@echo "🧪 Running tests in Docker..."
	@$(DOCKER_COMPOSE) run --rm app make test

# ==============================================================================
# Code Generation
# ==============================================================================

# Generate code
.PHONY: generate
generate: ## Generate code (protobuf, mocks, etc.)
	@echo "⚙️  Generating code..."
	@$(GO) generate ./...

# Generate mocks
.PHONY: generate-mocks
generate-mocks: ## Generate mocks for interfaces
	@echo "🎭 Generating mocks..."
	@$(GO) install github.com/vektra/mockery/v2@latest
	@mockery --all --dir ./internal --output ./internal/mocks --case underscore

# Generate protobuf files
.PHONY: generate-protos
generate-protos: ## Generate protobuf files
	@echo "📦 Generating protobuf files..."
	@command -v protoc >/dev/null || (echo "❌ protoc not found" && exit 1)
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		./api/v1/*.proto

# ==============================================================================
# Dependencies
# ==============================================================================

# Install dependencies
.PHONY: deps
deps: ## Install dependencies
	@echo "📦 Installing dependencies..."
	@$(GO) mod download
	@$(GO) mod verify

# Update dependencies
.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "🔄 Updating dependencies..."
	@$(GO) get -u ./...
	@$(GO) mod tidy

# Check for outdated dependencies
.PHONY: deps-outdated
deps-outdated: ## Check for outdated dependencies
	@echo "🔍 Checking for outdated dependencies..."
	@$(GO) list -u -m -json all | jq -r 'select(.Update) | "\(.Path) \(.Version) -> \(.Update.Version)"'

# ==============================================================================
# Release
# ==============================================================================

# Create a release
.PHONY: release
release: check-tools test lint security ## Create a release
	@echo "🚀 Creating release $(VERSION)..."
	@mkdir -p $(DIST_DIR)
	@$(GORELEASER) release --rm-dist --skip-publish --snapshot

# Create a local snapshot release
.PHONY: snapshot
snapshot: check-tools ## Create a local snapshot release
	@echo "📸 Creating snapshot release..."
	@$(GORELEASER) release --rm-dist --skip-publish --snapshot

# Bump version
.PHONY: bump-version
bump-version: ## Bump version (use: make bump-version VERSION=v1.0.0)
	@[ "$(VERSION)" ] || ( echo "VERSION is not set. Usage: make bump-version VERSION=v1.0.0"; exit 1 )
	@echo "🆙 Bumping version to $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@git push origin $(VERSION)

# ==============================================================================
# Documentation
# ==============================================================================

# Generate documentation
.PHONY: docs
docs: ## Generate documentation
	@echo "📚 Generating documentation..."
	@$(GO) run ./cmd/server --help

# Serve documentation
.PHONY: docs-serve
docs-serve: ## Serve documentation
	@echo "🌐 Serving documentation at http://localhost:3000"
	@docker run --rm -it -p 3000:3000 -v $(PWD)/docs:/docs squidfunk/mkdocs-material

# ==============================================================================
# Help
# ==============================================================================

# Help documentation
.PHONY: help
help: ## Display this help message
	@echo "\n\033[1mPayer Status IO - WebSocket Health Monitor\033[0m\n"
	@echo "Available targets:\n"
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""

# Show environment info
.PHONY: env
env: ## Show environment information
	@echo "\033[1mEnvironment Information\033[0m"
	@echo "------------------------"
	@echo "Project:     $(PROJECT_NAME)"
	@echo "Version:     $(VERSION)"
	@echo "Git Commit:  $(GIT_COMMIT)"
	@echo "Git Branch:  $(GIT_BRANCH)"
	@echo "Build Time:  $(BUILD_TIME)"
	@echo "Go Version:  $(shell go version)"
	@echo "OS/Arch:     $(shell uname -s)/$(shell uname -m)"

# Default target
.DEFAULT_GOAL := help
