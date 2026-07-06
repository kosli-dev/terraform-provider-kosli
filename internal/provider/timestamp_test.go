package provider

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTimestampToState(t *testing.T) {
	// Zero means "never" and maps to null.
	if v := timestampToState(0); !v.IsNull() {
		t.Errorf("expected null for 0, got %q", v.ValueString())
	}

	// Whole seconds render as RFC3339 UTC.
	want := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	if v := timestampToState(float64(want.Unix())); v.ValueString() != "2100-01-01T00:00:00Z" {
		t.Errorf("expected '2100-01-01T00:00:00Z', got %q", v.ValueString())
	}

	// Fractional seconds are truncated by the RFC3339 (second-precision) layout.
	if v := timestampToState(1234567890.5); v.ValueString() != "2009-02-13T23:31:30Z" {
		t.Errorf("expected '2009-02-13T23:31:30Z', got %q", v.ValueString())
	}
}

func TestTimestampToRFC3339State(t *testing.T) {
	if v := timestampToRFC3339State(0); !v.IsNull() {
		t.Errorf("expected null for 0, got %q", v.ValueString())
	}

	v := timestampToRFC3339State(4102444800)
	tm, diags := v.ValueRFC3339Time()
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !tm.Equal(time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected 2100-01-01T00:00:00Z, got %v", tm)
	}
}

func TestWholeSecondTimestampValidator(t *testing.T) {
	cases := []struct {
		value   types.String
		wantErr bool
	}{
		{types.StringValue("2100-01-01T00:00:00Z"), false},
		{types.StringValue("2100-01-01T10:00:00+01:00"), false},
		// Sub-second precision can never round-trip (API stores whole seconds)
		// and would force a replacement on every plan.
		{types.StringValue("2100-01-01T00:00:00.5Z"), true},
		{types.StringValue("2100-01-01T00:00:00.000000001Z"), true},
		// Malformed values are left to the timetypes.RFC3339 type to report.
		{types.StringValue("not-a-timestamp"), false},
		{types.StringNull(), false},
		{types.StringUnknown(), false},
	}

	for _, tc := range cases {
		req := validator.StringRequest{
			Path:        path.Root("expires_at"),
			ConfigValue: tc.value,
		}
		resp := &validator.StringResponse{}
		wholeSecondTimestampValidator{}.ValidateString(context.Background(), req, resp)

		if resp.Diagnostics.HasError() != tc.wantErr {
			t.Errorf("value %v: expected error=%t, got error=%t (%v)", tc.value, tc.wantErr, resp.Diagnostics.HasError(), resp.Diagnostics)
		}
	}
}
