terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Look a repo up by name. Set vcs_provider when the name exists on more than one provider.
data "kosli_repo" "api" {
  name         = "acme/platform/api"
  vcs_provider = "github"
}

# Look a repo up by its internal ID, shown by `kosli get repo`
data "kosli_repo" "by_id" {
  id = "6f1c2a9e-3b7d-4c55-9e0a-1d2f3b4c5d6e"
}

output "api_repo_id" {
  description = "Internal ID of the repo, for use with kosli_repo_tags"
  value       = data.kosli_repo.api.id
}

output "repo_name_for_id" {
  description = "Name of the repo with the given ID"
  value       = data.kosli_repo.by_id.name
}
