terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Query an existing logical environment
data "kosli_logical_environment" "production" {
  name = "production-aggregate"
}

# Use the data source to create a similar logical environment
resource "kosli_logical_environment" "staging" {
  name                  = "staging-aggregate"
  description           = "Staging version of ${data.kosli_logical_environment.production.name}"
  included_environments = data.kosli_logical_environment.production.included_environments
}

# Reference logical environment metadata
output "production_name" {
  description = "Name of the production logical environment"
  value       = data.kosli_logical_environment.production.name
}

output "production_type" {
  description = "Type of the environment (should be 'logical')"
  value       = data.kosli_logical_environment.production.type
}

output "production_description" {
  description = "Description of the production logical environment"
  value       = data.kosli_logical_environment.production.description
}

output "production_environments" {
  description = "List of physical environments included in production aggregate"
  value       = data.kosli_logical_environment.production.included_environments
}

output "production_last_modified" {
  description = "Timestamp of when production logical environment was last modified"
  value       = data.kosli_logical_environment.production.last_modified_at
}

# Count how many environments are aggregated
output "production_environment_count" {
  description = "Number of environments aggregated in production"
  value       = length(data.kosli_logical_environment.production.included_environments)
}

# Conditional logic based on aggregation
locals {
  # Check if logical environment is empty (no included environments)
  is_empty = length(data.kosli_logical_environment.production.included_environments) == 0

  # Check if it includes a specific environment
  includes_k8s = contains(
    data.kosli_logical_environment.production.included_environments,
    "production-k8s"
  )
}

output "production_is_empty" {
  description = "Whether production logical environment has no included environments"
  value       = local.is_empty
}

output "production_includes_k8s" {
  description = "Whether production aggregates a K8S environment"
  value       = local.includes_k8s
}
