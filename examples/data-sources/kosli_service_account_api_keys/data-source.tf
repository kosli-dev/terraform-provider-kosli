terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# A service account with an API key
resource "kosli_service_account" "ci" {
  name      = "ci-pipeline"
  privilege = "member"
}

resource "kosli_service_account_api_key" "ci_key" {
  service_account_name = kosli_service_account.ci.name
  description          = "Production CI key"
}

# List all active API keys for the service account (metadata only; the raw key
# value is never returned after creation).
data "kosli_service_account_api_keys" "ci" {
  service_account_name = kosli_service_account.ci.name

  depends_on = [kosli_service_account_api_key.ci_key]
}

# Example: surface keys that have never been used, for rotation/cleanup.
output "unused_api_key_ids" {
  description = "IDs of API keys that have never been used"
  value       = [for k in data.kosli_service_account_api_keys.ci.keys : k.id if k.last_used_at == null]
}
