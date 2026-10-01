terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Query where API key expiry warnings are sent
data "kosli_notification_config" "api_key_expiry" {
  notification_type = "api_key_expiry"
}

output "api_key_expiry_configured" {
  description = "Whether the organization overrides the recipients Kosli derives"
  value       = data.kosli_notification_config.api_key_expiry.configured
}

output "api_key_expiry_emails" {
  description = "Email addresses API key expiry warnings are sent to"
  value       = data.kosli_notification_config.api_key_expiry.emails
}

output "api_key_expiry_last_notified_at" {
  description = "When an API key expiry warning was last sent (RFC3339)"
  value       = data.kosli_notification_config.api_key_expiry.last_notified_at
}
