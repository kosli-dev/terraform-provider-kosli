# Attach a policy to an environment.
# Both the policy and environment must exist before creating the attachment.

resource "kosli_policy_attachment" "example" {
  environment_name = kosli_environment.example.name
  policy_name      = kosli_policy.example.name
}
