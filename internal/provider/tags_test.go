package provider

import "testing"

func TestTitleCase(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"environment", "Environment"},
		{"flow", "Flow"},
		{"a", "A"},
		{"already", "Already"},
	}
	for _, tt := range tests {
		if got := titleCase(tt.in); got != tt.want {
			t.Errorf("titleCase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
