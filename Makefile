# GitPersona Makefile

# Variables
BINARY_NAME=gitpersona
VERSION?=v0.1.0

# Go build flags
LDFLAGS=-ldflags "-X main.Version=${VERSION} -w -s"

.PHONY: help build test clean install uninstall dev release

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

clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	rm -f ${BINARY_NAME}
	rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

install: build ## Install to system
	@echo "📦 Installing GitPersona..."
	cp ${BINARY_NAME} ~/.local/bin/
	@echo "✅ Installed to ~/.local/bin/${BINARY_NAME}"

uninstall: ## Uninstall from system
	@echo "🗑️  Uninstalling GitPersona..."
	rm -f ~/.local/bin/${BINARY_NAME}
	@echo "✅ Uninstalled from ~/.local/bin/${BINARY_NAME}"

# Development helpers
fmt: ## Format Go code
	@echo "🎨 Formatting Go code..."
	gofmt -s -w .
	@echo "✅ Code formatted"

vet: ## Run go vet
	@echo "🔧 Running go vet..."
	go vet ./...
	@echo "✅ Go vet complete"

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
