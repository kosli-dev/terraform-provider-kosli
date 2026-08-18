# Resource: kosli_custom_attestation_type

This example demonstrates how to create custom attestation types in Kosli using Terraform.

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

## Examples Included

1. **Security Scan**: Attestation type using `jsonencode()` for schema definition
2. **Code Coverage**: Attestation type using heredoc syntax for better readability
3. **Code Quality**: Schema and summary loaded from standalone JSON files with `file()`

## Schema Definition Methods

### Using jsonencode()
```hcl
schema = jsonencode({
  type = "object"
  properties = {
    field_name = { type = "string" }
  }
})
```

### Using Heredoc
```hcl
schema = <<-EOT
  {
    "type": "object",
    "properties": {
      "field_name": { "type": "string" }
    }
  }
EOT
```

## JQ Rules

The `jq_rules` attribute accepts a list of jq expressions that must all evaluate to `true` for attestation compliance.

```hcl
jq_rules = [
  ".critical_vulnerabilities == 0",
  ".high_vulnerabilities < 5"
]
```

## Summary Rows

The optional `summary` attribute is a JSON array of ordered, labelled jq expressions.
Kosli renders them as rows on the attestation detail page, in the order given. A value that
is a valid URL renders as a clickable link.

```hcl
summary = jsonencode([
  { name = "Critical", expression = ".critical_vulnerabilities" },
  { name = "Report", expression = ".report_url" },
])
```

Because it is a plain JSON blob, the same definition can be kept in a file and shared with
other tooling:

```hcl
summary = file("${path.module}/summaries/code-quality.json")
```

If `summary` is omitted, the attestation detail page falls back to showing the jq
evaluation results as a pass/fail checklist. Removing `summary` from a type that had one
clears the summary on the next apply.

## Terraform Provider Configuration

Configuration can be provided via environment variables or provider block:

**Environment Variables:**
- `KOSLI_API_TOKEN` - API token for authentication
- `KOSLI_ORG` - Organization name
- `KOSLI_API_URL` - API endpoint (optional, defaults to EU region)

**Provider Block:**
```hcl
provider "kosli" {
  api_token = var.kosli_api_token
  org       = var.kosli_org
  api_url   = "https://app.kosli.com" # or https://app.us.kosli.com for US region
}
```
