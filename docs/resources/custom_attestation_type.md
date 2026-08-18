---
page_title: "kosli_custom_attestation_type Resource - terraform-provider-kosli"
subcategory: ""
description: |-
  Manages a custom attestation type in Kosli. Custom attestation types define how Kosli validates and evaluates evidence from proprietary tools, custom metrics, or specialized compliance requirements.
---

# Resource: kosli_custom_attestation_type

Manages a custom attestation type in Kosli. Custom attestation types define how Kosli validates and evaluates evidence from proprietary tools, custom metrics, or specialized compliance requirements.

Custom attestation types define the structure and validation rules for attestations in Kosli. They can include:

- A JSON Schema (optional) that defines the expected structure of attestation data
- JQ rules (optional) that evaluate the attestation data for compliance

**Note**: While both `schema` and `jq_rules` are optional attributes in Terraform, the Kosli API requires at least one of them to be provided when creating or updating a custom attestation type.

## Example Usage

```terraform
terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Security scan attestation type
resource "kosli_custom_attestation_type" "security_scan" {
  name        = "security-scan"
  description = "Validates security scan results"

  schema = jsonencode({
    type = "object"
    properties = {
      critical_vulnerabilities = { type = "integer" }
      high_vulnerabilities     = { type = "integer" }
      medium_vulnerabilities   = { type = "integer" }
      scan_date                = { type = "string" }
      scanner_version          = { type = "string" }
      report_url               = { type = "string" }
    }
    required = ["critical_vulnerabilities", "high_vulnerabilities", "scan_date"]
  })

  jq_rules = [
    ".critical_vulnerabilities == 0",
    ".high_vulnerabilities < 5"
  ]

  # Ordered, labelled values shown on the attestation detail page in Kosli.
  # A value that is a valid URL renders as a clickable link.
  summary_json = jsonencode([
    { name = "Critical", expression = ".critical_vulnerabilities" },
    { name = "High", expression = ".high_vulnerabilities" },
    { name = "Scanner", expression = ".scanner_version" },
    { name = "Report", expression = ".report_url" },
  ])
}

# Code coverage attestation type
resource "kosli_custom_attestation_type" "code_coverage" {
  name        = "code-coverage"
  description = "Validates code coverage metrics"

  schema = jsonencode({
    type = "object"
    properties = {
      line_coverage = {
        type    = "number"
        minimum = 0
        maximum = 100
      }
      branch_coverage = {
        type    = "number"
        minimum = 0
        maximum = 100
      }
      total_lines   = { type = "integer" }
      covered_lines = { type = "integer" }
    }
    required = ["line_coverage", "total_lines", "covered_lines"]
  })

  jq_rules = [
    ".line_coverage >= 80",
    ".branch_coverage >= 70"
  ]
}

# Attestation type whose schema and summary are kept in standalone JSON files,
# so the same definitions can be shared with the Kosli CLI
resource "kosli_custom_attestation_type" "code_quality" {
  name        = "code-quality"
  description = "Validates code quality metrics"

  schema       = file("${path.module}/schemas/code-quality.json")
  summary_json = file("${path.module}/summaries/code-quality.json")

  jq_rules = [
    ".line_coverage >= 80",
    ".lint_errors == 0"
  ]
}

# Age verification attestation type with only jq rules (no schema)
resource "kosli_custom_attestation_type" "age_verification" {
  name        = "age-verification"
  description = "Verifies age is over 21 using only jq rules without schema validation"

  jq_rules = [".age > 21"]
}

# Schema-only attestation type (no jq rules)
resource "kosli_custom_attestation_type" "schema_validation" {
  name        = "data-structure-validation"
  description = "Validates data structure using schema without evaluation rules"

  schema = jsonencode({
    type = "object"
    properties = {
      timestamp = { type = "string" }
      metadata  = { type = "object" }
      status = {
        type = "string"
        enum = ["pass", "fail", "skip"]
      }
    }
    required = ["timestamp", "status"]
  })
}
```

## Schema Validation

The `schema` attribute is optional and can contain a valid JSON Schema (draft-07) that defines the structure of attestation data. When provided, attestation data will be validated against this schema. Common schema types:

- **Security Scans**: Define vulnerability counts and scan metadata
- **Code Coverage**: Define coverage percentages and test metrics
- **Performance Tests**: Define response times and error rates

### Schema Example

```json
{
  "type": "object",
  "properties": {
    "critical_vulnerabilities": { "type": "integer" },
    "high_vulnerabilities": { "type": "integer" },
    "scan_date": { "type": "string" }
  },
  "required": ["critical_vulnerabilities", "high_vulnerabilities", "scan_date"]
}
```

## JQ Rules

The `jq_rules` attribute is optional and contains an array of JQ expressions that must ALL evaluate to `true` for an attestation to be considered compliant. When provided, each rule is evaluated against the attestation data. If omitted, no evaluation is performed.

### JQ Rules Examples

```hcl
jq_rules = [
  ".critical_vulnerabilities == 0",     # No critical vulnerabilities allowed
  ".high_vulnerabilities < 5",          # Less than 5 high vulnerabilities
  ".scan_date != null"                  # Scan date must be present
]
```

## Import

Custom attestation types can be imported using their name:

```shell
# Import an existing custom attestation type by name
terraform import kosli_custom_attestation_type.security_scan security-scan
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the custom attestation type. Must start with a letter or number and can only contain letters, numbers, periods, hyphens, underscores, and tildes. Changing this will force recreation of the resource.

### Optional

- `description` (String) Description of the custom attestation type. Explains what this attestation type validates.
- `jq_rules` (List of String) List of jq evaluation rules. Each rule is a jq expression that must evaluate to true for the attestation to be considered compliant. Example: `[".coverage >= 80"]`. If omitted, no evaluation is performed.
- `schema` (String) JSON Schema definition that defines the structure of attestation data. Can be provided inline using heredoc syntax or loaded from a file using `file()`. If omitted, no schema validation is performed. Semantic equality is used for comparison, so formatting differences are ignored.
- `summary_json` (String) JSON array of ordered, labelled jq expressions rendered as rows on the attestation detail page in Kosli. Each element is an object with a `name` (the row label) and an `expression` (a jq expression evaluated against the attestation data); values that are valid URLs render as links. Can be provided inline using `jsonencode()`/heredoc syntax or loaded from a file using `file()`, so the same JSON can be shared with the Kosli CLI. Example: `jsonencode([{ name = "Coverage", expression = ".coverage" }])`. If omitted, the attestation detail page falls back to showing the jq evaluation results as a pass/fail checklist; removing it from a type that had one clears the summary. Semantic JSON equality is used when reading the value back from Kosli, so your formatting is preserved rather than being rewritten to the API's compact form.
