# Provider requirements and configuration for the query example.
# `terraform query` resolves the `provider = kosli` references in
# list-resource.tfquery.hcl against this provider configuration; credentials
# come from the KOSLI_API_TOKEN / KOSLI_ORG environment variables.
terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

provider "kosli" {}
