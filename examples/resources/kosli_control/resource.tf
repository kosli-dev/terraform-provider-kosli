terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# Control requiring binary provenance for production artifacts
resource "kosli_control" "binary_provenance" {
  identifier  = "SDLC-001"
  name        = "Binary provenance"
  description = "All production artifacts must have build provenance attestations"

  links = {
    docs = "https://example.com/sdlc/binary-provenance"
  }
}

# Minimal control with only the required attributes
resource "kosli_control" "peer_review" {
  identifier = "SDLC-002"
  name       = "Peer review"
}
