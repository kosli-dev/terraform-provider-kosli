# Query an existing custom attestation type
data "kosli_custom_attestation_type" "security" {
  name = "security-scan"
}

# Use the queried schema in a new attestation type
resource "kosli_custom_attestation_type" "security_strict" {
  name        = "security-scan-strict"
  description = "Stricter security requirements"

  # Reuse the schema from the existing type
  schema = data.kosli_custom_attestation_type.security.schema

  # Apply stricter validation rules
  jq_rules = [
    ".critical_vulnerabilities == 0",
    ".high_vulnerabilities == 0",
    ".medium_vulnerabilities < 3"
  ]
}

# Reference attestation type metadata
output "security_scan_description" {
  description = "Description of the security scan attestation type"
  value       = data.kosli_custom_attestation_type.security.description
}

output "security_scan_rules" {
  description = "JQ rules for the security scan attestation type"
  value       = data.kosli_custom_attestation_type.security.jq_rules
}

output "security_scan_archived" {
  description = "Whether the security scan attestation type is archived"
  value       = data.kosli_custom_attestation_type.security.archived
}
