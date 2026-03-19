terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Attach a policy to an environment.
# Both the policy and environment must exist before creating the attachment.
resource "kosli_policy_attachment" "example" {
  environment_name = "my-environment"
  policy_name      = "my-policy"
}
