package provider

import (
	"context"
	"math"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

// wholeSecondTimestampValidator rejects RFC3339 values with sub-second
// precision. The API stores whole unix seconds, so a fractional-second value
// could never round-trip: the refreshed state would differ from config by the
// fraction and force a replacement on every plan.
type wholeSecondTimestampValidator struct{}

func (wholeSecondTimestampValidator) Description(context.Context) string {
	return "timestamp must not have sub-second precision"
}

func (v wholeSecondTimestampValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (wholeSecondTimestampValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	t, err := time.Parse(time.RFC3339, req.ConfigValue.ValueString())
	if err != nil {
		// Malformed values are reported by the timetypes.RFC3339 type itself.
		return
	}
	if t.Nanosecond() != 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Unsupported Timestamp Precision",
			"Sub-second precision is not supported; the API stores whole unix seconds. Remove the fractional seconds.",
		)
	}
}
