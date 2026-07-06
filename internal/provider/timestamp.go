package provider

import (
	"math"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// unixToTime converts a unix timestamp with optional fractional seconds (as
// returned by the Kosli API) to a UTC time.
func unixToTime(seconds float64) time.Time {
	sec, frac := math.Modf(seconds)
	return time.Unix(int64(sec), int64(frac*1e9)).UTC()
}

// timestampToState renders an API unix timestamp as an RFC3339 UTC string
// attribute value. Zero renders as null — the API uses 0 for "never" (no
// expiry, never used).
func timestampToState(seconds float64) types.String {
	if seconds == 0 {
		return types.StringNull()
	}
	return types.StringValue(unixToTime(seconds).Format(time.RFC3339))
}

// timestampToRFC3339State is timestampToState for timetypes.RFC3339
// attributes (user-facing timestamp inputs with semantic equality).
func timestampToRFC3339State(seconds float64) timetypes.RFC3339 {
	if seconds == 0 {
		return timetypes.NewRFC3339Null()
	}
	return timetypes.NewRFC3339TimeValue(unixToTime(seconds))
}
