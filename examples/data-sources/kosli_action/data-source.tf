terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Query an existing action
data "kosli_action" "compliance_alerts" {
  name = "compliance-alerts"
}

output "action_number" {
  description = "Server-assigned number of the action"
  value       = data.kosli_action.compliance_alerts.number
}

output "action_environments" {
  description = "Environments monitored by this action"
  value       = data.kosli_action.compliance_alerts.environments
}
