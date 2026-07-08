# List all active controls in the organization
list "kosli_control" "all" {
  provider = kosli
}

# List controls matching a search string, including full resource data
# (e.g. for `terraform query -generate-config-out=generated.tf`)
list "kosli_control" "sdlc" {
  provider         = kosli
  include_resource = true

  config {
    search = "SDLC"
  }
}

# Include archived controls in the results
list "kosli_control" "archived" {
  provider = kosli

  config {
    archived = true
  }
}
