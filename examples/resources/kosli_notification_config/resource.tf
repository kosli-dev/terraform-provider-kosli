terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Send API key expiry warnings to the platform team instead of the recipients
# Kosli derives by default. Set at least one of emails, slack_webhooks or
# webhooks; destroying the resource restores the derived recipients.
resource "kosli_notification_config" "api_key_expiry" {
  notification_type = "api_key_expiry"

  emails = [
    "platform-team@example.com",
    "security@example.com",
  ]

  # Webhook URLs are secrets: in real configurations pass them in through a
  # sensitive variable rather than committing them.
  slack_webhooks = ["https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXX"]
  webhooks       = ["https://ops.example.com/kosli/notifications"]
}
