terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# A single-file policy, read from a .rego file kept next to the configuration
resource "kosli_rego_policy" "artifact_provenance" {
  identifier   = "artifact-provenance"
  name         = "Artifact provenance"
  description  = "Requires every compliance decision to be linked to an artifact fingerprint"
  input_schema = "trail"
  comment      = "Initial version"

  files = {
    "policy.rego" = file("${path.module}/policies/artifact_provenance.rego")
  }
}

# A bundle of every .rego file in a directory, keyed by its path in the bundle
resource "kosli_rego_policy" "release_checks" {
  identifier   = "release-checks"
  name         = "Release checks"
  input_schema = "trails"

  files = {
    for name in fileset("${path.module}/policies", "**/*.rego") :
    name => file("${path.module}/policies/${name}")
  }
}

output "artifact_provenance_digest" {
  description = "Digest of the version an evaluation runs"
  value       = kosli_rego_policy.artifact_provenance.digest
}
