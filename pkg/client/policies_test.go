package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreatePolicy_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/policies/test-org") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("expected multipart/form-data, got %s", ct)
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		// Verify payload field
		payloadJSON := r.FormValue("payload")
		if payloadJSON == "" {
			t.Error("payload field is empty")
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		if payload["name"] != "test-policy" {
			t.Errorf("expected name 'test-policy', got %v", payload["name"])
		}
		if payload["type"] != "env" {
			t.Errorf("expected type 'env', got %v", payload["type"])
		}

		// Verify policy_file field
		_, fileHeader, err := r.FormFile("policy_file")
		if err != nil {
			t.Errorf("failed to get policy_file: %v", err)
		}
		if fileHeader.Filename != "policy.yaml" {
			t.Errorf("expected filename 'policy.yaml', got %s", fileHeader.Filename)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`"created"`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = c.CreatePolicy(context.Background(), &CreatePolicyRequest{
		Name:    "test-policy",
		Content: "_schema: https://kosli.com/schemas/policy/environment/v1\n",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreatePolicy_DefaultsTypeToEnv(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}
		payloadJSON := r.FormValue("payload")
		var payload map[string]any
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		if payload["type"] != "env" {
			t.Errorf("expected type 'env', got %v", payload["type"])
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Type is empty — should default to "env"
	err = c.CreatePolicy(context.Background(), &CreatePolicyRequest{
		Name:    "test-policy",
		Content: "_schema: https://kosli.com/schemas/policy/environment/v1\n",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetPolicy_Success(t *testing.T) {
	expected := Policy{
		Name:        "test-policy",
		Description: "Test policy",
		CreatedAt:   1700000000.0,
		Versions: []PolicyVersion{
			{
				Version:   2,
				Content:   "_schema: https://kosli.com/schemas/policy/environment/v1\n",
				CreatedAt: 1700000001.0,
				CreatedBy: "user@example.com",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/policies/test-org/test-policy") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	policy, err := c.GetPolicy(context.Background(), "test-policy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Name != expected.Name {
		t.Errorf("expected name %q, got %q", expected.Name, policy.Name)
	}
	if policy.Description != expected.Description {
		t.Errorf("expected description %q, got %q", expected.Description, policy.Description)
	}
	if len(policy.Versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(policy.Versions))
	}
	if policy.Versions[0].Version != 2 {
		t.Errorf("expected version 2, got %d", policy.Versions[0].Version)
	}
}

func TestGetPolicy_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"policy not found"}`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = c.GetPolicy(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected not found error, got %v", err)
	}
}

func TestListPolicies_Success(t *testing.T) {
	expected := []Policy{
		{Name: "policy-a", Description: "First policy"},
		{Name: "policy-b", Description: "Second policy"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/policies/test-org") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	policies, err := c.ListPolicies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(policies))
	}
	if policies[0].Name != "policy-a" {
		t.Errorf("expected first policy name 'policy-a', got %q", policies[0].Name)
	}
}

func TestListPolicies_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	policies, err := c.ListPolicies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 0 {
		t.Errorf("expected empty list, got %d policies", len(policies))
	}
}

func TestCreatePolicyMultipartRequest_NoContent(t *testing.T) {
	payload := map[string]any{"name": "test", "type": "env"}
	body, ct, err := createPolicyMultipartRequest(payload, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(ct, "multipart/form-data") {
		t.Errorf("expected multipart/form-data content type, got %s", ct)
	}
	if body == nil {
		t.Error("expected non-nil body")
	}
}
