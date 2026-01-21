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
    }
    required = ["critical_vulnerabilities", "high_vulnerabilities", "scan_date"]
  })

  jq_rules = [
    ".critical_vulnerabilities == 0",
    ".high_vulnerabilities < 5"
  ]
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
      status    = {
        type = "string"
        enum = ["pass", "fail", "skip"]
      }
    }
    required = ["timestamp", "status"]
  })
}
