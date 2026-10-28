package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// captureHeaders returns a server that records each request's headers.
func captureHeaders(t *testing.T) (*httptest.Server, *[]http.Header) {
	t.Helper()
	var got []http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Header.Clone())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)
	return server, &got
}

func TestDoRequest_AnalyticsHeaders_WithOperation(t *testing.T) {
	server, got := captureHeaders(t)
	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithUserAgent("terraform-provider-kosli/v1.2.3"),
		WithClientName("terraform/v1.2.3"),
		WithInvocationID("3f0c8a9e-6a7b-4c1d-9e2f-0a1b2c3d4e5f"),
	)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	ctx := WithOperation(context.Background(), "kosli_environment create")
	resp, err := c.Post(ctx, "/environments/test-org", map[string]string{"name": "prod"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	h := (*got)[0]
	for header, want := range map[string]string{
		HeaderClient:       "terraform/v1.2.3",
		HeaderOperation:    "kosli_environment create",
		HeaderInvocationID: "3f0c8a9e-6a7b-4c1d-9e2f-0a1b2c3d4e5f",
		"User-Agent":       "terraform-provider-kosli/v1.2.3",
	} {
		if h.Get(header) != want {
			t.Errorf("%s = %q, want %q", header, h.Get(header), want)
		}
	}
}

func TestDoRequest_AnalyticsHeaders_WithoutOperation(t *testing.T) {
	server, got := captureHeaders(t)
	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithClientName("terraform/v1.2.3"),
		WithInvocationID("3f0c8a9e-6a7b-4c1d-9e2f-0a1b2c3d4e5f"),
	)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	resp, err := c.Get(context.Background(), "/environments/test-org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	h := (*got)[0]
	if _, ok := h[HeaderOperation]; ok {
		t.Errorf("%s should be omitted, got %q", HeaderOperation, h.Get(HeaderOperation))
	}
	if h.Get(HeaderClient) != "terraform/v1.2.3" {
		t.Errorf("%s = %q, want %q", HeaderClient, h.Get(HeaderClient), "terraform/v1.2.3")
	}
	if h.Get(HeaderInvocationID) != "3f0c8a9e-6a7b-4c1d-9e2f-0a1b2c3d4e5f" {
		t.Errorf("%s = %q", HeaderInvocationID, h.Get(HeaderInvocationID))
	}
}

func TestDoRequest_AnalyticsHeaders_OmittedByDefault(t *testing.T) {
	server, got := captureHeaders(t)
	c, err := NewClient("test-token", "test-org", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	resp, err := c.Get(context.Background(), "/environments/test-org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	h := (*got)[0]
	for _, header := range []string{HeaderClient, HeaderOperation, HeaderInvocationID} {
		if _, ok := h[header]; ok {
			t.Errorf("%s should be omitted, got %q", header, h.Get(header))
		}
	}
	if h.Get("User-Agent") != DefaultUserAgent {
		t.Errorf("User-Agent = %q, want %q", h.Get("User-Agent"), DefaultUserAgent)
	}
}

func TestDoRequest_AnalyticsHeaders_SentOnRetry(t *testing.T) {
	var got []http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Header.Clone())
		if len(got) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithRetryPolicy(1, 1, 1),
		WithClientName("terraform/v1.2.3"),
		WithInvocationID("3f0c8a9e-6a7b-4c1d-9e2f-0a1b2c3d4e5f"),
	)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	resp, err := c.Delete(WithOperation(context.Background(), "kosli_flow delete"), "/flows/test-org/f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if len(got) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(got))
	}
	if got[1].Get(HeaderOperation) != "kosli_flow delete" {
		t.Errorf("retry %s = %q, want %q", HeaderOperation, got[1].Get(HeaderOperation), "kosli_flow delete")
	}
	if got[1].Get(HeaderInvocationID) != got[0].Get(HeaderInvocationID) {
		t.Errorf("retry changed %s", HeaderInvocationID)
	}
}

func TestAnalyticsOptions_RejectEmpty(t *testing.T) {
	for name, opt := range map[string]ClientOption{
		"client name":   WithClientName(""),
		"invocation id": WithInvocationID(""),
	} {
		if _, err := NewClient("test-token", "test-org", opt); err == nil {
			t.Errorf("%s: expected an error for an empty value", name)
		}
	}
}
