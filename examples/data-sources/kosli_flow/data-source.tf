terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Query an existing flow
data "kosli_flow" "example" {
  name = "my-application-flow"
}

# Create a new flow reusing the template from an existing one
resource "kosli_flow" "copy" {
  name        = "my-application-flow-copy"
  description = data.kosli_flow.example.description
  template    = data.kosli_flow.example.template
}

output "flow_name" {
  description = "The name of the flow"
  value       = data.kosli_flow.example.name
}

output "flow_template" {
  description = "The YAML template of the flow"
  value       = data.kosli_flow.example.template
}
