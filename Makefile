# ccmgr-ultra Makefile

# Variables
BINARY_NAME := ccmgr-ultra
MAIN_PATH := ./cmd/ccmgr-ultra
BUILD_DIR := build
INSTALL_DIR := $(HOME)/.local/bin

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# Build flags
LDFLAGS := -ldflags="-s -w"
BUILD_FLAGS := -trimpath

# Version info (can be overridden)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Full build flags with version info
FULL_LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION) -X main.date=$(BUILD_TIME) -X main.commit=$(GIT_COMMIT)"

.PHONY: all build clean test install uninstall run fmt vet deps tidy help test-env test-env-clean docs-serve docs-build docs-clean docs-install docs-check docs-deploy

# Default target
all: help

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) $(FULL_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-fast: ## Quick build without optimization
	@echo "Quick building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: ./$(BINARY_NAME)"

install: build ## Install the binary to ~/.local/bin
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo "Make sure $(INSTALL_DIR) is in your PATH"

uninstall: ## Remove the installed binary
	@echo "Removing $(BINARY_NAME) from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstalled"

run: build ## Run the application
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -v -cover -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: brew install golangci-lint"; \
	fi

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOGET) -v ./...

tidy: ## Tidy module dependencies
	@echo "Tidying module dependencies..."
	$(GOMOD) tidy

update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

dev: ## Run with hot reload (requires air)
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to regular run..."; \
		$(MAKE) run; \
	fi

check: fmt vet lint ## Run fmt, vet, and lint

test-env: ## Create a temporary git repo for testing
	@echo "Creating test environment..."
	@mkdir -p .testdirs
	@TESTDIR=".testdirs/test-$$(date +%Y%m%d-%H%M%S)"; \
	mkdir -p "$$TESTDIR" && \
	cd "$$TESTDIR" && \
	git init -q && \
	echo "test README" > README.md && \
	git add README.md && \
	git commit -q -m "Initial commit" && \
	echo "" && \
	echo "Test environment created at: $$TESTDIR" && \
	echo "Entering test environment shell (type 'exit' to return)..." && \
	echo "" && \
	exec $${SHELL:-bash} -i

test-env-clean: ## Clean up all test environments
	@echo "Cleaning test environments..."
	@if [ -d .testdirs ]; then \
		rm -rf .testdirs && \
		echo "All test environments removed"; \
	else \
		echo "No test environments found"; \
	fi

release: ## Build release binaries for multiple platforms
	@echo "Building release binaries..."
	@mkdir -p $(BUILD_DIR)/releases
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(FULL_LDFLAGS) \
		-o $(BUILD_DIR)/releases/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	
	# macOS ARM64 (M1/M2)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(FULL_LDFLAGS) \
		-o $(BUILD_DIR)/releases/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(FULL_LDFLAGS) \
		-o $(BUILD_DIR)/releases/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(FULL_LDFLAGS) \
		-o $(BUILD_DIR)/releases/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	
	@echo "Release binaries built in $(BUILD_DIR)/releases/"
	@ls -la $(BUILD_DIR)/releases/

# Documentation targets
docs-install: ## Install MkDocs and dependencies
	@echo "Installing MkDocs dependencies..."
	@if command -v pip >/dev/null 2>&1; then \
		pip install mkdocs-material mkdocs-minify-plugin; \
	elif command -v pip3 >/dev/null 2>&1; then \
		pip3 install mkdocs-material mkdocs-minify-plugin; \
	else \
		echo "pip or pip3 not found. Please install Python and pip first."; \
		exit 1; \
	fi
	@echo "MkDocs dependencies installed"

docs-check: ## Check if MkDocs is installed and configured correctly
	@echo "Checking MkDocs installation..."
	@if command -v mkdocs >/dev/null 2>&1; then \
		echo "✓ MkDocs is installed"; \
		mkdocs --version; \
		echo "✓ Configuration file found: mkdocs.yml"; \
		if mkdocs build --strict --quiet --site-dir .mkdocs-check 2>/dev/null; then \
			echo "✓ Configuration is valid"; \
			rm -rf .mkdocs-check; \
		else \
			echo "✗ Configuration has errors"; \
			rm -rf .mkdocs-check; \
			exit 1; \
		fi; \
	else \
		echo "✗ MkDocs not found. Run 'make docs-install' first"; \
		exit 1; \
	fi

docs-serve: docs-check ## Serve documentation locally with hot reload
	@echo "Starting MkDocs development server..."
	@echo "Open http://127.0.0.1:8000 in your browser"
	@mkdocs serve --dev-addr=127.0.0.1:8000

docs-build: docs-check ## Build documentation for deployment
	@echo "Building documentation..."
	@mkdocs build --strict
	@echo "Documentation built in site/ directory"

docs-clean: ## Clean documentation build artifacts
	@echo "Cleaning documentation build artifacts..."
	@rm -rf site/
	@rm -rf .mkdocs-check/
	@echo "Documentation artifacts cleaned"

docs-deploy: docs-check ## Deploy documentation to GitHub Pages (requires proper git setup)
	@echo "Deploying documentation to GitHub Pages..."
	@mkdocs gh-deploy --clean --message "Update documentation [skip ci]"
	@echo "Documentation deployed"

help: ## Show this help
	@echo "ccmgr-ultra Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*##.*$$' $(MAKEFILE_LIST) | sed 's/:.*##/: /' | awk '{printf "  %-15s %s\n", $$1, substr($$0, index($$0, $$2))}'
	@echo ""
	@echo "Examples:"
	@echo "  make build          # Build the binary"
	@echo "  make install        # Build and install to ~/.local/bin"
	@echo "  make test           # Run tests"
	@echo "  make check          # Run fmt, vet, and lint"
	@echo "  make release        # Build for multiple platforms"
	@echo "  make docs-serve     # Serve documentation locally"
	@echo "  make docs-build     # Build documentation for deployment"