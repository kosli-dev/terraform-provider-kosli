package kosli.controls.artifact_provenance

import rego.v1

default allow := false

allow if {
	input.artifact_fingerprint != ""
	input.is_compliant == true
}

violations contains msg if {
	not input.artifact_fingerprint
	msg := "artifact_fingerprint must be present on all compliance decisions"
}
