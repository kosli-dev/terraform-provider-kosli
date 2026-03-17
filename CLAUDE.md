# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a Terraform provider for Kosli (https://kosli.com), built using the terraform-plugin-framework. It enables Infrastructure-as-Code management of Kosli resources including custom attestation types and environments.

**Provider Status:** Early-stage under active development. APIs may change.

## Essential Commands

### Build and Install
```bash
make build              # Build the provider binary
make install            # Install locally to ~/.terraform.d/plugins/
make clean              # Remove build artifacts
```

### Testing
```bash
# Unit tests
make test               # Run unit tests with coverage (coverage.out)
make test-coverage      # Generate HTML coverage report

# Acceptance tests (requires KOSLI_API_TOKEN and KOSLI_ORG)
make testacc            # Run all acceptance tests
make testacc-custom-attestation-type          # Specific resource tests
make testacc-custom-attestation-type-datasource
make testacc-environment
make testacc-environment-datasource

# Individual test execution
go test -run TestClientGet ./pkg/client/...
go test -v ./...        # Verbose output
```

### Code Quality
```bash
make fmt                # Format code with gofmt
make vet                # Run go vet
make lint               # Run golangci-lint
```

### Documentation
```bash
make docs               # Generate docs using tfplugindocs
```

## Architecture

### Two-Layer Design

The codebase follows a clean separation between API client and Terraform provider:

**1. API Client Layer (`pkg/client/`)**
- Reusable Go client for Kosli API
- Can be imported by other Go projects
- Thin wrapper that transparently reflects API behavior
- Handles authentication, retries, error parsing
- Files:
  - `client.go` - Core HTTP client with retry logic (hashicorp/go-retryablehttp)
  - `custom_attestation_types.go` - Custom attestation type operations
  - `environments.go` - Environment operations
  - `errors.go` - API error handling

**2. Provider Layer (`internal/provider/`)**
- Terraform-specific implementation using terraform-plugin-framework
- Resources and data sources
- Schema definitions for Terraform configurations
- Files follow pattern: `resource_<name>.go`, `data_source_<name>.go`
- Acceptance tests: `*_acc_test.go`
- Unit tests: `*_test.go`

### Client Configuration

