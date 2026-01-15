# Outputs for the created attestation types

output "security_scan_name" {
  description = "Name of the security scan attestation type"
  value       = kosli_custom_attestation_type.security_scan.name
}

# Note: The 'org' field was removed from the resource schema per ADR-003.
# Organization context is configured at the provider level, not per-resource.
# If you need the organization name, reference the provider configuration directly.

output "code_coverage_rules" {
  description = "JQ rules for code coverage attestation type"
  value       = kosli_custom_attestation_type.code_coverage.jq_rules
}

output "queried_type_description" {
  description = "Description from the queried attestation type"
  value       = data.kosli_custom_attestation_type.security.description
}
