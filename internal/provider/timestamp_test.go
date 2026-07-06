package provider

import (
	"testing"
	"time"
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
