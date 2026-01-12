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
  org       = var.kosli_org
  api_url   = var.kosli_api_url
}

# Example 1: Security scan attestation type
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
    }
    required = ["critical_vulnerabilities", "high_vulnerabilities", "scan_date"]
  })

  jq_rules = [
    ".critical_vulnerabilities == 0",
    ".high_vulnerabilities < 5"
  ]
}

# Example 2: Code coverage attestation type
resource "kosli_custom_attestation_type" "code_coverage" {
  name        = "code-coverage"
  description = "Validates code coverage metrics"

  schema = <<-EOT
    {
      "type": "object",
      "properties": {
        "line_coverage": {
          "type": "number",
          "minimum": 0,
          "maximum": 100
        },
        "branch_coverage": {
          "type": "number",
          "minimum": 0,
          "maximum": 100
        },
        "total_lines": {
          "type": "integer"
        },
        "covered_lines": {
          "type": "integer"
        }
      },
      "required": ["line_coverage", "total_lines", "covered_lines"]
    }
  EOT

  jq_rules = [
    ".line_coverage >= 80",
    ".branch_coverage >= 70"
  ]
}

# Example 3: Load external schema from file
resource "kosli_custom_attestation_type" "from_file" {
  name        = "performance-test"
  description = "Validates performance test results"

  schema = file("${path.module}/schemas/performance-schema.json")

  jq_rules = [
    ".response_time_p95 < 200",
    ".error_rate < 0.01"
  ]
}

# Example 4: Query existing attestation type
data "kosli_custom_attestation_type" "security" {
  name = kosli_custom_attestation_type.security_scan.name
}

# Example 5: Create variant based on existing type
resource "kosli_custom_attestation_type" "security_strict" {
  name        = "security-scan-strict"
  description = "Stricter security requirements"

  # Reuse schema from data source
  schema = data.kosli_custom_attestation_type.security.schema

  # Apply stricter rules
  jq_rules = [
    ".critical_vulnerabilities == 0",
    ".high_vulnerabilities == 0",
    ".medium_vulnerabilities < 3"
  ]
}
