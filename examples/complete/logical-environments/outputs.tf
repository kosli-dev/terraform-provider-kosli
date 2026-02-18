# ============================================================================
# Physical Environment Outputs
# ============================================================================

output "physical_environments" {
  description = "Map of all physical environments created"
  value = {
    production = {
      k8s_us_east = kosli_environment.prod_k8s_us_east.name
      k8s_eu_west = kosli_environment.prod_k8s_eu_west.name
      ecs         = kosli_environment.prod_ecs.name
      lambda      = kosli_environment.prod_lambda.name
      s3          = kosli_environment.prod_s3.name
    }
    staging = {
      k8s = kosli_environment.staging_k8s.name
      ecs = kosli_environment.staging_ecs.name
    }
  }
}

# ============================================================================
# Logical Environment Outputs
# ============================================================================

output "logical_environments" {
  description = "Map of all logical environments and their included physical environments"
  value = {
    production_all = {
      name                  = kosli_logical_environment.production_all.name
      description           = kosli_logical_environment.production_all.description
      included_environments = kosli_logical_environment.production_all.included_environments
      environment_count     = length(kosli_logical_environment.production_all.included_environments)
    }
    production_by_region = {
      us_east = {
        name                  = kosli_logical_environment.production_us.name
        included_environments = kosli_logical_environment.production_us.included_environments
      }
      eu_west = {
        name                  = kosli_logical_environment.production_eu.name
        included_environments = kosli_logical_environment.production_eu.included_environments
      }
    }
    production_by_service = {
      kubernetes = {
        name                  = kosli_logical_environment.production_kubernetes.name
        included_environments = kosli_logical_environment.production_kubernetes.included_environments
      }
      containers = {
        name                  = kosli_logical_environment.production_containers.name
        included_environments = kosli_logical_environment.production_containers.included_environments
      }
      serverless = {
        name                  = kosli_logical_environment.production_serverless.name
        included_environments = kosli_logical_environment.production_serverless.included_environments
      }
    }
    staging_all = {
      name                  = kosli_logical_environment.staging_all.name
      included_environments = kosli_logical_environment.staging_all.included_environments
    }
  }
}

# ============================================================================
# Data Source Outputs
# ============================================================================

output "production_all_metadata" {
  description = "Metadata from production-all logical environment data source"
  value = {
    name                  = data.kosli_logical_environment.prod_all.name
    type                  = data.kosli_logical_environment.prod_all.type
    description           = data.kosli_logical_environment.prod_all.description
    included_environments = data.kosli_logical_environment.prod_all.included_environments
    last_modified_at      = data.kosli_logical_environment.prod_all.last_modified_at
    environment_count     = length(data.kosli_logical_environment.prod_all.included_environments)
  }
}

# ============================================================================
# Summary Statistics
# ============================================================================

output "environment_summary" {
  description = "Summary statistics of environments"
  value = {
    total_physical_environments = 7 # 5 prod + 2 staging
    total_logical_environments  = 8 # all combinations created
    production_aggregations = {
      all_prod       = length(kosli_logical_environment.production_all.included_environments)
      us_region      = length(kosli_logical_environment.production_us.included_environments)
      eu_region      = length(kosli_logical_environment.production_eu.included_environments)
      kubernetes     = length(kosli_logical_environment.production_kubernetes.included_environments)
      containers     = length(kosli_logical_environment.production_containers.included_environments)
      serverless     = length(kosli_logical_environment.production_serverless.included_environments)
    }
  }
}

# ============================================================================
# Useful Aggregations for Monitoring
# ============================================================================

output "all_production_environment_names" {
  description = "List of all production physical environment names"
  value       = kosli_logical_environment.production_all.included_environments
}

output "all_kubernetes_clusters" {
  description = "List of all production Kubernetes cluster names"
  value       = kosli_logical_environment.production_kubernetes.included_environments
}

output "serverless_components" {
  description = "List of all production serverless component names"
  value       = kosli_logical_environment.production_serverless.included_environments
}
