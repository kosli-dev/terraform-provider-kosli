terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Minimal flow with only a name
resource "kosli_flow" "minimal" {
  name = "my-service"
}

# Flow with description
resource "kosli_flow" "with_description" {
  name        = "api-service"
  description = "CD pipeline for the API service"
}

# Flow with a YAML template defining trails and attestations
# The template can also be loaded from a file: template = file("template.yml")
resource "kosli_flow" "with_template" {
  name        = "backend-service"
  description = "Backend service CD pipeline with full attestation template"

  template = <<-YAML
version: 1
trail:
  attestations:
    - name: pull-request
      type: pull_request
    - name: unit-tests
      type: generic
  artifacts:
    - name: docker-image
      attestations:
        - name: sbom
          type: generic
        - name: security-scan
          type: snyk
YAML
}
