package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAttachPolicy_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/environments/test-org/test-env/policies") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		names, ok := body["policy_names"].([]any)
		if !ok || len(names) != 1 || names[0] != "test-policy" {
			t.Errorf("unexpected policy_names: %v", body["policy_names"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = c.AttachPolicy(context.Background(), "test-env", "test-policy")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAttachPolicy_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"environment not found"}`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = c.AttachPolicy(context.Background(), "nonexistent-env", "test-policy")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected not found error, got %v", err)
	}
}

func TestDetachPolicy_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/environments/test-org/test-env/policies") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		names, ok := body["policy_names"].([]any)
		if !ok || len(names) != 1 || names[0] != "test-policy" {
			t.Errorf("unexpected policy_names: %v", body["policy_names"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = c.DetachPolicy(context.Background(), "test-env", "test-policy")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDetachPolicy_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"environment not found"}`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = c.DetachPolicy(context.Background(), "nonexistent-env", "test-policy")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected not found error, got %v", err)
	}
}

// TestGetEnvironmentPolicies_StringItems tests parsing when the API returns policies as plain strings.
func TestGetEnvironmentPolicies_StringItems(t *testing.T) {
	// The GET /environments/{org}/{env} endpoint returns policies as a JSON array of strings.
	envJSON := `{
		"name": "test-env",
		"policies": ["policy-a", "policy-b"]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/environments/test-org/test-env") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(envJSON))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	policies, err := c.GetEnvironmentPolicies(context.Background(), "test-env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(policies))
	}
	if policies[0].Name != "policy-a" {
		t.Errorf("expected first policy name 'policy-a', got %q", policies[0].Name)
	}
	if policies[1].Name != "policy-b" {
		t.Errorf("expected second policy name 'policy-b', got %q", policies[1].Name)
	}
}

// TestGetEnvironmentPolicies_ObjectItems tests parsing when the API returns policies as objects.
func TestGetEnvironmentPolicies_ObjectItems(t *testing.T) {
	envJSON := `{
		"name": "test-env",
		"policies": [{"name": "policy-a"}, {"name": "policy-b"}]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(envJSON))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	policies, err := c.GetEnvironmentPolicies(context.Background(), "test-env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(policies))
	}
	if policies[0].Name != "policy-a" {
		t.Errorf("expected first policy name 'policy-a', got %q", policies[0].Name)
	}
	if policies[1].Name != "policy-b" {
		t.Errorf("expected second policy name 'policy-b', got %q", policies[1].Name)
	}
}

// TestGetEnvironmentPolicies_Empty tests that an empty policies array returns an empty slice.
func TestGetEnvironmentPolicies_Empty(t *testing.T) {
	envJSON := `{"name": "test-env", "policies": []}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(envJSON))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	policies, err := c.GetEnvironmentPolicies(context.Background(), "test-env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 0 {
		t.Errorf("expected empty list, got %d policies", len(policies))
	}
}

// TestGetEnvironmentPolicies_NilPolicies tests that a nil policies field returns an empty slice.
func TestGetEnvironmentPolicies_NilPolicies(t *testing.T) {
	envJSON := `{"name": "test-env"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(envJSON))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	policies, err := c.GetEnvironmentPolicies(context.Background(), "test-env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 0 {
		t.Errorf("expected empty list, got %d policies", len(policies))
	}
}

// TestGetEnvironmentPolicies_EnvNotFound tests that a 404 returns a not-found error.
func TestGetEnvironmentPolicies_EnvNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"environment not found"}`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = c.GetEnvironmentPolicies(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected not found error, got %v", err)
	}
}
