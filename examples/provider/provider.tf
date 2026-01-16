terraform {
  required_providers {
    kosli = {
      source  = "kosli-dev/kosli"
      version = "~> 0.1"
    }
  }
}

# Configure the Kosli Provider
provider "kosli" {
  # API token (can also be set via KOSLI_API_TOKEN environment variable)
  api_token = var.kosli_api_token

  # Organization name (can also be set via KOSLI_ORG environment variable)
  org = var.kosli_org

  # Optional: API endpoint URL (defaults to https://app.kosli.com)
  # api_url = "https://app.us.kosli.com"  # Use US region

  # Optional: HTTP client timeout in seconds (defaults to 30)
  # timeout = 60
}

variable "kosli_api_token" {
  description = "Kosli API token"
  type        = string
  sensitive   = true
}

variable "kosli_org" {
  description = "Kosli organization name"
  type        = string
}
