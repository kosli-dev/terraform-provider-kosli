terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Configure the Kosli Provider
# Authentication via environment variables (recommended)
provider "kosli" {
  # API token - set via KOSLI_API_TOKEN environment variable
  # Organization name - set via KOSLI_ORG environment variable

  # Optional: API endpoint URL (defaults to https://app.kosli.com)
  # api_url = "https://app.us.kosli.com"  # Use US region

  # Optional: HTTP client timeout in seconds (defaults to 30)
  # timeout = 60
}
