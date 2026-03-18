terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Query an existing policy
data "kosli_policy" "production" {
  name = "prod-requirements"
}

output "policy_latest_version" {
  description = "Latest version number of the policy"
  value       = data.kosli_policy.production.latest_version
}

output "policy_content" {
  description = "YAML content of the latest policy version"
  value       = data.kosli_policy.production.content
}
