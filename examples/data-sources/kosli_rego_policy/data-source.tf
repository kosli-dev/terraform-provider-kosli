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

# Copy a policy: publish the files read from one policy under another
# identifier. With a second provider configuration (an alias for another
# organization) on the data source, this promotes a policy between organizations.
resource "kosli_rego_policy" "artifact_provenance_copy" {
  identifier   = "artifact-provenance-copy"
  name         = "${data.kosli_rego_policy.artifact_provenance.name} (copy)"
  input_schema = data.kosli_rego_policy.artifact_provenance.input_schema
  comment      = "Copied from artifact-provenance version ${data.kosli_rego_policy.artifact_provenance.version}"
  files        = data.kosli_rego_policy.artifact_provenance.files
}

# Pin a version: warn on every plan and apply when the version an evaluation
# runs is no longer the reviewed one. The digest identifies the files, so it
# stays the same when the reviewed source is published again under a new
# version number.
check "artifact_provenance_is_reviewed_version" {
  assert {
    condition     = data.kosli_rego_policy.artifact_provenance.digest == data.kosli_rego_policy.artifact_provenance_v1.digest
    error_message = "The current version of artifact-provenance is not the reviewed version 1."
  }
}
