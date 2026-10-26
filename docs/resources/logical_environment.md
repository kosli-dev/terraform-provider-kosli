---
page_title: "kosli_logical_environment Resource - terraform-provider-kosli"
subcategory: ""
description: |-
  Manages a Kosli logical environment. Logical environments aggregate multiple physical environments for organizational purposes.
  ~> Important: Logical environments can ONLY contain physical environments (K8S, ECS, S3, docker, server, lambda), not other logical environments. Attempting to include a logical environment will result in an error from the Kosli API. See ADR-004 https://github.com/kosli-dev/terraform-provider-kosli/blob/main/adrs/004-logical-environment-validation.md for validation strategy.
  ~> Note: This resource manages logical environment configuration only. For querying environment metadata such as last_modified_at and archived status, use the kosli_logical_environment data source.
  ~> Note: Destroying this resource archives the logical environment in Kosli. Archiving requires an API token with the Admin role; a Member token can create and update logical environments but terraform destroy fails with 403 Forbidden.
---

# Resource: kosli_logical_environment

Manages a Kosli logical environment. Logical environments aggregate multiple physical environments for organizational purposes.

~> **Important:** Logical environments can ONLY contain physical environments (K8S, ECS, S3, docker, server, lambda), not other logical environments. Attempting to include a logical environment will result in an error from the Kosli API. See [ADR-004](https://github.com/kosli-dev/terraform-provider-kosli/blob/main/adrs/004-logical-environment-validation.md) for validation strategy.

~> **Note:** This resource manages logical environment configuration only. For querying environment metadata such as `last_modified_at` and `archived` status, use the `kosli_logical_environment` data source.

~> **Note:** Destroying this resource archives the logical environment in Kosli. Archiving requires an API token with the **Admin** role; a Member token can create and update logical environments but `terraform destroy` fails with `403 Forbidden`.

Logical environments in Kosli aggregate multiple physical environments for organizational purposes, providing:

- **Unified Visibility**: View compliance status across multiple environments at once
- **Flexible Grouping**: Organize environments by region, service type, tier, or team
- **Simplified Reporting**: Generate compliance reports for logical groupings
- **Team Organization**: Allow different teams to focus on specific environment groups

## Physical Environments Only

~> **Important:** Logical environments can ONLY contain physical environments (K8S, ECS, S3, docker, server, lambda), not other logical environments. Attempting to include a logical environment will result in an API error. See [ADR-004](https://github.com/kosli-dev/terraform-provider-kosli/blob/main/adrs/004-logical-environment-validation.md) for the validation strategy.

## Example Usage

```terraform
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
```

## Complete Example

For a comprehensive example showing logical environments aggregating physical environments by region, service type, and tier, see the [complete logical environments example](https://github.com/kosli-dev/terraform-provider-kosli/tree/main/examples/complete/logical-environments).

## Common Use Cases

### By Environment Tier

Aggregate all production or staging environments for unified compliance reporting:

```terraform
resource "kosli_logical_environment" "production_all" {
  name        = "production-all"
  description = "All production environments"

  included_environments = [
    kosli_environment.prod_k8s.name,
    kosli_environment.prod_ecs.name,
    kosli_environment.prod_lambda.name,
  ]
}
```

### By Geographic Region

Group environments by region for regional compliance or disaster recovery:

```terraform
resource "kosli_logical_environment" "production_us_east" {
  name        = "production-us-east"
  description = "All production environments in US East"

  included_environments = [
    kosli_environment.prod_k8s_us_east.name,
    kosli_environment.prod_ecs_us_east.name,
  ]
}
```

### By Service Type

Organize environments by technology stack or service type:

```terraform
resource "kosli_logical_environment" "all_kubernetes" {
  name        = "all-kubernetes-clusters"
  description = "All Kubernetes clusters across regions"

  included_environments = [
    kosli_environment.k8s_us_east.name,
    kosli_environment.k8s_eu_west.name,
    kosli_environment.k8s_ap_south.name,
  ]
}
```

## Empty Logical Environments

Logical environments can be created with empty `included_environments` lists and populated later:

```terraform
resource "kosli_logical_environment" "future_environments" {
  name                  = "planned-expansion"
  description           = "Placeholder for future environments"
  included_environments = []
}
```

## Import

Logical environments can be imported using their name:

```shell
#!/bin/bash

# Import an existing logical environment by name
terraform import kosli_logical_environment.production_all production-aggregate

# Import multiple logical environments
terraform import kosli_logical_environment.cloud_services cloud-services
terraform import kosli_logical_environment.future_environments future-environments
```

## Destroying Logical Environments

Kosli never hard-deletes environments. When you run `terraform destroy` or remove a `kosli_logical_environment` from your configuration, the provider archives the logical environment. The physical environments it included are not affected.

~> **Note:** Archiving an environment requires a Kosli API token with the **Admin** role. A token with the Member role can create and update logical environments but cannot archive them, so `terraform destroy` fails with `403 Forbidden` and the resource stays in state. If you cannot use an Admin token and only want Terraform to stop managing the logical environment, run `terraform state rm kosli_logical_environment.<name>` instead; the logical environment then remains active in Kosli. See the [provider documentation](https://registry.terraform.io/providers/kosli-dev/kosli/latest/docs#required-permissions) for how to set up a service account with the right permissions.

## Querying Metadata

~> **Note:** This resource manages logical environment configuration only. For querying environment metadata such as `last_modified_at` and `archived` status, use the `kosli_logical_environment` data source.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `included_environments` (List of String) List of physical environment names to aggregate. Only physical environments are allowed (K8S, ECS, S3, docker, server, lambda). Can be empty.
- `name` (String) Name of the logical environment. Must be unique within the organization. Changing this will force recreation of the resource.

### Optional

- `description` (String) Description of the logical environment. Explains the purpose and aggregation strategy.
- `tags` (Map of String) Key-value pairs to tag the logical environment.

### Read-Only

- `type` (String) Type of the environment. Always set to `logical` (computed by provider, not user-configurable).
