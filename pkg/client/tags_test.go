package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestTagResource_SetTags verifies PATCH method, correct path, and set_tags in body
func TestTagResource_SetTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		// Verify path contains correct segments
		if !strings.Contains(r.URL.Path, "/tags/test-org/environment/production-k8s") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Parse and verify body
		var body TagResourcePayload
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if body.SetTags["env"] != "prod" {
			t.Errorf("expected set_tags.env = 'prod', got %q", body.SetTags["env"])
		}
		if body.SetTags["team"] != "platform" {
			t.Errorf("expected set_tags.team = 'platform', got %q", body.SetTags["team"])
		}
		if len(body.RemoveTags) != 0 {
			t.Errorf("expected empty remove_tags, got %v", body.RemoveTags)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"OK"`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := &TagResourcePayload{
		SetTags:    map[string]string{"env": "prod", "team": "platform"},
		RemoveTags: []string{},
	}

	err = c.TagResource(context.Background(), "environment", "production-k8s", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestTagResource_RemoveTags verifies remove_tags array is sent correctly
func TestTagResource_RemoveTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		var body TagResourcePayload
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if len(body.SetTags) != 0 {
			t.Errorf("expected empty set_tags, got %v", body.SetTags)
		}
		if len(body.RemoveTags) != 2 {
			t.Fatalf("expected 2 remove_tags, got %d", len(body.RemoveTags))
		}
		// Verify both keys are present (order not guaranteed)
		removeSet := map[string]bool{}
		for _, k := range body.RemoveTags {
			removeSet[k] = true
		}
		if !removeSet["old-tag"] {
			t.Error("expected remove_tags to contain 'old-tag'")
		}
		if !removeSet["deprecated"] {
			t.Error("expected remove_tags to contain 'deprecated'")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"OK"`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := &TagResourcePayload{
		SetTags:    map[string]string{},
		RemoveTags: []string{"old-tag", "deprecated"},
	}

	err = c.TagResource(context.Background(), "environment", "staging-env", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestTagResource_Combined verifies set and remove can be sent together
func TestTagResource_Combined(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		var body TagResourcePayload
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if body.SetTags["new-tag"] != "new-value" {
			t.Errorf("expected set_tags.new-tag = 'new-value', got %q", body.SetTags["new-tag"])
		}
		if len(body.RemoveTags) != 1 || body.RemoveTags[0] != "old-tag" {
			t.Errorf("expected remove_tags = ['old-tag'], got %v", body.RemoveTags)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"OK"`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := &TagResourcePayload{
		SetTags:    map[string]string{"new-tag": "new-value"},
		RemoveTags: []string{"old-tag"},
	}

	err = c.TagResource(context.Background(), "environment", "my-env", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestTagResource_ServerError verifies 4xx error handling
func TestTagResource_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "invalid tag key",
		})
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := &TagResourcePayload{
		SetTags:    map[string]string{"bad key": "value"},
		RemoveTags: []string{},
	}

	err = c.TagResource(context.Background(), "environment", "my-env", payload)
	if err == nil {
		t.Fatal("expected error for 400 response, got nil")
	}
}

// TestTagResource_ResourceTypes verifies the path is built correctly for different resource types
func TestTagResource_ResourceTypes(t *testing.T) {
	tests := []struct {
		resourceType string
		resourceID   string
		expectedPath string
	}{
		{"environment", "prod-k8s", "/tags/test-org/environment/prod-k8s"},
		{"flow", "my-flow", "/tags/test-org/flow/my-flow"},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, tt.expectedPath) {
					t.Errorf("expected path to contain %q, got %q", tt.expectedPath, r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`"OK"`))
			}))
			defer server.Close()

			c, err := NewClient("test-token", "test-org",
				WithBaseURL(server.URL),
				WithAPIPath(""),
			)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			payload := &TagResourcePayload{
				SetTags:    map[string]string{"key": "value"},
				RemoveTags: []string{},
			}

			err = c.TagResource(context.Background(), tt.resourceType, tt.resourceID, payload)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
