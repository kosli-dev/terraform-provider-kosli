# Data Source: kosli_custom_attestation_type

This example demonstrates how to query existing custom attestation types in Kosli using Terraform data sources.

## Usage

```bash
# Set environment variables
export KOSLI_API_TOKEN="your-api-token"
export KOSLI_ORG="your-organization"

# Initialize Terraform
terraform init

# Preview changes
terraform plan

# Apply configuration
terraform apply
```

## What This Example Shows

1. **Fetching existing attestation type**: Query an attestation type by name
2. **Using data source attributes**: Reference attributes in outputs
3. **Reusing schemas**: Create variants of existing attestation types

## Data Source Attributes

The data source provides read-only access to:

- `name` - (Required) Name of the attestation type to query
- `description` - Description of the attestation type
- `schema` - JSON Schema definition
- `jq_rules` - List of jq evaluation rules
- `archived` - Whether the attestation type is archived
- `org` - Organization name

## Use Cases

### Reference Existing Types
```hcl
data "kosli_custom_attestation_type" "existing" {
  name = "security-scan"
}

output "rules" {
  value = data.kosli_custom_attestation_type.existing.jq_rules
}
```

### Create Variants
```hcl
resource "kosli_custom_attestation_type" "strict_variant" {
  name   = "${data.kosli_custom_attestation_type.existing.name}-strict"
  schema = data.kosli_custom_attestation_type.existing.schema

  jq_rules = [
    # Stricter rules here
  ]
}
```

## Prerequisites

The attestation type you're querying must already exist in your Kosli organization. You can create it:
- Through the Kosli web interface
- Using the Kosli CLI
- Using another Terraform configuration with the resource

## Terraform Provider Configuration

Configuration can be provided via environment variables or provider block:

**Environment Variables:**
- `KOSLI_API_TOKEN` - API token for authentication
- `KOSLI_ORG` - Organization name
- `KOSLI_API_URL` - API endpoint (optional, defaults to EU region)
