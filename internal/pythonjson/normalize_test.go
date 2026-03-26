package pythonjson

import (
	"encoding/json"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		skipJSON bool // skip JSON validation for non-JSON test cases
	}{
		{
			name:     "object with boolean values",
			input:    "{'enabled': True, 'disabled': False, 'value': None}",
			expected: `{"enabled": true, "disabled": false, "value": null}`,
		},
		{
			name:     "array with boolean values",
			input:    "{'flags': [True, False, None]}",
			expected: `{"flags": [true, false, null]}`,
		},
		{
			name:     "nested structures",
			input:    "{'config': {'debug': True, 'items': [False, None]}}",
			expected: `{"config": {"debug": true, "items": [false, null]}}`,
		},
		{
			name:     "boolean as first array element",
			input:    "[True, False, 'text']",
			expected: `[true, false, "text"]`,
		},
		{
			name:     "mixed contexts",
			input:    "{'x': True, 'y': [False, None], 'z': {'nested': True}}",
			expected: `{"x": true, "y": [false, null], "z": {"nested": true}}`,
		},
		{
			name:     "double-quoted string with embedded single quotes",
			input:    `{'title': "CycloneDX's format", 'type': 'object'}`,
			expected: `{"title": "CycloneDX's format", "type": "object"}`,
		},
		{
			name:     "double-quoted string with embedded single-quoted word",
			input:    `{'description': "use 'CycloneDX' for BOM", 'version': '1.0'}`,
			expected: `{"description": "use 'CycloneDX' for BOM", "version": "1.0"}`,
		},
		{
			name:     "single-quoted string with embedded double quotes",
			input:    `{'pattern': 'value "quoted" here', 'type': 'string'}`,
			expected: `{"pattern": "value \"quoted\" here", "type": "string"}`,
		},
		{
			name:     "mixed quoting with Python booleans",
			input:    `{'title': "it's required", 'required': True, 'nullable': False}`,
			expected: `{"title": "it's required", "required": true, "nullable": false}`,
		},
		{
			name:     "escaped single quote in single-quoted string",
			input:    `{'desc': 'it\'s here', 'type': 'string'}`,
			expected: `{"desc": "it's here", "type": "string"}`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no strings just structure",
			input:    "[True, False, None, 42]",
			expected: "[true, false, null, 42]",
		},
		{
			name:     "keyword-like substrings in strings are not converted",
			input:    "{'key': 'TrueBlue', 'other': 'NonEmpty'}",
			expected: `{"key": "TrueBlue", "other": "NonEmpty"}`,
		},
		{
			name:     "keyword-like bare identifiers are not converted",
			input:    "[TrueBlue, True, FalsePositive, False, NoneType, None]",
			expected: `[TrueBlue, true, FalsePositive, false, NoneType, null]`,
			skipJSON: true, // bare identifiers aren't valid JSON — just testing keyword boundary logic
		},
		{
			name:     "escape sequences pass through",
			input:    `{'msg': 'line1\nline2\ttabbed'}`,
			expected: `{"msg": "line1\nline2\ttabbed"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("Normalize() = %q, expected %q", result, tt.expected)
			}

			// Verify non-empty results are valid JSON
			if result != "" && !tt.skipJSON {
				var jsonObj any
				if err := json.Unmarshal([]byte(result), &jsonObj); err != nil {
					t.Errorf("Result is not valid JSON: %v", err)
				}
			}
		})
	}
}
