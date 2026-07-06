terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Service account that the API key belongs to
resource "kosli_service_account" "ci" {
  name        = "ci-pipeline"
  description = "CI/CD pipeline service account"
  privilege   = "member"
}

# A non-expiring API key
resource "kosli_service_account_api_key" "ci_key" {
  service_account_name = kosli_service_account.ci.name
  description          = "Production CI key"
}

# An API key that expires (Unix timestamp, seconds)
resource "kosli_service_account_api_key" "ci_key_expiring" {
  service_account_name = kosli_service_account.ci.name
  description          = "Temporary CI key"
  expires_at           = 4102444800 # 2100-01-01
}

# The raw key is only available on creation and is sensitive
output "ci_api_key" {
  value     = kosli_service_account_api_key.ci_key.key
  sensitive = true
}
