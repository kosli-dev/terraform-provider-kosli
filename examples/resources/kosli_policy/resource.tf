terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Minimal policy requiring provenance for all artifacts
resource "kosli_policy" "minimal" {
  name = "basic-requirements"
  content = <<-YAML
    _schema: https://kosli.com/schemas/policy/environment/v1
    artifacts:
      provenance:
        required: true
  YAML
}

# Production policy with full compliance requirements
resource "kosli_policy" "production" {
  name        = "prod-requirements"
  description = "Compliance requirements for production environments"
  content     = <<-YAML
    _schema: https://kosli.com/schemas/policy/environment/v1
    artifacts:
      provenance:
        required: true
      trail-compliance:
        required: true
      attestations:
        - name: unit-test
          type: junit
        - name: dependency-scan
          type: "*"
  YAML
}
