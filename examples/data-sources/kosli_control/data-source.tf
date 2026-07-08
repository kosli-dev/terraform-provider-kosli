terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Create a control
resource "kosli_control" "binary_provenance" {
  identifier  = "SDLC-001"
  name        = "Binary provenance"
  description = "All production artifacts must have build provenance attestations"
}

# Look up the control via data source
data "kosli_control" "binary_provenance" {
  identifier = kosli_control.binary_provenance.identifier
}

# Reference control metadata
output "control_version" {
  description = "Current version of the control"
  value       = data.kosli_control.binary_provenance.version
}

output "control_policies" {
  description = "Environment policies referencing the control"
  value       = data.kosli_control.binary_provenance.policies_referencing
}

output "control_tags" {
  description = "Tags on the control"
  value       = data.kosli_control.binary_provenance.tags
}
