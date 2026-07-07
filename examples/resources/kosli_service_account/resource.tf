terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Service account for a CI/CD pipeline
resource "kosli_service_account" "ci" {
  name        = "ci-pipeline"
  description = "CI/CD pipeline service account"
  privilege   = "member"
}

# Read-only service account (e.g. for dashboards)
resource "kosli_service_account" "dashboard" {
  name      = "dashboard-readonly"
  privilege = "reader"
}
