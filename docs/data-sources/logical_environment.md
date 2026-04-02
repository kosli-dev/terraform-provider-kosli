---
page_title: "kosli_logical_environment Data Source - terraform-provider-kosli"
subcategory: ""
description: |-
  Fetches details of an existing Kosli logical environment. Use this data source to reference logical environments and access their aggregated physical environments.
---

# Data Source: kosli_logical_environment

Fetches details of an existing Kosli logical environment. Use this data source to reference logical environments and access their aggregated physical environments.

Use this data source to query existing logical environments in Kosli. This is useful for:

- **Referencing Metadata**: Access `last_modified_at` timestamps and other computed attributes
- **Cross-Stack References**: Reference logical environments created outside Terraform
- **Dynamic Configuration**: Use existing logical environment configurations to create variants
- **Validation**: Verify logical environments exist before referencing them
- **Monitoring**: Create conditional logic based on logical environment state

## Example Usage

```terraform
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

output "production_tags" {
  description = "Tags on the production logical environment"
  value       = data.kosli_logical_environment.production.tags
}
```

## Type Validation

-> **Note:** This data source validates that the queried environment is of type `logical`. Attempting to query a physical environment will result in an error. Use the `kosli_environment` data source for physical environments instead.

## Use Cases

### Reference Metadata

Query logical environment metadata for monitoring or conditional logic:

```terraform
data "kosli_logical_environment" "production" {
  name = "production-all"
}

output "production_last_modified" {
  value = data.kosli_logical_environment.production.last_modified_at
}

output "production_environment_count" {
  value = length(data.kosli_logical_environment.production.included_environments)
}
```

### Create Variants

Use an existing logical environment as a template for creating similar ones:

```terraform
data "kosli_logical_environment" "production" {
  name = "production-all"
}

resource "kosli_logical_environment" "staging" {
  name        = "staging-all"
  description = "Staging version of ${data.kosli_logical_environment.production.name}"

  # Reuse the same environment structure
  included_environments = data.kosli_logical_environment.production.included_environments
}
```

### Cross-Stack References

Reference logical environments created in other Terraform workspaces or outside Terraform:

```terraform
data "kosli_logical_environment" "shared_production" {
  name = "production-all" # Created in infrastructure workspace
}

# Use in application deployment workspace
locals {
  production_environments = data.kosli_logical_environment.shared_production.included_environments
}
```

### Conditional Logic

Create conditional logic based on logical environment configuration:

```terraform
data "kosli_logical_environment" "production" {
  name = "production-all"
}

locals {
  # Check if environment is empty
  is_empty = length(data.kosli_logical_environment.production.included_environments) == 0

  # Check if it includes a specific environment
  includes_k8s = contains(
    data.kosli_logical_environment.production.included_environments,
    "production-k8s"
  )
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the logical environment to query.

### Read-Only

- `description` (String) The description of the logical environment.
- `included_environments` (List of String) List of physical environment names aggregated by this logical environment.
- `last_modified_at` (Number) Unix timestamp (with fractional seconds) of when the logical environment was last modified.
- `tags` (Map of String) Key-value pairs tagging the logical environment.
- `type` (String) The environment type (always `logical` for logical environments).
