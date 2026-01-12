# Terraform Provider for Kosli

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> [!WARNING]
> This is an early-stage provider under active development. Features and APIs may change.
> We recommend not to use this provider in production environments yet, and to pin to specific versions when you do.


Manage [Kosli](https://kosli.com) resources using Terraform. This provider allows you to define and manage Kosli custom attestation types as Infrastructure-as-Code, enabling you to integrate proprietary tools, custom metrics, or specialized compliance requirements into your Kosli workflows.

The Terraform provider enables you to automate the management of Kosli resources alongside your infrastructure.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) 1.21 or later (for development)
- Kosli account and API credentials

## Quick Start

### Installation

The provider will be available on the [Terraform Registry](https://registry.terraform.io/providers/kosli-dev/kosli/). Add it to your Terraform configuration:

```hcl
terraform {
  required_providers {
    kosli = {
      source  = "kosli-dev/kosli"
      version = "~> 0.1"
    }
  }
}

provider "kosli" {
  api_token = var.kosli_api_token
  org       = var.kosli_org_name
  api_url   = "https://app.kosli.com"  # Optional, defaults to EU region
}
```

### Basic Configuration

Here's a minimal example to get started. For complete examples with variables and multiple use cases, see the [examples](examples/) directory.

```hcl
resource "kosli_custom_attestation_type" "example" {
  name        = "coverage-check"
  description = "Validate test coverage meets minimum threshold"

  schema = <<-EOT
    {
      "type": "object",
      "properties": {
        "line_coverage": {
          "type": "number"
        }
      },
      "required": ["line_coverage"]
    }
  EOT

  jq_rules = [
    ".line_coverage >= 80"
  ]
}
```

**Example configurations:**
- [Complete examples](examples/kosli_custom_attestation_type/) - Creating, managing, and referencing attestation types

## Documentation

### Attestation Types

Attestation types are custom data structures that define how Kosli validates and evaluates evidence. They act as templates specifying:

- **JSON Schema**: Defines the structure and data types for attestation data
- **Evaluation Rules**: jq-formatted rules that must evaluate to `true` for compliance
- **Naming Convention**: Names must start with a letter/number and contain only letters, numbers, periods, hyphens, underscores, and tildes

Common use cases include:
- Security scan validation (e.g., no critical vulnerabilities)
- Test coverage requirements (e.g., minimum 80% coverage)
- Code quality checks (e.g., no failing tests)
- Custom compliance criteria specific to your organization

Full documentation will be published on the [Terraform Registry](https://registry.terraform.io/providers/kosli-dev/kosli/).

For more details on attestation types, see the [Kosli documentation](https://docs.kosli.com/client_reference/kosli_create_attestation-type/).

## Supported Resources

### Resources
- `kosli_custom_attestation_type` - Create and manage custom attestation types

### Data Sources
- `kosli_custom_attestation_type` - Reference existing attestation types

*Additional resources will be added as the provider matures.*

## Configuration

The Kosli provider requires authentication via API token and organization name:

```hcl
provider "kosli" {
  api_token = var.kosli_api_token
  org       = var.kosli_org_name
  # api_url is optional, defaults to https://app.kosli.com (EU region)
  # For US region, use: https://app.us.kosli.com
  # timeout is optional, defaults to 30s
}
```

Alternatively, use environment variables:

```bash
export KOSLI_API_TOKEN="your-api-token"
export KOSLI_ORG="your-org-name"
export KOSLI_API_URL="https://app.kosli.com"  # Optional, EU (default) or US region
```

### Regional Endpoints

Kosli operates in two regions:

- **EU Region** (default): `https://app.kosli.com`
- **US Region**: `https://app.us.kosli.com`

Configure the appropriate endpoint based on where your Kosli organization is hosted.

### Getting API Credentials

1. Log in to your [Kosli account](https://app.kosli.com)
2. Navigate to Settings → API Tokens
3. Create a new API token with appropriate scopes
4. Store securely (use Terraform variables or secrets management)

## Development

### Quick Start

We use **Make** to standardize development workflows:

```bash
# Clone and navigate to repository
git clone https://github.com/kosli-dev/terraform-provider-kosli.git
cd terraform-provider-kosli

# View available commands
make help

# Build the provider
make build

# Run tests with coverage
make test

# Install locally for development
make install
```

### Common Development Tasks

```bash
make fmt          # Format code
make vet          # Run go vet
make lint         # Run linter (requires golangci-lint)
make test         # Run unit tests with coverage
make testacc      # Run acceptance tests (requires KOSLI_API_TOKEN and KOSLI_ORG)
make clean        # Remove build artifacts
```

For detailed development guides, testing procedures, and contribution guidelines, see **[CONTRIBUTING.md](CONTRIBUTING.md)**.

### Project Structure

```
.
├── .github/
│   └── workflows/                          # GitHub Actions workflows
│       ├── test.yml                        # Test workflow
│       └── release.yml                     # Release workflow
├── docs/                                    # Generated documentation
├── examples/                                # Terraform configuration examples
├── internal/
│   ├── provider/
│   │   ├── data_source_attestation_type.go # Attestation type data source
│   │   ├── provider.go                     # Provider configuration
│   │   └── resource_attestation_type.go    # Attestation type resource
│   └── utils/                              # Helper functions
├── pkg/
│   └── client/                             # Kosli API client (public, reusable)
│       ├── attestation_types.go            # Attestation types API methods
│       ├── client.go                       # Core client implementation
│       └── client_test.go                  # Client tests
├── go.mod                                   # Go module definition
├── go.sum                                   # Go module checksums
└── main.go                                  # Provider entry point
```

### Contributing

We welcome contributions! Please see **[CONTRIBUTING.md](CONTRIBUTING.md)** for:

- Development environment setup
- Building and testing guide
- Code quality standards
- Pull request process
- Project structure overview

**Quick contribution checklist:**

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Make your changes and add tests
4. Run `make fmt && make vet && make test`
5. Submit a pull request

For questions or discussions, see [GitHub Issues](https://github.com/kosli-dev/terraform-provider-kosli/issues) or [Discussions](https://github.com/kosli-dev/terraform-provider-kosli/discussions).

## Roadmap

### v0.1 (Initial Release)
- [ ] Provider authentication and configuration
- [ ] Attestation types resource
- [ ] Attestation types data source

### Future Versions
- [ ] Additional Kosli resources (environments, flows, etc.)
- [ ] Enhanced attestation type features
- [ ] Import existing attestation types

See [GitHub Issues](https://github.com/kosli-dev/terraform-provider-kosli/issues) for detailed feature tracking.

## Troubleshooting

### Authentication Errors

Verify your API token and organization are valid:

```bash
# For EU region (default)
curl -H "Authorization: Bearer $KOSLI_API_TOKEN" https://app.kosli.com/api/v2/environments/$KOSLI_ORG

# For US region
curl -H "Authorization: Bearer $KOSLI_API_TOKEN" https://app.us.kosli.com/api/v2/environments/$KOSLI_ORG
```

### API Timeouts

Increase the timeout if you're experiencing timeout errors:

```hcl
provider "kosli" {
  api_token = var.kosli_api_token
  timeout   = 60  # seconds
}
```

### Resource State Issues

If Terraform state becomes out of sync with Kosli:

```bash
terraform refresh
```

## Support

- **Documentation**: https://docs.kosli.com
- **Issues**: [GitHub Issues](https://github.com/kosli-dev/terraform-provider-kosli/issues)
- **Community**: [Kosli Slack Community](https://koslicommunity.slack.com)
- **Email**: support@kosli.com

## License

This provider is released under the [MIT License](LICENSE).

## About Kosli

Kosli is a software intelligence platform that helps teams maintain visibility and governance over their Software Delivery Lifecycle (SDLC). Learn more at [kosli.com](https://kosli.com).
