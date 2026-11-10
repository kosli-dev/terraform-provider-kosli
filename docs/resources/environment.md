---
page_title: "kosli_environment Resource - terraform-provider-kosli"
subcategory: ""
description: |-
  Manages a Kosli environment. Environments represent deployment targets where artifacts are deployed. Supports physical environment types: K8S, ECS, S3, docker, server, lambda, azure-apps, and cloud-run.
  ~> Note: Environment policies will be available in a future release. For querying environment metadata such as last_modified_at, last_reported_at, and archived status, use the kosli_environment data source.
  ~> Note: Destroying this resource archives the environment in Kosli. Archiving requires an API token with the Admin role; a Member token can create and update environments but terraform destroy fails with 403 Forbidden.
  ~> Note: Changing name renames the environment in place (see kosli rename environment): it keeps its identity and history, and the old name keeps resolving to the renamed environment until another environment is created with, or renamed to, that name.
---

# Resource: kosli_environment

Manages a Kosli environment. Environments represent deployment targets where artifacts are deployed. Supports physical environment types: K8S, ECS, S3, docker, server, lambda, azure-apps, and cloud-run.

~> **Note:** Environment policies will be available in a future release. For querying environment metadata such as `last_modified_at`, `last_reported_at`, and `archived` status, use the `kosli_environment` data source.

~> **Note:** Destroying this resource archives the environment in Kosli. Archiving requires an API token with the **Admin** role; a Member token can create and update environments but `terraform destroy` fails with `403 Forbidden`.

~> **Note:** Changing `name` renames the environment in place (see `kosli rename environment`): it keeps its identity and history, and the old name keeps resolving to the renamed environment until another environment is created with, or renamed to, that name.

Kosli environments track deployments and provide visibility into what's running in your infrastructure. Physical environments represent actual runtime locations such as:

- **K8S**: Kubernetes clusters
- **ECS**: Amazon Elastic Container Service clusters
- **S3**: Amazon S3 buckets
- **docker**: Docker containers
- **server**: Bare-metal or VM servers
- **lambda**: AWS Lambda functions
- **azure-apps**: Azure App Service and Function apps
- **cloud-run**: Google Cloud Run services

-> **Note:** For aggregating multiple physical environments into logical groups, use the `kosli_logical_environment` resource.

~> **Note:** Environment tags are managed through a separate Kosli API and are not included in this Terraform resource.

~> **Note:** Environment policies will be available in a future release as a separate resource (`kosli_environment_policy`).

## Example Usage

```terraform
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

# ECS environment
resource "kosli_environment" "staging_ecs" {
  name        = "staging-ecs"
  type        = "ECS"
  description = "Staging ECS cluster"
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
  name        = "production-servers"
  type        = "server"
  description = "Production bare-metal servers"
}

# Lambda environment
resource "kosli_environment" "serverless_functions" {
  name        = "serverless-lambda"
  type        = "lambda"
  description = "AWS Lambda functions"
}

# Azure apps environment
resource "kosli_environment" "azure_apps" {
  name        = "production-azure-apps"
  type        = "azure-apps"
  description = "Azure App Service and Function apps"
}

# Cloud Run environment
resource "kosli_environment" "cloud_run" {
  name        = "production-cloud-run"
  type        = "cloud-run"
  description = "Google Cloud Run services"
}

# K8S environment with tags for IaC traceability
resource "kosli_environment" "tagged" {
  name        = "production-k8s-tagged"
  type        = "K8S"
  description = "Production cluster managed by Terraform"
  tags = {
    managed-by  = "terraform"
    environment = "production"
    team        = "platform"
  }
}
```

## Environment Types

The Kosli API validates `type`. The known physical environment types are:

- `K8S` - Kubernetes clusters
- `ECS` - Amazon Elastic Container Service
- `S3` - Amazon S3 buckets
- `docker` - Docker containers
- `server` - Bare-metal or VM servers
- `lambda` - AWS Lambda functions
- `azure-apps` - Azure App Service and Function apps
- `cloud-run` - Google Cloud Run services

A `type` of `logical` produces a warning: use the `kosli_logical_environment` resource for logical environments.

## Import

Environments can be imported using their name:

```shell
#!/bin/bash

# Import an existing environment by name
terraform import kosli_environment.production_k8s production-k8s

# Import multiple environments
terraform import kosli_environment.staging_ecs staging-ecs
terraform import kosli_environment.data_lake data-lake-s3
```

## Renaming Environments

Changing `name` renames the environment in place, the same as `kosli rename environment`. The environment keeps its identity, snapshot history and tags, and the plan shows an update rather than a replacement.

After a rename the old name still resolves to the renamed environment for both reads and writes, so `kosli snapshot` commands that still use the old name report into it. The old name stops resolving once another environment is created with, or renamed to, that name.

Terraform has no ordering between two resources that are both renamed in one plan. If one resource takes the name another is giving up, such as `a` to `b` together with `b` to `c`, or a swap, the rename that runs first can hit a name that still exists. Apply chained renames in separate steps.

~> **Note:** `type` cannot be changed in Kosli. A plan that changes `type` still replaces the environment, even when `name` changes in the same plan.

## Destroying Environments

Kosli never hard-deletes environments. When you run `terraform destroy` or remove a `kosli_environment` from your configuration, the provider archives the environment: it disappears from the environment list in Kosli, but its snapshot history is kept.

~> **Note:** Archiving an environment requires a Kosli API token with the **Admin** role. A token with the Member role can create and update environments but cannot archive them, so `terraform destroy` fails with `403 Forbidden` and the resource stays in state. If you cannot use an Admin token and only want Terraform to stop managing the environment, run `terraform state rm kosli_environment.<name>` instead; the environment then remains active in Kosli. See the [provider documentation](https://registry.terraform.io/providers/kosli-dev/kosli/latest/docs#required-permissions) for how to set up a service account with the right permissions.

## Monitoring Environments

For querying environment metadata such as `last_modified_at` and `last_reported_at` timestamps, use the `kosli_environment` data source. This is useful for monitoring and creating conditional logic based on environment state.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the environment. Must be unique within the organization. Changing this renames the environment in place, keeping its snapshot history; the old name keeps resolving to the renamed environment until another environment is created with, or renamed to, that name.
- `type` (String) Type of the environment. Known values: `K8S`, `ECS`, `S3`, `docker`, `server`, `lambda`, `azure-apps`, `cloud-run`. The Kosli API validates the value. For `logical`, use the `kosli_logical_environment` resource. Changing this will force recreation of the resource.

### Optional

- `description` (String) Description of the environment. Explains the purpose and characteristics of this deployment target.
- `tags` (Map of String) Key-value pairs to tag the environment.
