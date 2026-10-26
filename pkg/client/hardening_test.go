package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{"https", "https://app.kosli.com", ""},
		{"https with path", "https://kosli.example.com/proxy/", ""},
		{"http localhost", "http://localhost:8080", ""},
		{"http localhost subdomain", "http://api.localhost:8080", ""},
		{"http ipv4 loopback", "http://127.0.0.1:9999", ""},
		{"http ipv6 loopback", "http://[::1]:9999", ""},
		{"empty", "", "base URL cannot be empty"},
		{"http remote", "http://app.kosli.com", "must use https"},
		{"http private ip", "http://10.0.0.5", "must use https"},
		{"ftp", "ftp://app.kosli.com", "must use https"},
		{"no scheme", "app.kosli.com", "must include a scheme and host"},
		{"scheme only", "https://", "must include a scheme and host"},
		{"unparseable", "https://app.kosli.com/%zz", "not a valid URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBaseURL(tt.url)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateBaseURL(%q) = %v, want nil", tt.url, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateBaseURL(%q) = %v, want error containing %q", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestNewClient_RejectsPlaintextBaseURL(t *testing.T) {
	_, err := NewClient("test-token", "test-org", WithBaseURL("http://app.kosli.com"))
	if err == nil || !strings.Contains(err.Error(), "must use https") {
		t.Fatalf("expected https error, got %v", err)
	}
}

func TestParseResponse_RejectsOversizedBody(t *testing.T) {
	big := strings.Repeat("x", MaxResponseBodySize+1)
	resp := &http.Response{Body: io.NopCloser(strings.NewReader(`"` + big + `"`))}

	var out string
	err := ParseResponse(resp, &out)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected size error, got %v", err)
	}
}

func TestParseErrorResponse_TruncatesOversizedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, strings.Repeat("y", maxErrorBodySize*4))
	}))
	defer server.Close()

	c, err := NewClient("token", "org", WithBaseURL(server.URL), WithRetryPolicy(0, time.Millisecond, time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Get(context.Background(), "/x")
	if !IsBadRequest(err) {
		t.Fatalf("expected 400 APIError, got %v", err)
	}
	// Oversized non-JSON bodies fall back to the status text rather than echoing the body.
	if !strings.Contains(err.Error(), http.StatusText(http.StatusBadRequest)) || strings.Contains(err.Error(), "yyyy") {
		t.Fatalf("unexpected message: %v", err)
	}
}
