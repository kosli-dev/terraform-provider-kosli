terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Basic K8S environment
resource "kosli_environment" "production_k8s" {
  name        = "production-k8s"
  type        = "K8S"
  description = "Production Kubernetes cluster"
}

# ECS environment with scaling
resource "kosli_environment" "staging_ecs" {
  name            = "staging-ecs"
  type            = "ECS"
  description     = "Staging ECS cluster"
  include_scaling = true
}

# S3 environment
resource "kosli_environment" "data_lake" {
  name        = "data-lake-s3"
  type        = "S3"
  description = "Data lake S3 bucket environment"
}

# Docker environment
resource "kosli_environment" "local_docker" {
  name = "local-docker"
  type = "docker"
}

# Server environment
resource "kosli_environment" "production_servers" {
  name            = "production-servers"
  type            = "server"
  description     = "Production bare-metal servers"
  include_scaling = false
}

# Lambda environment
resource "kosli_environment" "serverless_functions" {
  name        = "serverless-lambda"
  type        = "lambda"
  description = "AWS Lambda functions"
}