The client supports:
- Environment variables: `KOSLI_API_TOKEN`, `KOSLI_ORG`, `KOSLI_API_URL`
- Regional endpoints: EU (https://app.kosli.com), US (https://app.us.kosli.com)
- Configurable timeouts (default 30s)
- Automatic retry with exponential backoff (3 retries by default)
- Custom User-Agent with provider version

### Current Resources

**Resources:**
- `kosli_action` - Manage webhook notification actions triggered by environment compliance events
- `kosli_custom_attestation_type` - Manage custom attestation types (JSON schema + jq rules)
- `kosli_environment` - Manage physical environments (K8S, ECS, S3, docker, server, lambda)
- `kosli_logical_environment` - Manage logical environments that aggregate physical environments

**Data Sources:**
- `kosli_custom_attestation_type` - Reference existing attestation types
- `kosli_environment` - Reference existing physical environments
- `kosli_logical_environment` - Reference existing logical environments

**Note:** Logical environments can ONLY contain physical environments, not other logical environments. See ADR-004 for the validation strategy.

## Key Patterns

### Provider Entry Point
- `main.go` - Registers provider with Terraform
- Provider version set via ldflags during build (GoReleaser)
- Debug mode available with `-debug` flag

### API Client Usage
```go
client, err := client.NewClient(apiToken, org,
    client.WithTimeout(60*time.Second),
    client.WithUserAgent("terraform-provider-kosli/v0.1.0"),
)
```

### Resource Implementation
Resources must implement terraform-plugin-framework interfaces:
- `Metadata()` - Resource type name
- `Schema()` - Define Terraform schema
- `Create()`, `Read()`, `Update()`, `Delete()` - CRUD operations
- `ImportState()` - Support terraform import

### Error Handling
- Client returns structured errors with HTTP status codes
- Provider converts API errors to Terraform diagnostics
- Check `pkg/client/errors.go` for error types

## Development Workflow

### Adding a New Resource

1. Define API methods in `pkg/client/<resource>.go`
2. Add tests in `pkg/client/<resource>_test.go`
3. Create `internal/provider/resource_<name>.go` with schema and CRUD
4. Create `internal/provider/resource_<name>_test.go` for unit tests
5. Create `internal/provider/resource_<name>_acc_test.go` for acceptance tests
6. Add examples in `examples/resources/<resource_name>/`
7. **Run `make docs` and commit the generated `docs/` files** — CI validates that docs are up to date; skipping this will break the build

> **Important:** Any time you add or modify a Terraform resource or data source (schema changes, new attributes, new resource types), you **must** run `make docs` and stage the generated files before committing. The CI pipeline validates examples and generated docs — missing or stale docs will cause the build to fail.

### Testing Requirements

- Unit tests should mock HTTP responses
- Acceptance tests (`TF_ACC=1`) create real resources - use test org
- All acceptance tests require `KOSLI_API_TOKEN` and `KOSLI_ORG` env vars
- Tests timeout after 30 minutes

### Example Validation

All examples in `examples/` are validated in CI:
- `terraform init -backend=false`
- `terraform validate`

Ensure examples include:
- `terraform.tfvars.example` for required variables
- Comments explaining configuration
- Must pass `terraform validate`

## CI/CD Pipeline

The main pipeline (`.github/workflows/main.yml`) implements Kosli CD flows:

**Pipeline Stages:**
1. Setup - Creates Kosli flow and begins trail
2. Attest PR - Records PR approval
3. Test - Runs unit tests, acceptance tests, linting (all attested to Kosli)
4. Build - Builds binary, generates SBOM, attests artifacts
5. Validate Examples - Tests all Terraform examples

**Kosli Integration:**
- Artifacts attested: binary, SBOM, test results, PR approval
- Flow template: `kosli/template.yml`
- All attestations linked to git commit SHA

### PR Quality Checks

The PR quality workflow (`.github/workflows/pr-quality.yml`) runs automated checks on pull requests:

**Two Jobs:**
1. **Title Validation** - Enforces conventional commit format on PR titles
   - Runs on: opened, edited, reopened, synchronize
   - Validates format: `type(scope): description` (scope optional)
   - Requires lowercase first letter in description
   - Allowed types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert

2. **Claude Code Review** - Automated AI code review
   - Runs on: opened, synchronize (new PRs and commits only)
   - Requires: `ANTHROPIC_API_KEY` secret configured in repository settings
   - Provides inline comments on code quality, bugs, security, performance
   - Posts feedback directly to PR with progress tracking
   - Permissions required: `contents: read`, `pull-requests: write`, `id-token: write` (for OIDC auth)

**Required Secret:**
- `ANTHROPIC_API_KEY` - Add in repository Settings → Secrets and variables → Actions
- Used for Claude Code automated review functionality

## Release Process

Releases use GoReleaser (`.goreleaser.yml`) triggered by git tags:

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

**Automated steps:**
- Multi-platform builds (macOS, Linux, Windows for amd64/arm64)
- GPG signing of artifacts
- SBOM generation
- GitHub Release with changelog
- Binary naming: `terraform-provider-kosli_v{version}`

**Conventional Commits:**
- `feat:` → Features section
- `fix:` → Bug Fixes section
- `docs:` → Documentation section
- Merge commits and `chore:` excluded from release notes

## Important Context

### Attestation Types
- Define JSON schema for attestation data structure
- Include jq rules that evaluate to `true` for compliance
- Naming: Must start with letter/number, contain only letters, numbers, `.`, `-`, `_`, `~`
- Use cases: Security scans, test coverage, code quality, custom compliance

### Service Account Requirements
Provider requires Kosli Service Account with **Admin permissions** to manage resources.

### Architecture Decision Records
The `adrs/` directory contains important architectural decisions:
- ADR-001: Terraform schema design patterns
- ADR-002: API client architecture and reusability
- ADR-003: Resource schema design and API abstraction
- ADR-004: Logical environment validation strategy

Review ADRs when making architectural changes.

## Local Development

### Testing with Local Provider

After `make install`, use in Terraform configs:

```hcl
terraform {
  required_providers {
    kosli = {
      source  = "kosli-dev/kosli"
      version = "dev"
    }
  }
}
```

### Debugging
```bash
export TF_LOG=DEBUG
export TF_LOG_PATH=./terraform.log
terraform apply
```

## Project Structure

```
terraform-provider-kosli/
├── adrs/                   # Architecture Decision Records
├── docs/                   # Generated documentation (don't edit)
├── examples/               # Terraform configuration examples
│   ├── resources/         # Resource examples
│   ├── data-sources/      # Data source examples
│   └── complete/          # End-to-end examples
├── internal/provider/     # Terraform provider implementation
├── pkg/client/            # Reusable Kosli API client
├── templates/             # tfplugindocs templates
├── main.go                # Provider entry point
├── Makefile               # Build automation
└── .github/workflows/     # CI/CD pipelines
```
