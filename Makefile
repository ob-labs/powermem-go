.PHONY: help build test clean install lint fmt examples

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME=powermem-go
GO=go
GOFLAGS=-v
GOTEST=$(GO) test
GOLINT=golangci-lint

help: ## Show help information
	@echo "PowerMem Go SDK - Makefile Help"
	@echo ""
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build project
	@echo "Building project..."
	$(GO) build $(GOFLAGS) ./...

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	$(GOTEST) -v ./tests/...

test-core: ## Run core tests
	@echo "Running core tests..."
	$(GOTEST) -v ./tests/core/...


clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	$(GO) clean
	rm -f coverage.out coverage.html
	rm -rf bin/

install: ## Install dependencies
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy

lint: ## Run linter
	@echo "Running linter..."
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
		echo "Install: brew install golangci-lint (macOS) or visit https://golangci-lint.run/usage/install/"; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "goimports not installed, skipping import formatting"; \
		echo "Install: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

examples: ## Build example programs
	@echo "Building example programs..."
	@mkdir -p bin/examples
	$(GO) build -o bin/examples/basic ./examples/basic
	$(GO) build -o bin/examples/advanced ./examples/advanced
	$(GO) build -o bin/examples/multi_agent ./examples/multi_agent
	@echo "Example programs built to bin/examples/"

check: fmt vet lint test ## Run all checks (format, vet, lint, test)

all: clean install fmt vet build test ## Complete build process
