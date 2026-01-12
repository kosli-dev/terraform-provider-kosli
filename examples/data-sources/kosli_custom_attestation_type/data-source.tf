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

# Fetch details of an existing custom attestation type
data "kosli_custom_attestation_type" "existing" {
  name = "security-scan"
}

# Use the data source attributes in outputs
output "attestation_type_description" {
  description = "Description of the attestation type"
  value       = data.kosli_custom_attestation_type.existing.description
}

output "attestation_type_rules" {
  description = "JQ evaluation rules for the attestation type"
  value       = data.kosli_custom_attestation_type.existing.jq_rules
}

output "attestation_type_archived" {
  description = "Whether the attestation type is archived"
  value       = data.kosli_custom_attestation_type.existing.archived
}

# Example: Use data source to reference in another resource
resource "kosli_custom_attestation_type" "variant" {
  name        = "${data.kosli_custom_attestation_type.existing.name}-strict"
  description = "Stricter variant of ${data.kosli_custom_attestation_type.existing.name}"

  # Reuse the same schema
  schema = data.kosli_custom_attestation_type.existing.schema

  # But with stricter rules
  jq_rules = [
    ".critical_vulnerabilities == 0",
    ".high_vulnerabilities == 0",  # Stricter: no high vulnerabilities allowed
    ".medium_vulnerabilities < 3"
  ]
}
