terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Read the current version of a policy: the one an evaluation runs
data "kosli_rego_policy" "artifact_provenance" {
  identifier = "artifact-provenance"
}

# Read a specific version, whatever its validation status
data "kosli_rego_policy" "artifact_provenance_v1" {
  identifier = "artifact-provenance"
  version    = 1
}

output "current_version" {
  description = "Number of the version an evaluation runs"
  value       = data.kosli_rego_policy.artifact_provenance.version
}

output "current_digest" {
  description = "Digest a decision made with the current version cites"
  value       = data.kosli_rego_policy.artifact_provenance.digest
}

output "policy_source" {
  description = "Source of the policy's main file"
  value       = data.kosli_rego_policy.artifact_provenance.files["policy.rego"]
}
