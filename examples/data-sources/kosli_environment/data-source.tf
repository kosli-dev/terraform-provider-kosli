terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Create an environment with tags
resource "kosli_environment" "production" {
  name        = "production-k8s"
  type        = "K8S"
  description = "Production Kubernetes cluster"
  tags = {
    managed-by  = "terraform"
    environment = "production"
    team        = "platform"
  }
}

# Query the environment via data source to read back its attributes and tags
data "kosli_environment" "production" {
  name = kosli_environment.production.name
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

# Access tags applied to the environment
output "production_tags" {
  description = "Tags applied to the production environment"
  value       = data.kosli_environment.production.tags
}

# Check if a specific tag exists
output "production_managed_by" {
  description = "Who manages the production environment (from tags)"
  value       = try(data.kosli_environment.production.tags["managed-by"], "unknown")
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
