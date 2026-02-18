terraform {
  required_providers {
    kosli = {
      source  = "kosli-dev/kosli"
      version = "~> 0.3"
    }
  }
}

provider "kosli" {
  api_token = var.kosli_api_token
  org       = var.kosli_org
  api_url   = var.kosli_api_url
}

# ============================================================================
# Physical Environments - Production
# ============================================================================

# Production Kubernetes environments across regions
resource "kosli_environment" "prod_k8s_us_east" {
  name        = "prod-k8s-us-east"
  type        = "K8S"
  description = "Production Kubernetes in US East"
}

resource "kosli_environment" "prod_k8s_eu_west" {
  name        = "prod-k8s-eu-west"
  type        = "K8S"
  description = "Production Kubernetes in EU West"
}

# Production ECS environment
resource "kosli_environment" "prod_ecs" {
  name        = "prod-ecs-us-east"
  type        = "ECS"
  description = "Production ECS cluster in US East"
}

# Production Lambda functions
resource "kosli_environment" "prod_lambda" {
  name        = "prod-lambda-us-east"
  type        = "lambda"
  description = "Production Lambda functions in US East"
}

# Production S3 bucket
resource "kosli_environment" "prod_s3" {
  name        = "prod-s3-data-lake"
  type        = "S3"
  description = "Production data lake S3 bucket"
}

# ============================================================================
# Physical Environments - Staging
# ============================================================================

resource "kosli_environment" "staging_k8s" {
  name        = "staging-k8s"
  type        = "K8S"
  description = "Staging Kubernetes cluster"
}

resource "kosli_environment" "staging_ecs" {
  name        = "staging-ecs"
  type        = "ECS"
  description = "Staging ECS cluster"
}

# ============================================================================
# Logical Environment - All Production
# ============================================================================

# Aggregate all production environments for unified visibility
resource "kosli_logical_environment" "production_all" {
  name        = "production-all"
  description = "All production environments across regions and services"

  included_environments = [
    kosli_environment.prod_k8s_us_east.name,
    kosli_environment.prod_k8s_eu_west.name,
    kosli_environment.prod_ecs.name,
    kosli_environment.prod_lambda.name,
    kosli_environment.prod_s3.name,
  ]
}

# ============================================================================
# Logical Environment - Production by Region
# ============================================================================

# Production US environments
resource "kosli_logical_environment" "production_us" {
  name        = "production-us-east"
  description = "All production environments in US East region"

  included_environments = [
    kosli_environment.prod_k8s_us_east.name,
    kosli_environment.prod_ecs.name,
    kosli_environment.prod_lambda.name,
  ]
}

# Production EU environments
resource "kosli_logical_environment" "production_eu" {
  name        = "production-eu-west"
  description = "All production environments in EU West region"

  included_environments = [
    kosli_environment.prod_k8s_eu_west.name,
  ]
}

# ============================================================================
# Logical Environment - Production by Service Type
# ============================================================================

# All Kubernetes production environments
resource "kosli_logical_environment" "production_kubernetes" {
  name        = "production-kubernetes-all"
  description = "All production Kubernetes clusters across regions"

  included_environments = [
    kosli_environment.prod_k8s_us_east.name,
    kosli_environment.prod_k8s_eu_west.name,
  ]
}

# All container-based production environments (K8S + ECS)
resource "kosli_logical_environment" "production_containers" {
  name        = "production-containers"
  description = "All container-based production environments"

  included_environments = [
    kosli_environment.prod_k8s_us_east.name,
    kosli_environment.prod_k8s_eu_west.name,
    kosli_environment.prod_ecs.name,
  ]
}

# All serverless production environments
resource "kosli_logical_environment" "production_serverless" {
  name        = "production-serverless"
  description = "All serverless production environments (Lambda + S3)"

  included_environments = [
    kosli_environment.prod_lambda.name,
    kosli_environment.prod_s3.name,
  ]
}

# ============================================================================
# Logical Environment - All Staging
# ============================================================================

resource "kosli_logical_environment" "staging_all" {
  name        = "staging-all"
  description = "All staging environments for pre-production testing"

  included_environments = [
    kosli_environment.staging_k8s.name,
    kosli_environment.staging_ecs.name,
  ]
}

# ============================================================================
# Query Existing Logical Environment
# ============================================================================

# Query the production-all logical environment to use its configuration
data "kosli_logical_environment" "prod_all" {
  name = kosli_logical_environment.production_all.name

  depends_on = [kosli_logical_environment.production_all]
}

# ============================================================================
# Dynamic Logical Environment Based on Data Source
# ============================================================================

# Create a disaster recovery logical environment that mirrors production
resource "kosli_logical_environment" "disaster_recovery" {
  name        = "disaster-recovery-reference"
  description = "Reference to production environments for DR planning (based on: ${data.kosli_logical_environment.prod_all.name})"

  # Note: In a real scenario, you would reference actual DR environments
  # This demonstrates using data source outputs
  included_environments = data.kosli_logical_environment.prod_all.included_environments
}
