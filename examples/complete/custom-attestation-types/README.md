# Complete Example: Kosli Custom Attestation Types

This example demonstrates comprehensive usage of the Kosli Terraform provider, including resources, data sources, and various schema definition methods.

## What This Example Includes

1. **Multiple attestation types** with different schema definition methods
2. **Data source usage** to query existing attestation types
3. **Schema reuse** by creating variants of existing types
4. **External schema files** loaded from JSON files
5. **Outputs** to display created resource attributes

## Usage

### 1. Set up your variables

Copy the example tfvars file:
```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars` and add your credentials:
```hcl
kosli_api_token = "your-kosli-api-token"
kosli_org       = "your-organization-name"
```

**DO NOT** commit `terraform.tfvars` to version control.

### 2. Initialize Terraform

```bash
terraform init
```

### 3. Preview changes

```bash
terraform plan
```

### 4. Apply configuration

```bash
terraform apply
```

### 5. View outputs

```bash
terraform output
```

### 6. Clean up (when done)

```bash
terraform destroy
```

## Files in This Example

- `main.tf` - Main configuration with resources and data sources
- `variables.tf` - Input variable definitions
- `outputs.tf` - Output values
- `terraform.tfvars.example` - Template for your variable values
- `schemas/performance-schema.json` - External schema file example

## Attestation Types Created

### 1. Security Scan
Validates security scan results with vulnerability counts.

**Schema format**: Inline with `jsonencode()`

### 2. Code Coverage
Validates code coverage metrics.

**Schema format**: Heredoc syntax

### 3. Performance Test
Validates performance test results.

**Schema format**: External JSON file

### 4. Security Scan (Strict)
Stricter variant of the security scan attestation.

**Schema source**: Reused from data source

## Schema Definition Methods

This example demonstrates three ways to define JSON schemas:

### Method 1: Inline with jsonencode()
```hcl
schema = jsonencode({
  type = "object"
  properties = {
    field = { type = "string" }
  }
})
```

**Pros**: Type-safe, HCL native
**Best for**: Simple schemas

### Method 2: Heredoc
```hcl
schema = <<-EOT
  {
    "type": "object",
    "properties": {
      "field": { "type": "string" }
    }
  }
EOT
```

**Pros**: Readable, supports complex JSON
**Best for**: Medium-sized schemas

### Method 3: External File
```hcl
schema = file("${path.module}/schemas/my-schema.json")
```

**Pros**: Reusable, version controlled separately
**Best for**: Large or shared schemas

## Data Source Usage

The example shows how to:
1. Query an existing attestation type
2. Reference its attributes in outputs
3. Reuse its schema in a new resource

```hcl
data "kosli_custom_attestation_type" "security" {
  name = kosli_custom_attestation_type.security_scan.name
}

resource "kosli_custom_attestation_type" "variant" {
  schema = data.kosli_custom_attestation_type.security.schema
  # ...
}
```

## Environment Variables

Instead of using `terraform.tfvars`, you can set environment variables:

```bash
export KOSLI_API_TOKEN="your-api-token"
export KOSLI_ORG="your-organization"
export KOSLI_API_URL="https://app.kosli.com"  # Optional

terraform apply
```

## Regional Endpoints

- **EU Region** (default): `https://app.kosli.com`
- **US Region**: `https://app.us.kosli.com`

Specify the region in your variables or provider block:

```hcl
provider "kosli" {
  api_url = "https://app.us.kosli.com"
}
```

## Next Steps

- Explore the [resource documentation](../../resources/kosli_custom_attestation_type/)
- Check out the [data source documentation](../../data-sources/kosli_custom_attestation_type/)
- Learn more at [docs.kosli.com](https://docs.kosli.com)
