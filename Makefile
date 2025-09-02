# GitPersona Makefile

# Variables
BINARY_NAME=gitpersona
DOCKER_IMAGE=gitpersona
VERSION?=v0.1.0
COMMIT?=$(shell git rev-parse --short HEAD)
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go build flags
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME} -w -s"

.PHONY: help build test clean docker docker-push install uninstall demo demo-clean ci-test ci-lint ci-security

help: ## Show this help message
	@echo "GitPersona - Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "🏗️  Building GitPersona..."
	go build ${LDFLAGS} -o ${BINARY_NAME} .
	@echo "✅ Build complete: ${BINARY_NAME}"

test: ## Run tests
	@echo "🧪 Running tests..."
	go test -v -timeout 5m ./...

test-coverage: ## Run tests with coverage
	@echo "🧪 Running tests with coverage..."
	go test -v -timeout 5m -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

test-bench: ## Run benchmark tests
	@echo "⚡ Running benchmark tests..."
	go test -bench=. -benchmem ./internal/models

clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	rm -f ${BINARY_NAME}
	rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

docker: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t ${DOCKER_IMAGE}:${VERSION} .
	docker tag ${DOCKER_IMAGE}:${VERSION} ${DOCKER_IMAGE}:latest
	@echo "✅ Docker image built: ${DOCKER_IMAGE}:${VERSION}"

docker-push: ## Push Docker image
	@echo "📤 Pushing Docker image..."
	docker push ${DOCKER_IMAGE}:${VERSION}
	docker push ${DOCKER_IMAGE}:latest
	@echo "✅ Docker image pushed"

install: build ## Install to system
	@echo "📦 Installing GitPersona..."
	sudo cp ${BINARY_NAME} /usr/local/bin/
	@echo "✅ Installed to /usr/local/bin/${BINARY_NAME}"

uninstall: ## Uninstall from system
	@echo "🗑️  Uninstalling GitPersona..."
	sudo rm -f /usr/local/bin/${BINARY_NAME}
	@echo "✅ Uninstalled from /usr/local/bin/${BINARY_NAME}"

demo: ## Run demo environment
	@echo "🎭 Starting GitPersona demo..."
	docker-compose up -d
	@echo "✅ Demo environment started"
	@echo "🌐 Access at: http://localhost:8080"

demo-clean: ## Clean demo environment
	@echo "🧹 Cleaning demo environment..."
	docker-compose down -v
	@echo "✅ Demo environment cleaned"

# CI Testing with act
ci-test: ## Test CI workflow locally with act
	@echo "🧪 Testing CI workflow locally..."
	@if ! command -v act &> /dev/null; then \
		echo "❌ act not found. Install from: https://github.com/nektos/act"; \
		exit 1; \
	fi
	act push --workflows .github/workflows/ci.yml

ci-lint: ## Test linting workflow locally
	@echo "🔍 Testing linting workflow locally..."
	@if ! command -v act &> /dev/null; then \
		echo "❌ act not found. Install from: https://github.com/nektos/act"; \
		exit 1; \
	fi
	act pull_request --workflows .github/workflows/ci.yml --job quality

ci-security: ## Test security workflow locally
	@echo "🔒 Testing security workflow locally..."
	@if ! command -v act &> /dev/null; then \
		echo "❌ act not found. Install from: https://github.com/nektos/act"; \
		exit 1; \
	fi
	act pull_request --workflows .github/workflows/ci.yml --job security

# Development helpers
fmt: ## Format Go code
	@echo "🎨 Formatting Go code..."
	gofmt -s -w .
	@echo "✅ Code formatted"

vet: ## Run go vet
	@echo "🔧 Running go vet..."
	go vet ./...
	@echo "✅ Go vet complete"

lint: ## Run golangci-lint
	@echo "🔍 Running golangci-lint..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "❌ golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi
	golangci-lint run --timeout=5m

deps: ## Download and verify dependencies
	@echo "📦 Downloading dependencies..."
	go mod download
	go mod verify
	@echo "✅ Dependencies ready"

# Quick development workflow
dev: deps fmt vet test build ## Full development workflow
	@echo "🚀 Development workflow complete!"

# Release helpers
release: clean test build ## Prepare release build
	@echo "🎉 Release build ready: ${BINARY_NAME}"
	@echo "Version: ${VERSION}"
	@echo "Commit: ${COMMIT}"
	@echo "Build Time: ${BUILD_TIME}"

# Pre-commit validation
pre-commit: ## Run pre-commit hooks on all files
	pre-commit run --all-files

# Pre-commit validation (staged files only)
pre-commit-staged: ## Run pre-commit hooks on staged files only
	pre-commit run

# Install pre-commit hooks
install-hooks: ## Install pre-commit hooks
	pre-commit install
