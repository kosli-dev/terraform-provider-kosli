terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Action that fires on non-compliant environment events
resource "kosli_action" "compliance_alerts" {
  name         = "compliance-alerts"
  environments = ["production-k8s"]
  triggers     = ["ON_NON_COMPLIANT_ENV", "ON_COMPLIANT_ENV"]
  webhook_url  = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXX"
}

# Action that fires on scaling events
resource "kosli_action" "scaling_alerts" {
  name         = "scaling-alerts"
  environments = ["staging-ecs"]
  triggers     = ["ON_SCALED_ARTIFACT"]
  webhook_url  = "https://outlook.office.com/webhook/XXXX"
}
