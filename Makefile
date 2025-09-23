# GitPersona Makefile

.PHONY: build test clean lint fmt vet install dev deps coverage benchmark help

# Build variables
BINARY_NAME=gitpersona
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target
all: build

## Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

## Install the binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) .

## Run tests
test:
	@echo "Running tests..."
	go test -race ./...

## Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

## Run linting
lint:
	@echo "Running linter..."
	golangci-lint run

## Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

## Development build with debug info
dev:
	@echo "Building development version..."
	go build -gcflags="all=-N -l" -o $(BINARY_NAME)-dev .

## Clean build artifacts
clean:
	@echo "Cleaning..."
	go clean
	rm -f $(BINARY_NAME) $(BINARY_NAME)-dev coverage.out coverage.html

## Run full CI pipeline locally
ci: deps fmt vet lint test coverage

## Show help
help:
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Security scanning
.PHONY: security
security:
	@echo "Running security scan..."
	gosec ./...

# Docker targets
.PHONY: docker-build docker-run
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .

docker-run:
	@echo "Running Docker container..."
	docker run --rm -it $(BINARY_NAME):$(VERSION)

# Release targets
.PHONY: release-build
release-build:
	@echo "Building release binaries..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .

# Development helpers
.PHONY: watch
watch:
	@echo "Watching for changes..."
	while true; do \
		inotifywait -r -e modify --include="\.go$$" .; \
		make build; \
	done