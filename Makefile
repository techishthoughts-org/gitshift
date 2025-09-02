# GitPersona Makefile

.PHONY: help build clean test install docker dev lint format check

# Variables
BINARY_NAME=gitpersona
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"

# Default target
help: ## Show this help
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Build targets
build: ## Build the application
	@echo "Building ${BINARY_NAME} v${VERSION}"
	go build ${LDFLAGS} -o ${BINARY_NAME} .

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-amd64.exe .

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f ${BINARY_NAME}
	rm -f ${BINARY_NAME}-*
	go clean

# Development targets
dev: ## Run in development mode
	go run ${LDFLAGS} . $(ARGS)

install: build ## Install the binary to $GOPATH/bin
	@echo "Installing ${BINARY_NAME} to ${GOPATH}/bin"
	mv ${BINARY_NAME} ${GOPATH}/bin/

# Testing targets
test: ## Run tests
	@echo "Running tests..."
	go test -v -race -timeout 30s ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -race -timeout 30s -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

benchmark: ## Run benchmark tests
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Code quality targets
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...

format: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

check: format vet lint test ## Run all checks

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t ${BINARY_NAME}:${VERSION} .
	docker tag ${BINARY_NAME}:${VERSION} ${BINARY_NAME}:latest

docker-run: docker-build ## Run Docker container
	docker run -it --rm \
		-v ~/.config/gh-switcher:/home/appuser/.config/gh-switcher \
		-v ~/.ssh:/home/appuser/.ssh:ro \
		${BINARY_NAME}:latest

docker-dev: ## Start development environment with Docker Compose
	docker-compose up -d dev
	docker-compose exec dev sh

docker-test: ## Run tests in Docker
	docker-compose run --rm test

docker-clean: ## Clean Docker images and containers
	docker-compose down -v
	docker rmi ${BINARY_NAME}:${VERSION} ${BINARY_NAME}:latest || true

# Release targets
release: check build-all ## Prepare release
	@echo "Creating release ${VERSION}..."
	mkdir -p dist
	mv ${BINARY_NAME}-* dist/
	@echo "Release binaries available in dist/"

# Documentation targets
docs: ## Generate documentation
	@echo "Generating documentation..."
	go doc -all . > DOCS.md

# Utility targets
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

update-deps: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

mod-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	go mod verify

security: ## Run security checks
	@echo "Running security checks..."
	gosec ./...

# Demo targets
demo-setup: build ## Set up demo accounts
	@echo "Setting up demo accounts..."
	./$(BINARY_NAME) add work --name "John Work" --email "john@work.com" --description "Work account" || true
	./$(BINARY_NAME) add personal --name "John Personal" --email "john@personal.com" --description "Personal account" || true
	./$(BINARY_NAME) list

demo-clean: ## Clean demo configuration
	@echo "Cleaning demo configuration..."
	rm -rf ~/.config/gh-switcher/config.yaml
	rm -f .gh-switcher.yaml

# Git hooks
setup-hooks: ## Set up git hooks
	@echo "Setting up git hooks..."
	cp scripts/hooks/pre-commit .git/hooks/
	chmod +x .git/hooks/pre-commit

# Version info
version: ## Show version information
	@echo "Version: ${VERSION}"
	@echo "Commit: ${COMMIT}"
	@echo "Build Time: ${BUILD_TIME}"

info: version ## Show build information
	@echo "Go Version: $(shell go version)"
	@echo "Platform: $(shell go env GOOS)/$(shell go env GOARCH)"

# Quick commands for development workflow
quick: format test build ## Quick development cycle: format, test, build

all: clean check build-all ## Complete build process

# Default target when no arguments provided
.DEFAULT_GOAL := help
