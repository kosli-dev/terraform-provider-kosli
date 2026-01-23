terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Query an existing environment
data "kosli_environment" "production" {
  name = "production-k8s"
}

# Use the data source to create a similar environment
resource "kosli_environment" "staging" {
  name        = "staging-k8s"
  type        = data.kosli_environment.production.type
  description = "Staging environment similar to ${data.kosli_environment.production.name}"
}

# Reference environment metadata for monitoring
output "production_last_modified" {
  description = "Timestamp of when production environment was last modified"
  value       = data.kosli_environment.production.last_modified_at
}

output "production_last_reported" {
  description = "Timestamp of when production environment last reported a snapshot"
  value       = data.kosli_environment.production.last_reported_at
}

output "production_type" {
  description = "Type of the production environment"
  value       = data.kosli_environment.production.type
}

output "production_includes_scaling" {
  description = "Whether production environment includes scaling events"
  value       = data.kosli_environment.production.include_scaling
}

# Conditional logic based on environment metadata
locals {
  # Check if environment has never reported a snapshot
  needs_attention = data.kosli_environment.production.last_reported_at == null
}

output "production_needs_attention" {
  description = "Whether production environment needs attention (never reported)"
  value       = local.needs_attention
}
