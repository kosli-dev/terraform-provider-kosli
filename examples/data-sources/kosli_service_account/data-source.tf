terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Create a service account
resource "kosli_service_account" "ci" {
  name        = "ci-pipeline"
  description = "CI/CD pipeline service account"
  privilege   = "member"
}

# Look up the service account via data source
data "kosli_service_account" "ci" {
  name = kosli_service_account.ci.name
}

# Reference service account metadata
output "ci_privilege" {
  description = "Privilege (role) of the CI service account"
  value       = data.kosli_service_account.ci.privilege
}

output "ci_created_at" {
  description = "Unix timestamp of when the service account was created"
  value       = data.kosli_service_account.ci.created_at
}
