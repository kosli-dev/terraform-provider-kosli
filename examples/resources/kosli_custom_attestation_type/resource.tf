terraform {
  required_providers {
    kosli = {
      source  = "kosli-dev/kosli"
      version = "~> 0.1"
    }
  }
}

provider "kosli" {
  # Configuration can be provided via:
  # - Environment variables: KOSLI_API_TOKEN, KOSLI_ORG, KOSLI_API_URL
  # - Provider block attributes (shown below)

  # api_token = var.kosli_api_token
  # org       = var.kosli_org
  # api_url   = "https://app.kosli.com" # Optional, defaults to EU region
}

# Basic example: Custom attestation type with inline JSON schema
resource "kosli_custom_attestation_type" "security_scan" {
  name        = "security-scan"
  description = "Validates security scan results meet compliance requirements"

  schema = jsonencode({
    type = "object"
    properties = {
      critical_vulnerabilities = {
        type = "integer"
      }
      high_vulnerabilities = {
        type = "integer"
      }
      scan_date = {
        type = "string"
      }
    }
    required = ["critical_vulnerabilities", "high_vulnerabilities"]
  })

  jq_rules = [
    ".critical_vulnerabilities == 0",
    ".high_vulnerabilities < 5"
  ]
}

# Example with heredoc for better readability
resource "kosli_custom_attestation_type" "code_coverage" {
  name        = "code-coverage-check"
  description = "Validates test coverage meets minimum thresholds"

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
        }
      },
      "required": ["line_coverage"]
    }
  EOT

  jq_rules = [
    ".line_coverage >= 80",
    ".branch_coverage >= 70"
  ]
}
