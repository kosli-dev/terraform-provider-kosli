# Terraform Provider for Kosli

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> [!WARNING]
> This is an early-stage provider under active development. Features and APIs may change.
> We recommend not to use this provider in production environments yet, and to pin to specific versions when you do.


Manage [Kosli](https://kosli.com) resources using Terraform. This provider allows you to define and manage Kosli custom attestation types as Infrastructure-as-Code, enabling you to integrate proprietary tools, custom metrics, or specialized compliance requirements into your Kosli workflows.

The Terraform provider enables you to automate the management of Kosli resources alongside your infrastructure.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.8
- [Go](https://golang.org/doc/install) 1.23 or later (for development)
- Kosli account and API credentials

## Quick Start

### Installation

The provider is available on the [Terraform Registry](https://registry.terraform.io/providers/kosli-dev/kosli/latest). Add it to your Terraform configuration:

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
- [Resource examples](examples/resources/) - Creating and managing resources
- [Data source examples](examples/data-sources/) - Referencing existing resources
- [Complete examples](examples/complete/) - End-to-end scenarios

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

Full documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/kosli-dev/kosli/latest/docs) and in the [docs/](docs/) directory.

For more details on attestation types, see the [Kosli documentation](https://docs.kosli.com/client_reference/kosli_create_attestation-type/).

## Supported Resources

### Resources
- `kosli_custom_attestation_type` - Create and manage custom attestation types

### Data Sources
- `kosli_custom_attestation_type` - Reference existing attestation types

*Additional resources will be added as the provider matures.*

## Configuration

The Kosli provider requires authentication via API token and organization name.

### Using Environment Variables (Recommended)

The recommended approach is to use environment variables, especially for sensitive credentials:

```bash
export KOSLI_API_TOKEN="your-api-token"
export KOSLI_ORG="your-org-name"
export KOSLI_API_URL="https://app.kosli.com"  # Optional, defaults to EU region
```

Then configure the provider without hardcoded credentials:

```hcl
provider "kosli" {
  # Credentials are read from environment variables:
  # KOSLI_API_TOKEN, KOSLI_ORG, KOSLI_API_URL
}
```

### Using Terraform Variables

Alternatively, use Terraform variables (ensure you manage secrets securely):

```hcl
provider "kosli" {
  api_token = var.kosli_api_token  # Use secure variable management
  org       = var.kosli_org_name
  api_url   = "https://app.kosli.com"  # Optional, defaults to EU region
  timeout   = 30                        # Optional, defaults to 30s
}
```

### Regional Endpoints

Kosli operates in two regions:

- **EU Region** (default): `https://app.kosli.com`
- **US Region**: `https://app.us.kosli.com`

Configure the appropriate endpoint based on where your Kosli organization is hosted.

### Getting API Credentials

**Recommended: Use Service Accounts**

Service accounts provide secure, programmatic access to Kosli without tying credentials to individual users:

1. Log in to your [Kosli account](https://app.kosli.com)
2. Navigate to **Settings → Service Accounts**
3. Click **Add New Service Account**
4. Give it a descriptive name (e.g., "Terraform Automation")
5. Click the **"..."** menu on the service account
6. Select **Add API Key**
7. Copy the API key and store it securely

**Store credentials securely:**
- Use environment variables (see Configuration above)
- For CI/CD: Use your platform's secrets management (GitHub Secrets, GitLab CI/CD variables, etc.)
- For local development: Use a `.envrc` file (with direnv) or similar - **never commit credentials to version control**

## Contributing

We welcome contributions! Whether you're fixing a bug, adding a feature, or improving documentation, your help is appreciated.

### Quick Start for Contributors

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Make your changes and add tests
4. Run `make fmt && make vet && make test`
5. Submit a pull request

### Development Guide

For comprehensive development information, see **[CONTRIBUTING.md](CONTRIBUTING.md)**:

- **Development environment setup** - Prerequisites and dependencies
- **Building and testing** - Make commands and workflows
- **Code quality standards** - Formatting, linting, and best practices
- **Pull request process** - Detailed submission guidelines and review timeline
- **Project structure** - Directory organization and architecture
- **Release process** - How releases are created and published

### Common Development Commands

```bash
make help         # View all available commands
make build        # Build the provider
make test         # Run unit tests with coverage
make testacc      # Run acceptance tests
make install      # Install locally for testing
```

### Getting Help

- **Questions**: [GitHub Discussions](https://github.com/kosli-dev/terraform-provider-kosli/discussions)
- **Bug reports**: [GitHub Issues](https://github.com/kosli-dev/terraform-provider-kosli/issues)
- **Community**: [Kosli Community](https://www.kosli.com/community/)

## Roadmap

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
