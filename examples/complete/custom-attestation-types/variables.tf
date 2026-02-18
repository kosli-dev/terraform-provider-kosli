variable "kosli_api_token" {
  description = "Kosli API token for authentication"
  type        = string
  sensitive   = true
}

variable "kosli_org" {
  description = "Kosli organization name"
  type        = string
}

variable "kosli_api_url" {
  description = "Kosli API URL (defaults to EU region)"
  type        = string
  default     = "https://app.kosli.com"
}
