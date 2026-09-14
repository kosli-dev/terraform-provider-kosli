terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Action that fires when an environment becomes compliant or non-compliant
resource "kosli_action" "compliance_alerts" {
  name         = "compliance-alerts"
  environments = ["production-k8s"]
  triggers     = ["ON_NON_COMPLIANT_ENV", "ON_COMPLIANT_ENV"]
  webhook_url  = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXX"
}

# Action that fires when a running artifact becomes compliant or non-compliant,
# or gains new provenance (UI: "Artifact changed")
resource "kosli_action" "artifact_changed_alerts" {
  name         = "artifact-changed-alerts"
  environments = ["staging-ecs"]
  triggers     = ["ON_SCALED_ARTIFACT"]
  webhook_url  = "https://outlook.office.com/webhook/XXXX"
}

# Action that fires on artifact lifecycle events in the environment
resource "kosli_action" "artifact_lifecycle_alerts" {
  name         = "artifact-lifecycle-alerts"
  environments = ["staging-ecs"]
  triggers     = ["ON_STARTED_ARTIFACT", "ON_EXITED_ARTIFACT", "ON_ALLOWED_ARTIFACT"]
  webhook_url  = "https://outlook.office.com/webhook/XXXX"
}
