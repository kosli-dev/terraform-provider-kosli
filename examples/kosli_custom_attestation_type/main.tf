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
  api_url   = var.kosli_api_url # Optional, defaults to EU region
}

# Example 1: Define a custom attestation type with inline schema using heredoc
resource "kosli_custom_attestation_type" "age_verification" {
  name        = "age-verification"
  description = "Verify person meets age requirements"

  schema = <<-EOT
    {
      "type": "object",
      "properties": {
        "age": {
          "type": "integer"
        },
        "name": {
          "type": "string"
        }
      },
      "required": ["age", "name"]
    }
  EOT

  jq_rules = [
    ".age >= 18",
    ".age <= 65"
  ]
}

# Example 2: Define a custom attestation type using a schema file
resource "kosli_custom_attestation_type" "coverage_check" {
  name        = "coverage-check"
  description = "Validate test coverage meets minimum threshold"

  schema = file("${path.module}/schemas/coverage-schema.json")

  jq_rules = [
    ".line_coverage >= 80"
  ]
}

# Example 3: Security scan attestation type with inline JSON
resource "kosli_custom_attestation_type" "security_scan" {
  name        = "security-scan"
  description = "Validate security scan results"

  schema = <<-EOT
    {
      "type": "object",
      "properties": {
        "critical_vulnerabilities": {
          "type": "integer"
        },
        "high_vulnerabilities": {
          "type": "integer"
        },
        "scan_date": {
          "type": "string"
        }
      },
      "required": ["critical_vulnerabilities", "high_vulnerabilities"]
    }
  EOT

  jq_rules = [
    ".critical_vulnerabilities == 0",
    ".high_vulnerabilities < 5"
  ]
}

# Example 4: Reference an existing attestation type using a data source
data "kosli_custom_attestation_type" "existing_type" {
  name = "security-scan"
}

# Use the data source attributes
output "existing_type_description" {
  value = data.kosli_custom_attestation_type.existing_type.description
}

output "existing_type_jq_rules" {
  value = data.kosli_custom_attestation_type.existing_type.jq_rules
}

# Example 5: Schema as a data source (type-safe, reusable approach)
# This follows AWS provider patterns for IAM policies and other structured data
data "kosli_attestation_schema" "reusable_coverage_schema" {
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
    total_lines = {
      type = "integer"
    }
    covered_lines = {
      type = "integer"
    }
  }
  required = ["line_coverage", "total_lines", "covered_lines"]
}

resource "kosli_custom_attestation_type" "coverage_with_data_source" {
  name        = "coverage-check-typed"
  description = "Type-safe coverage check using schema data source"
  schema      = data.kosli_attestation_schema.reusable_coverage_schema.json

  jq_rules = [
    ".line_coverage >= 80",
    ".branch_coverage >= 70"
  ]
}

# The schema data source can be reused across multiple attestation types
resource "kosli_custom_attestation_type" "strict_coverage_check" {
  name        = "strict-coverage-check"
  description = "Stricter coverage requirements using the same schema"
  schema      = data.kosli_attestation_schema.reusable_coverage_schema.json

  jq_rules = [
    ".line_coverage >= 90",
    ".branch_coverage >= 85"
  ]
}
