# Makefile for terraform-provider-kosli

# Binary name
BINARY=terraform-provider-kosli

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Terraform provider installation directory
# This follows the terraform provider plugin directory structure
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)
INSTALL_DIR=~/.terraform.d/plugins/registry.terraform.io/kosli-dev/kosli/0.1.0/$(OS_ARCH)

# Coverage output
COVERAGE_OUT=coverage.out

.PHONY: all build clean test test-coverage testacc fmt vet lint install docs help default

# Default target
default: build

# Build the provider binary
build:
	@echo "Building $(BINARY)..."
	$(GOBUILD) -o $(BINARY) -v

# Install the provider locally for development
install: build
	@echo "Installing $(BINARY) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BINARY) $(INSTALL_DIR)/
	@echo "Provider installed successfully"
	@echo "You can now use it in your Terraform configurations with:"
	@echo "  terraform {"
	@echo "    required_providers {"
	@echo "      kosli = {"
	@echo "        source = \"kosli-dev/kosli\""
	@echo "        version = \"0.1.0\""
	@echo "      }"
	@echo "    }"
	@echo "  }"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -f $(BINARY)
	@rm -f $(COVERAGE_OUT)
	@echo "Clean complete"

# Run unit tests with coverage
test:
	@echo "Running tests with coverage..."
	$(GOTEST) -cover -coverprofile=$(COVERAGE_OUT) ./...

# Generate and display coverage report
test-coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=$(COVERAGE_OUT)

# Run acceptance tests
testacc:
	@echo "Running acceptance tests..."
	TF_ACC=1 $(GOTEST) -v ./...

# Format Go code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run linter (if golangci-lint is available)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

# Generate documentation using tfplugindocs
docs:
	@echo "Generating documentation..."
	@if command -v tfplugindocs >/dev/null 2>&1; then \
		tfplugindocs generate; \
		echo "Documentation generated successfully in docs/"; \
	else \
		echo "tfplugindocs not installed. Install it with:"; \
		echo "  go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest"; \
		exit 1; \
	fi

# Alias for all
all: build

# Display help information
help:
	@echo "Terraform Provider Kosli - Makefile targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build         Build the provider binary"
	@echo "  install       Install the provider locally for development"
	@echo "  clean         Remove build artifacts"
	@echo ""
	@echo "Test targets:"
	@echo "  test          Run unit tests with coverage enabled"
	@echo "  test-coverage Generate and display coverage report"
	@echo "  testacc       Run acceptance tests (with TF_ACC=1)"
	@echo ""
	@echo "Code quality targets:"
	@echo "  fmt           Format Go code"
	@echo "  vet           Run go vet"
	@echo "  lint          Run linter (requires golangci-lint)"
	@echo ""
	@echo "Documentation targets:"
	@echo "  docs          Generate provider documentation (requires tfplugindocs)"
	@echo ""
	@echo "Other targets:"
	@echo "  help          Display this help information"
	@echo "  all           Build the provider (same as build)"
	@echo ""
	@echo "Example usage:"
	@echo "  make build    # Build the provider"
	@echo "  make test     # Run tests with coverage"
	@echo "  make install  # Install locally for development"
