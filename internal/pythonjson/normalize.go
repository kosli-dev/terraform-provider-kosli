// Package pythonjson converts Python repr() string representations to valid JSON.
//
// The Kosli API returns type_schema in Python repr() format rather than valid JSON.
// This package provides normalization from Python format to RFC 7159 JSON.
package pythonjson

import "strings"

// Normalize converts a Python repr() string to valid JSON.
//
// It handles:
//   - Single-quoted strings → double-quoted (with proper escaping)
//   - Double-quoted strings (Python uses these when value contains single quotes) → preserved
//   - Python booleans (True/False) → JSON (true/false)
//   - Python None → JSON null
//   - Escape sequences within strings
//
// This is a client-side workaround. Ideally the API should return valid JSON.
func Normalize(pythonStr string) string {
	var result strings.Builder
	result.Grow(len(pythonStr))

	i := 0
	for i < len(pythonStr) {
		ch := pythonStr[i]

		switch ch {
		case '\'':
			// Single-quoted Python string → convert to double-quoted JSON string
			result.WriteByte('"')
			i++
			for i < len(pythonStr) {
				ch = pythonStr[i]
				if ch == '\\' && i+1 < len(pythonStr) {
					next := pythonStr[i+1]
					if next == '\'' {
						// \' in Python single-quoted string → literal ' (no escaping needed in JSON)
						result.WriteByte('\'')
						i += 2
					} else {
						// Other escape sequences pass through
						result.WriteByte('\\')
						result.WriteByte(next)
						i += 2
					}
				} else if ch == '\'' {
					// End of single-quoted string
					result.WriteByte('"')
					i++
					break
				} else if ch == '"' {
					// Double quote inside single-quoted string must be escaped for JSON
					result.WriteString("\\\"")
					i++
				} else {
					result.WriteByte(ch)
					i++
				}
			}

		case '"':
			// Double-quoted Python string — already valid JSON quoting, pass through
			result.WriteByte('"')
			i++
			for i < len(pythonStr) {
				ch = pythonStr[i]
				if ch == '\\' && i+1 < len(pythonStr) {
					result.WriteByte('\\')
					result.WriteByte(pythonStr[i+1])
					i += 2
				} else if ch == '"' {
					result.WriteByte('"')
					i++
					break
				} else {
					result.WriteByte(ch)
					i++
				}
			}

		case 'T':
			if matchKeyword(pythonStr, i, "True") {
				result.WriteString("true")
				i += 4
			} else {
				result.WriteByte(ch)
				i++
			}

		case 'F':
			if matchKeyword(pythonStr, i, "False") {
				result.WriteString("false")
				i += 5
			} else {
				result.WriteByte(ch)
				i++
			}

		case 'N':
			if matchKeyword(pythonStr, i, "None") {
				result.WriteString("null")
				i += 4
			} else {
				result.WriteByte(ch)
				i++
			}

		default:
			result.WriteByte(ch)
			i++
		}
	}

	return result.String()
}

// matchKeyword checks if the keyword at position i is a standalone Python keyword
// (not part of a longer identifier).
func matchKeyword(s string, i int, keyword string) bool {
	if !strings.HasPrefix(s[i:], keyword) {
		return false
	}
	if i > 0 && isIdentChar(s[i-1]) {
		return false
	}
	end := i + len(keyword)
	if end < len(s) && isIdentChar(s[end]) {
		return false
	}
	return true
}

// isIdentChar returns true if ch is a letter, digit, or underscore.
func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}
