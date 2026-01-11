# Contributing to Terraform Provider for Kosli

Thank you for your interest in contributing to the Terraform Provider for Kosli! This guide will help you get started with development, testing, and contributing to the project.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Building the Provider](#building-the-provider)
- [Testing](#testing)
- [Code Quality](#code-quality)
- [Local Development](#local-development)
- [Submitting Changes](#submitting-changes)
- [Project Structure](#project-structure)

## Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) 1.23 or later
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Git](https://git-scm.com/downloads)
- (Optional) [golangci-lint](https://golangci-lint.run/usage/install/) for linting

### Clone the Repository

```bash
git clone https://github.com/kosli-dev/terraform-provider-kosli.git
cd terraform-provider-kosli
```

### Install Dependencies

```bash
go mod download
```

## Development Workflow

We use a **Makefile** to standardize common development tasks. Run `make help` to see all available targets:

```bash
make help
```

### Quick Start Development Cycle

```bash
# 1. Make your changes
# 2. Format code
make fmt

# 3. Run tests
make test

# 4. Run linter and vet
make vet
make lint  # if golangci-lint is installed

# 5. Build the provider
make build

# 6. Install locally for testing
make install
```

## Building the Provider

### Build Binary

Build the provider binary in the project root:

```bash
make build
```

This creates `terraform-provider-kosli` in the current directory.

### Clean Build Artifacts

Remove build artifacts and cached files:

```bash
make clean
```

## Testing

### Unit Tests

Run unit tests with coverage:

```bash
make test
```

This runs tests across all packages and generates a coverage report saved to `coverage.out`.

**View coverage:**
- Text summary: Already displayed during `make test`
- HTML report: Run `make test-coverage` to open coverage in your browser

### Coverage Report

Generate and view an HTML coverage report:

```bash
make test-coverage
```

### Acceptance Tests

Acceptance tests create real resources and require valid Kosli API credentials.

**Set up environment variables:**

```bash
export KOSLI_API_TOKEN="your-api-token"
export KOSLI_ORG="your-org-name"
```

**Run acceptance tests:**

```bash
make testacc
```

> **Warning:** Acceptance tests may create/modify/delete resources in your Kosli organization. Use a test organization when possible.

### Running Specific Tests

Use Go's standard test flags:

```bash
# Run tests for a specific package
go test ./pkg/client/...

# Run a specific test
go test -run TestClientGet ./pkg/client/...

# Run tests with verbose output
go test -v ./...
```

## Code Quality

### Format Code

Format all Go code using `gofmt`:

```bash
make fmt
```

### Run go vet

Check for common Go mistakes:

```bash
make vet
```

### Linting

Run golangci-lint (requires installation):

```bash
make lint
```

**Install golangci-lint:**

```bash
# macOS
brew install golangci-lint

# Linux/Windows
# See https://golangci-lint.run/usage/install/
```

## Local Development

### Install Provider Locally

Install the provider to your local Terraform plugins directory for testing:

```bash
make install
```

This installs to: `~/.terraform.d/plugins/registry.terraform.io/kosli-dev/kosli/0.1.0/{OS_ARCH}/`

### Using the Local Provider

After installing locally, create a Terraform configuration:

```hcl
terraform {
  required_providers {
    kosli = {
      source  = "kosli-dev/kosli"
      version = "0.1.0"
    }
  }
}

provider "kosli" {
  api_token = var.kosli_api_token
  org       = var.kosli_org_name
}

resource "kosli_attestation_type" "test" {
  name        = "test-type"
  description = "Testing local provider"

  schema = jsonencode({
    type = "object"
    properties = {
      status = { type = "string" }
    }
  })

  jq_rules = [".status == \"pass\""]
}
```

**Initialize and test:**

```bash
terraform init
terraform plan
terraform apply
```

### Debugging

Enable Terraform logging to see provider interactions:

```bash
export TF_LOG=DEBUG
export TF_LOG_PATH=./terraform.log
terraform apply
```

## Submitting Changes

### Branching Strategy

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-description
   ```

2. Make your changes and commit:
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

### Commit Message Convention

Follow conventional commits:

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Adding or updating tests
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

**Examples:**
```
feat: add support for custom attestation types
fix: handle nil pointer in API client
docs: update installation instructions
test: add coverage for error handling
```

### Pull Request Process

1. **Ensure all checks pass:**
   ```bash
   make fmt
   make vet
   make test
   ```

2. **Push your branch:**
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Create a Pull Request** on GitHub with:
   - Clear title describing the change
   - Description of what changed and why
   - Reference to any related issues (e.g., "Closes #123")
   - Test results or screenshots if applicable

4. **Address review feedback** and update your PR as needed

## Project Structure

```
terraform-provider-kosli/
├── adrs/                   # Architecture Decision Records
├── docs/                   # Generated Terraform Registry documentation
├── examples/               # Example Terraform configurations
├── internal/               # Internal provider implementation (future)
│   └── provider/          # Terraform provider resources/data sources
├── pkg/                   # Public packages
│   └── client/            # Kosli API client (reusable)
├── templates/             # Documentation templates
├── go.mod                 # Go module definition
├── Makefile              # Build and test automation
├── main.go               # Provider entry point (future)
└── README.md             # Project overview
```

### Key Directories

- **`adrs/`** - Architecture Decision Records documenting design decisions
- **`pkg/client/`** - Reusable Go API client for Kosli (can be imported by other projects)
- **`internal/provider/`** - Terraform-specific provider implementation (future)
- **`examples/`** - Terraform configuration examples for testing and documentation
- **`docs/`** - Generated documentation (do not edit manually)
- **`templates/`** - tfplugindocs templates for documentation generation

## Development Tips

### API Client Development

The API client (`pkg/client/`) is designed to be:
- **Reusable** - Can be imported by other Go projects
- **Thin wrapper** - Transparently reflects API behavior
- **Well-tested** - Unit tests with high coverage

When adding API methods:
1. Add methods to appropriate file (e.g., `custom_attestation_types.go`)
2. Add tests in corresponding `_test.go` file
3. Update documentation

### Provider Development

When implementing Terraform resources:
1. Define schema using `terraform-plugin-framework`
2. Implement CRUD operations (Create, Read, Update, Delete)
3. Use the API client from `pkg/client/`
4. Add acceptance tests
5. Update examples and documentation

### Running Tests During Development

Keep tests running in watch mode (requires external tool like `watchexec`):

```bash
# Install watchexec
brew install watchexec  # macOS

# Run tests on file changes
watchexec -e go -r "make test"
```

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/kosli-dev/terraform-provider-kosli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/kosli-dev/terraform-provider-kosli/discussions)
- **Kosli Docs**: [docs.kosli.com](https://docs.kosli.com)

## Code of Conduct

Please note that this project follows a Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## License

By contributing to this project, you agree that your contributions will be licensed under the [MIT License](LICENSE).
