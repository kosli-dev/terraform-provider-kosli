terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# First, create physical environments that will be aggregated
resource "kosli_environment" "production_k8s" {
  name        = "production-k8s"
  type        = "K8S"
  description = "Production Kubernetes cluster"
}

resource "kosli_environment" "production_ecs" {
  name        = "production-ecs"
  type        = "ECS"
  description = "Production ECS cluster"
}

resource "kosli_environment" "production_lambda" {
  name        = "production-lambda"
  type        = "lambda"
  description = "Production Lambda functions"
}

# Basic logical environment aggregating production environments
resource "kosli_logical_environment" "production_all" {
  name        = "production-aggregate"
  description = "Aggregates all production environments for unified visibility"

  included_environments = [
    kosli_environment.production_k8s.name,
    kosli_environment.production_ecs.name,
    kosli_environment.production_lambda.name,
  ]
}

# Logical environment with just two environments
resource "kosli_logical_environment" "cloud_services" {
  name        = "cloud-services"
  description = "All cloud-based services"

  included_environments = [
    kosli_environment.production_ecs.name,
    kosli_environment.production_lambda.name,
  ]
}

# Minimal logical environment with empty list (can be populated later)
resource "kosli_logical_environment" "future_environments" {
  name                  = "future-environments"
  included_environments = []
}

# Logical environment without description (optional)
resource "kosli_logical_environment" "simple" {
  name = "simple-aggregate"

  included_environments = [
    kosli_environment.production_k8s.name,
  ]
}

# Logical environment with tags
resource "kosli_logical_environment" "tagged" {
  name        = "production-tagged"
  description = "Tagged production logical environment"

  included_environments = [
    kosli_environment.production_k8s.name,
    kosli_environment.production_ecs.name,
  ]

  tags = {
    managed-by  = "terraform"
    environment = "production"
    team        = "platform"
  }
}
