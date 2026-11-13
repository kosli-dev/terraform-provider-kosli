terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Find every GitHub repo under the acme/platform namespace
data "kosli_repos" "platform" {
  namespace    = "acme/platform"
  vcs_provider = "github"
}

# Tag each of them. Key on the repo ID: names can repeat across VCS providers.
resource "kosli_repo_tags" "platform" {
  for_each = { for r in data.kosli_repos.platform.repos : r.id => r }

  repo_id = each.key
  tags = {
    team = "platform"
    tier = "1"
  }
}

# Tag a single repo found by its name
data "kosli_repo" "billing" {
  name         = "acme/payments/billing"
  vcs_provider = "github"
}

resource "kosli_repo_tags" "billing" {
  repo_id = data.kosli_repo.billing.id
  tags = {
    team = "payments"
  }
}
