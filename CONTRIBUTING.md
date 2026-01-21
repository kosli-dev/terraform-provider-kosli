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

resource "kosli_custom_attestation_type" "test" {
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

## Release Process

This project uses [GoReleaser](https://goreleaser.com/) to automate multi-platform binary builds and GitHub releases.

### Prerequisites for Releases

#### Install GoReleaser

```bash
# macOS
brew install goreleaser

# Linux/Windows
go install github.com/goreleaser/goreleaser/v2@latest
```

#### GPG Signing Setup

Terraform Registry requires signed binaries. Set up GPG if releasing:

1. **Generate GPG key** (if needed):
   ```bash
   gpg --full-generate-key
   # Choose: RSA and RSA, 4096 bits, no expiration
   # Use your GitHub email
   # Passphrase: Leave empty (press Enter twice) for CI/CD automation
   ```

2. **List your GPG keys and get the fingerprint**:
   ```bash
   gpg --list-secret-keys --keyid-format LONG
   ```

   This will output something like:
   ```
   sec   rsa4096/ABCD1234EFGH5678 2024-01-20 [SC]
         1234567890ABCDEF1234567890ABCDEF12345678
   uid                 [ultimate] Your Name <your-email@example.com>
   ```

   The 40-character string is the fingerprint. Copy it for step 4.

3. **Export the specific key for GitHub Actions** (replace `FINGERPRINT` with your actual 40-character fingerprint):
   ```bash
   # Public key (for Terraform Registry)
   gpg --armor --export FINGERPRINT > public-key.asc

   # Private key (for GitHub Secrets) - this will output to console, copy the entire output
   gpg --armor --export-secret-keys FINGERPRINT
   ```

   **Tip**: If you prefer using email and have multiple keys with the same email, use the fingerprint to be specific.

4. **Configure environment**:
   ```bash
   export GPG_FINGERPRINT=<your-40-char-fingerprint>
   ```

### Testing Release Locally

Test the release configuration without publishing:

```bash
# Validate configuration
goreleaser check

# Build snapshot (no release, no git tags required)
goreleaser build --snapshot --clean

# Full release dry-run
goreleaser release --snapshot --clean
```

Built binaries will be in the `dist/` directory.

### Release Workflow

Releases are fully automated via GitHub Actions using the workflow at `.github/workflows/release.yml`.

#### Required GitHub Secrets

Before creating a release, ensure the following secrets are configured in the repository:

- `GPG_PRIVATE_KEY` - Your GPG private key (exported with `gpg --armor --export-secret-keys`)
- `GPG_PASSPHRASE` - (Optional) The passphrase for your GPG key. If you generated the key without a passphrase for CI/CD automation, you can omit this secret or set it to an empty string.
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions (no configuration needed)

#### Creating a Release

1. **Update CHANGELOG.md** with release notes for the version
2. **Create and push a version tag**:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```
3. **GitHub Actions workflow automatically**:
   - Checks out code with full git history
   - Sets up Go using the version from `go.mod`
   - Imports GPG key for signing
   - Runs GoReleaser to build multi-platform binaries
   - Signs all artifacts with GPG
   - Creates GitHub Release with changelog
   - Publishes binaries and checksums

4. **Release artifacts** are published to GitHub Releases with:
   - Multi-platform binaries (macOS, Linux, Windows)
   - SHA256 checksums
   - GPG signatures
   - Auto-generated changelog

#### Manual Workflow Trigger

The release workflow can also be triggered manually from the GitHub Actions UI for testing purposes:
1. Navigate to **Actions** → **Release** workflow
2. Click **Run workflow**
3. Select the branch and click **Run workflow**

### Conventional Commits & Release Notes

GoReleaser automatically organizes release notes using conventional commits:

- `feat:` → **Features** section
- `fix:` → **Bug Fixes** section
- `docs:` → **Documentation** section
- `refactor:` → **Refactoring** section
- `test:` → **Testing** section
- `build:` or `ci:` → **Build & CI** section

Commits starting with `chore:`, `style:`, or merge commits are excluded from release notes.

**Example:**
```bash
git commit -m "feat: add custom attestation type resource"
git commit -m "fix: handle nil pointer in API client"
git commit -m "docs: update installation guide"
```

These will be automatically grouped in the release notes under their respective sections.

### Multi-Platform Builds

GoReleaser builds binaries for:
- **macOS**: Intel (amd64) and Apple Silicon (arm64)
- **Linux**: x64 (amd64) and ARM64 (arm64)
- **Windows**: x64 (amd64)

Binary naming follows Terraform provider conventions:
- Binary: `terraform-provider-kosli_v{version}`
- Archive: `terraform-provider-kosli_{version}_{os}_{arch}.tar.gz`

### Pre-release Testing with Release Candidates

Release candidates allow testing versions before official releases.

#### Creating Release Candidates

```bash
# Create release candidate tag
git tag -a v0.1.0-rc.1 -m "Release candidate 1 for v0.1.0"
git push origin v0.1.0-rc.1
```

The tag format uses a hyphen followed by `rc.N` where N is the release candidate number.

#### GoReleaser Behavior

Our `.goreleaser.yml` includes `prerelease: auto`, which automatically:
- Detects release candidate tags (e.g., `v0.1.0-rc.1`)
- Marks the GitHub Release as "Pre-release"
- Generates release notes using the same changelog groups

No additional configuration is needed for release candidates.

#### Terraform Registry Implications

**✅ Supported**: Release candidate versions work with Terraform Registry.

**⚠️ Not Auto-selected**: Users must explicitly specify the RC version:

```hcl
terraform {
  required_providers {
    kosli = {
      source  = "kosli-dev/kosli"
      version = "0.1.0-rc.1"  # Must specify RC explicitly
    }
  }
}
```

**Version Constraint Behavior**:
- `version = ">= 0.1.0"` will **not** match `0.1.0-rc.1` (pre-releases excluded)
- `version = "0.1.0-rc.1"` matches exactly
- `version = "~> 0.1.0-rc"` matches `rc.1`, `rc.2`, etc.

#### Testing Workflow

```bash
# 1. Create first release candidate
git tag -a v0.1.0-rc.1 -m "Release candidate 1"
git push origin v0.1.0-rc.1

# 2. Test with explicit RC version in Terraform configurations

# 3. If issues found, create additional RCs
git tag -a v0.1.0-rc.2 -m "Release candidate 2"
git push origin v0.1.0-rc.2

# 4. When stable, create final release
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

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

### Example Development

When adding or updating Terraform examples in the `examples/` directory:

1. **Follow the standard structure:**
   - `examples/resources/<resource_name>/` - Resource usage examples
   - `examples/data-sources/<data_source_name>/` - Data source examples
   - `examples/complete/` - Comprehensive examples with multiple resources

2. **Include a README.md** for each example explaining:
   - What the example demonstrates
   - Prerequisites (API tokens, organization setup)
   - How to run the example
   - Expected outcomes

3. **Validate all examples** before committing:
   ```bash
   terraform validate examples/resources/kosli_custom_attestation_type/
   terraform validate examples/data-sources/kosli_custom_attestation_type/
   terraform validate examples/complete/
   ```

4. **Example requirements:**
   - Must include `terraform.tfvars.example` for any required variables
   - Should demonstrate best practices (schema definition methods, error handling)
   - Must be syntactically correct and pass `terraform validate`
   - Should include comments explaining non-obvious configuration

5. **Testing examples locally:**
   - Set up provider development overrides in `~/.terraformrc`
   - Skip `terraform init` (use dev overrides)
   - Run `terraform plan` to verify configuration
   - For complete testing: `terraform apply` (requires valid Kosli credentials)

**Note:** The CI pipeline automatically validates all examples on every PR to ensure they remain correct as the provider evolves.

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
