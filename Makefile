.PHONY: all build build-linux build-linux-arm64 build-all check clean dev deps fmt install lint release release-draft run test test-cover uninstall

APP_NAME := camgo-scrcpy
VERSION := $(shell git describe --always --tags --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.buildtime=$(BUILD_TIME)"

# Go commands
GO := go
GOTEST := $(GO) test
GOVET := $(GO) vet
GOBUILD := $(GO) build $(LDFLAGS)
GOMOD := $(GO) mod

# Build for current platform
build:
	$(GOBUILD) -o $(APP_NAME) .

# Build for Linux amd64
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -o $(APP_NAME)-linux-amd64 .

# Build for Linux arm64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -o $(APP_NAME)-linux-arm64 .

# Build all platforms
build-all: build-linux build-linux-arm64
	@echo "Built for all platforms"

# Run the full local verification flow
all: deps fmt lint test build

# Run the same checks we expect before merging
check: fmt lint test

# Run the application
run: build
	./$(APP_NAME)

# Run tests
test:
	$(GOTEST) -v -race ./...

# Run tests with coverage
test-cover:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	$(GOVET) ./...

# Format code
fmt:
	$(GO) fmt ./...

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Clean build artifacts
clean:
	rm -f $(APP_NAME)
	rm -f $(APP_NAME)-*
	rm -f coverage.out coverage.html

# Install globally (requires sudo)
install: build
	install -Dm755 $(APP_NAME) /usr/local/bin/$(APP_NAME)

# Uninstall
uninstall:
	rm -f /usr/local/bin/$(APP_NAME)

# Full development setup
dev: deps fmt lint test build

# Cross-compile for release
release:
	goreleaser build --clean --output ./dist

# Create release with goreleaser
release-draft:
	goreleaser release --clean --draft
