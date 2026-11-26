package policy

import rego.v1

# Kosli wraps the trail under input.trail and type-checks the policy against
# the input schema, so a reference to a field the schema lacks is rejected.

default allow := false

allow if count(input.trail.compliance_status.artifacts_statuses) > 0

violations contains "the trail has no artifacts" if {
	count(input.trail.compliance_status.artifacts_statuses) == 0
}
