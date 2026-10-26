package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAPIPath(t *testing.T) {
	tests := []struct {
		name     string
		segments []string
		want     string
	}{
		{"single", []string{"environments"}, "/environments"},
		{"plain", []string{"environments", "my-org", "prod"}, "/environments/my-org/prod"},
		{"slash in name", []string{"environments", "my-org", "prod/archive"}, "/environments/my-org/prod%2Farchive"},
		{"dot dot", []string{"environments", "my-org", "../flows"}, "/environments/my-org/..%2Fflows"},
		{"bare dot dot", []string{"environments", "my-org", ".."}, "/environments/my-org/%2E%2E"},
		{"bare dot", []string{"environments", "my-org", "."}, "/environments/my-org/%2E"},
		{"dots inside name", []string{"environments", "my-org", "v1..2"}, "/environments/my-org/v1..2"},
		{"query and fragment chars", []string{"environments", "my-org", "prod?x=1#f"}, "/environments/my-org/prod%3Fx=1%23f"},
		{"space and unicode", []string{"flows", "my org", "ünïcode"}, "/flows/my%20org/%C3%BCn%C3%AFcode"},
		{"empty segment", []string{"flows", ""}, "/flows/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiPath(tt.segments...); got != tt.want {
				t.Errorf("apiPath(%q) = %q, want %q", tt.segments, got, tt.want)
			}
		})
	}
}

// Path metacharacters in a name must not reach a different endpoint.
func TestClient_PathSegmentsAreEscaped(t *testing.T) {
	var got []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Method+" "+r.URL.RequestURI())
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c, err := NewClient("token", "my-org",
		WithBaseURL(server.URL),
		WithRetryPolicy(0, time.Millisecond, time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, _ = c.GetEnvironment(context.Background(), "prod/archive")
	_ = c.ArchiveEnvironment(context.Background(), "../../flows/my-org/x")
	_, _ = c.GetFlow(context.Background(), "name?archived=true")
	_ = c.ArchiveEnvironment(context.Background(), "..")

	want := []string{
		"GET /api/v2/environments/my-org/prod%2Farchive",
		"PUT /api/v2/environments/my-org/..%2F..%2Fflows%2Fmy-org%2Fx/archive",
		"GET /api/v2/flows/my-org/name%3Farchived=true",
		"PUT /api/v2/environments/my-org/%2E%2E/archive",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d requests %v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("request %d = %q, want %q", i, got[i], want[i])
		}
	}
}
