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

# Repos owned by the platform or payments team that have a tier tag with any value
data "kosli_repos" "tiered" {
  tag_filter {
    key    = "team"
    values = ["platform", "payments"]
  }
  tag_filter {
    key = "tier"
  }
}

output "service_repo_ids" {
  description = "Internal IDs of the matching repos, for use with kosli_repo_tags"
  value       = data.kosli_repos.services.repos[*].id
}

output "tiered_repo_names" {
  description = "Names of the tiered repos owned by the platform or payments team"
  value       = data.kosli_repos.tiered.repos[*].name
}
