terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# All repos whose name contains "service", on a self-hosted GitLab
data "kosli_repos" "services" {
  search       = "service"
  vcs_provider = "gitlab"
  vcs_instance = "gitlab.acme.com"
}

# Repos owned by the platform team that have any tier tag
data "kosli_repos" "platform_tiered" {
  tag = ["team:platform", "tier"]
}

output "service_repo_ids" {
  description = "Internal IDs of the matching repos, for use with kosli_repo_tags"
  value       = data.kosli_repos.services.repos[*].id
}

output "platform_tiered_repo_names" {
  description = "Names of the platform team's tiered repos"
  value       = data.kosli_repos.platform_tiered.repos[*].name
}
