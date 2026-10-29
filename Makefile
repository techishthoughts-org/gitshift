# gitshift - SSH-first GitHub and GitLab identity management
# Manage multiple provider identities with Git, SSH, GPG, and GitHub CLI integration

.PHONY: all build test clean lint fmt vet install deps dev help demo docs release-check

# Build variables
BINARY_NAME=gitshift
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X github.com/techishthoughts/gitshift/cmd.version=$(VERSION) -X github.com/techishthoughts/gitshift/cmd.buildTime=$(BUILD_TIME)"

# Default target
all: build

## Build the binary
build:
	@echo "🔨 Building $(BINARY_NAME) $(VERSION)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "✅ Build complete: ./$(BINARY_NAME)"

## Install the binary to GOPATH/bin
install: build
	@echo "📦 Installing $(BINARY_NAME)..."
	@mkdir -p $${GOPATH:-$$HOME/go}/bin
	@cp $(BINARY_NAME) $${GOPATH:-$$HOME/go}/bin/
	@chmod +x $${GOPATH:-$$HOME/go}/bin/$(BINARY_NAME)
	@echo "✅ Installed: $${GOPATH:-$$HOME/go}/bin/$(BINARY_NAME)"

## Run all tests
test:
	@echo "🧪 Running tests..."
	@go test -v ./...
	@echo "✅ Tests passed"

## Format and clean code
fmt:
	@echo "🎨 Formatting code..."
	@go fmt ./...
	@echo "✅ Code formatted"

## Run go vet
vet:
	@echo "🔍 Running go vet..."
	@go vet ./...
	@echo "✅ Vet passed"

## Run linting (if golangci-lint available)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "🔍 Running linter..."; \
		golangci-lint run; \
		echo "✅ Linting passed"; \
	else \
		echo "⚠️  golangci-lint not found, skipping"; \
	fi

## Download and tidy dependencies
deps:
	@echo "📦 Managing dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies updated"

## Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@go clean
	@rm -rf $(BINARY_NAME) coverage.out dist
	@echo "✅ Clean complete"

## Development workflow: build and test
dev: deps fmt vet build test
	@echo "🚀 Development build complete!"

## Demo - show gitshift in action
demo: build
	@echo "🎭 gitshift Demo"
	@echo "=================="
	@echo ""
	@echo "📋 Current accounts:"
	@./$(BINARY_NAME) list || echo "No accounts configured yet"
	@echo ""
	@echo "🔍 SSH key discovery:"
	@./$(BINARY_NAME) discover || true
	@echo ""
	@echo "🔧 Available commands:"
	@./$(BINARY_NAME) --help

## Show help
help:
	@echo "🎭 gitshift - GitHub and GitLab Identity Management"
	@echo ""
	@echo "Available make targets:"
	@echo "  build      Build the gitshift binary"
	@echo "  install    Install gitshift to GOPATH/bin"
	@echo "  test       Run tests"
	@echo "  fmt        Format Go code"
	@echo "  vet        Run go vet"
	@echo "  lint       Run golangci-lint (if available)"
	@echo "  deps       Download and tidy dependencies"
	@echo "  clean      Clean build artifacts"
	@echo "  dev        Full development workflow"
	@echo "  demo       Show gitshift in action"
	@echo "  docs       Regenerate docs/reference/commands"
	@echo "  release-check  Build release archives locally (no publish)"
	@echo "  help       Show this help"
	@echo ""
	@echo "🚀 Quick start:"
	@echo "  make build    # Build the binary"
	@echo "  ./gitshift discover  # Find existing SSH keys"
	@echo "  ./gitshift ssh-keygen myaccount --email me@example.com"

## Regenerate the command reference in docs/reference/commands
docs:
	@go run ./tools/gendocs
	@echo "✅ Command reference regenerated"

## Build every release archive locally with GoReleaser (nothing is published)
release-check:
	@command -v goreleaser >/dev/null 2>&1 || { echo "❌ Install GoReleaser v2: https://goreleaser.com/install/"; exit 1; }
	@goreleaser check
	@HOMEBREW_TAP_TOKEN=unused goreleaser release --snapshot --clean --skip=publish
	@echo "✅ Snapshot archives, checksums and cask written to dist/"
