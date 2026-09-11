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

# Omitting expires_at lets the server choose the expiry, which is the maximum
# it allows: 365 days from creation. No API key can be created that never
# expires.
resource "kosli_service_account_api_key" "ci_key" {
  service_account_name = kosli_service_account.ci.name
  description          = "Production CI key"
}

# An API key with an explicit expiry (RFC3339 timestamp). It must be in the
# future and no more than 365 days out - the server silently shortens anything
# longer.
resource "kosli_service_account_api_key" "ci_key_expiring" {
  service_account_name = kosli_service_account.ci.name
  description          = "Temporary CI key"
  # Pick a date within 365 days of when you apply; update it as you rotate.
  expires_at           = "2027-01-01T00:00:00Z"
}

# The raw key is only available on creation and is sensitive
output "ci_api_key" {
  value     = kosli_service_account_api_key.ci_key.key
  sensitive = true
}
