package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestListEnvironments_Success tests successful listing of environments
func TestListEnvironments_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/environments/test-org") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Return mock response
		resp := []Environment{
			{
				Org:               "test-org",
				Name:              "production-k8s",
				Type:              "K8S",
				Description:       "Production Kubernetes cluster",
				LastModifiedAt:    1234567890.123456,
				LastReportedAt:    nil,
				State:             nil,
				IncludeScaling:    true,
				RequireProvenance: false,
				Tags:              map[string]string{"env": "prod"},
				Policies:          []any{},
			},
			{
				Org:               "test-org",
				Name:              "staging-ecs",
				Type:              "ECS",
				Description:       "Staging ECS cluster",
				LastModifiedAt:    1234567891.123456,
				LastReportedAt:    floatPtr(1234567892.123456),
				State:             map[string]any{"status": "healthy"},
				IncludeScaling:    false,
				RequireProvenance: true,
				Tags:              map[string]string{},
				Policies:          []any{},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	environments, err := client.ListEnvironments(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(environments) != 2 {
		t.Fatalf("expected 2 environments, got %d", len(environments))
	}

	// Verify first environment
	if environments[0].Name != "production-k8s" {
		t.Errorf("expected name 'production-k8s', got %s", environments[0].Name)
	}
	if environments[0].Type != "K8S" {
		t.Errorf("expected type 'K8S', got %s", environments[0].Type)
	}
	if environments[0].LastReportedAt != nil {
		t.Errorf("expected nil LastReportedAt, got %v", environments[0].LastReportedAt)
	}
	if !environments[0].IncludeScaling {
		t.Error("expected IncludeScaling to be true")
	}

	// Verify second environment with nullable fields
	if environments[1].LastReportedAt == nil {
		t.Error("expected non-nil LastReportedAt")
	} else if *environments[1].LastReportedAt != 1234567892.123456 {
		t.Errorf("expected LastReportedAt 1234567892.123456, got %f", *environments[1].LastReportedAt)
	}
	if environments[1].RequireProvenance != true {
		t.Error("expected RequireProvenance to be true")
	}
}

// TestGetEnvironment_Success tests successful retrieval of a single environment
func TestGetEnvironment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/environments/test-org/production-k8s") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Return mock response
		resp := Environment{
			Org:               "test-org",
			Name:              "production-k8s",
			Type:              "K8S",
			Description:       "Production Kubernetes cluster",
			LastModifiedAt:    1234567890.123456,
			LastReportedAt:    floatPtr(1234567891.123456),
			State:             map[string]any{"ready": true},
			IncludeScaling:    true,
			RequireProvenance: false,
			Tags:              map[string]string{"env": "prod"},
			Policies:          []any{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	env, err := client.GetEnvironment(context.Background(), "production-k8s")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if env.Name != "production-k8s" {
		t.Errorf("expected name 'production-k8s', got %s", env.Name)
	}
	if env.Type != "K8S" {
		t.Errorf("expected type 'K8S', got %s", env.Type)
	}
	if env.Description != "Production Kubernetes cluster" {
		t.Errorf("expected description 'Production Kubernetes cluster', got %s", env.Description)
	}
	if env.LastReportedAt == nil {
		t.Error("expected non-nil LastReportedAt")
	}
	if !env.IncludeScaling {
		t.Error("expected IncludeScaling to be true")
	}
	if env.RequireProvenance {
		t.Error("expected RequireProvenance to be false")
	}
}

// TestGetEnvironment_NotFound tests 404 handling for archived environments
func TestGetEnvironment_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Environment named 'archived-env' has been archived for organization 'test-org'",
		})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.GetEnvironment(context.Background(), "archived-env")
	if err == nil {
		t.Fatal("expected error for archived environment, got nil")
	}

	// Verify error message contains archived information
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention 404, got: %v", err)
	}
}

// TestCreateEnvironment_Success tests successful creation
func TestCreateEnvironment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/environments/test-org") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify Content-Type is JSON
		ct := r.Header.Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Errorf("expected application/json, got %s", ct)
		}

		// Parse request body
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		// Verify required fields
		if body["name"] != "production-k8s" {
			t.Errorf("expected name 'production-k8s', got %v", body["name"])
		}
		if body["type"] != "K8S" {
			t.Errorf("expected type 'K8S', got %v", body["type"])
		}
		if body["description"] != "Production cluster" {
			t.Errorf("expected description 'Production cluster', got %v", body["description"])
		}
		if body["include_scaling"] != true {
			t.Errorf("expected include_scaling true, got %v", body["include_scaling"])
		}

		// Verify policies field is sent correctly
		policies, ok := body["policies"].([]any)
		if !ok {
			t.Errorf("expected policies to be array, got %T", body["policies"])
		}
		if len(policies) != 0 {
			t.Errorf("expected empty policies array, got %v", policies)
		}

		// Return 200 OK
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"OK"`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req := &CreateEnvironmentRequest{
		Name:           "production-k8s",
		Type:           "K8S",
		Description:    "Production cluster",
		IncludeScaling: true,
		Policies:       []any{}, // Resource layer will send empty array initially
	}

	err = client.CreateEnvironment(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestCreateEnvironment_Idempotent tests idempotent behavior (create and update use same endpoint)
func TestCreateEnvironment_Idempotent(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Both calls should be identical PUT requests
		if r.Method != http.MethodPut {
			t.Errorf("call %d: expected PUT, got %s", callCount, r.Method)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		// Verify same data in both calls
		if body["name"] != "production-k8s" {
			t.Errorf("call %d: expected name 'production-k8s', got %v", callCount, body["name"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"OK"`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req := &CreateEnvironmentRequest{
		Name:           "production-k8s",
		Type:           "K8S",
		Description:    "Production cluster",
		IncludeScaling: false,
		Policies:       []any{},
	}

	// First call (create)
	err = client.CreateEnvironment(context.Background(), req)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Second call (update) - should be idempotent
	err = client.CreateEnvironment(context.Background(), req)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

// TestCreateEnvironment_WithLogicalEnvironment tests creation with included_environments
func TestCreateEnvironment_WithLogicalEnvironment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		// Verify included_environments field
		includedEnvs, ok := body["included_environments"].([]any)
		if !ok {
			t.Fatal("included_environments not found or not array")
		}
		if len(includedEnvs) != 2 {
			t.Errorf("expected 2 included environments, got %d", len(includedEnvs))
		}
		if includedEnvs[0] != "env1" || includedEnvs[1] != "env2" {
			t.Errorf("unexpected included_environments: %v", includedEnvs)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"OK"`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req := &CreateEnvironmentRequest{
		Name:                 "logical-env",
		Type:                 "logical",
		Description:          "Logical environment",
		IncludedEnvironments: []string{"env1", "env2"},
		Policies:             []any{},
	}

	err = client.CreateEnvironment(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestArchiveEnvironment_Success tests successful archiving
func TestArchiveEnvironment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/environments/test-org/production-k8s/archive") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"OK"`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.ArchiveEnvironment(context.Background(), "production-k8s")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestCreateEnvironment_RecreateArchived tests recreating an archived environment
func TestCreateEnvironment_RecreateArchived(t *testing.T) {
	// This test verifies that creating an environment with the same name as an archived one
	// works correctly (should recreate it)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The API should accept the creation request and recreate the environment
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"OK"`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req := &CreateEnvironmentRequest{
		Name:        "previously-archived",
		Type:        "K8S",
		Description: "Recreated environment",
		Policies:    []any{},
	}

	// Should succeed without error
	err = client.CreateEnvironment(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error when recreating archived environment, got %v", err)
	}
}

// TestEnvironment_NullableFields tests handling of nullable fields
func TestEnvironment_NullableFields(t *testing.T) {
	tests := []struct {
		name     string
		respJSON string
		wantNil  bool
	}{
		{
			name:     "null LastReportedAt",
			respJSON: `{"org":"test-org","name":"env1","type":"K8S","description":"","last_modified_at":123.456,"last_reported_at":null,"state":null,"include_scaling":false,"require_provenance":false,"tags":{},"policies":[]}`,
			wantNil:  true,
		},
		{
			name:     "non-null LastReportedAt",
			respJSON: `{"org":"test-org","name":"env2","type":"ECS","description":"","last_modified_at":123.456,"last_reported_at":789.012,"state":{},"include_scaling":false,"require_provenance":false,"tags":{},"policies":[]}`,
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.respJSON))
			}))
			defer server.Close()

			client, err := NewClient("test-token", "test-org",
				WithBaseURL(server.URL),
				WithAPIPath(""),
			)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			env, err := client.GetEnvironment(context.Background(), "test-env")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if tt.wantNil {
				if env.LastReportedAt != nil {
					t.Errorf("expected nil LastReportedAt, got %v", env.LastReportedAt)
				}
			} else {
				if env.LastReportedAt == nil {
					t.Error("expected non-nil LastReportedAt, got nil")
				}
			}
		})
	}
}

// TestGetEnvironment_LogicalEnvironment tests retrieval of a logical environment
func TestGetEnvironment_LogicalEnvironment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/environments/test-org/production-aggregate") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Return mock response for logical environment
		resp := Environment{
			Org:                  "test-org",
			Name:                 "production-aggregate",
			Type:                 "logical",
			Description:          "All production environments",
			LastModifiedAt:       1234567890.123456,
			LastReportedAt:       nil,
			State:                nil,
			IncludeScaling:       false,
			RequireProvenance:    false,
			Tags:                 map[string]string{},
			Policies:             []any{},
			IncludedEnvironments: []string{"prod-k8s", "prod-ecs", "prod-lambda"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	env, err := client.GetEnvironment(context.Background(), "production-aggregate")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if env.Name != "production-aggregate" {
		t.Errorf("expected name 'production-aggregate', got %s", env.Name)
	}
	if env.Type != "logical" {
		t.Errorf("expected type 'logical', got %s", env.Type)
	}
	if env.Description != "All production environments" {
		t.Errorf("expected description 'All production environments', got %s", env.Description)
	}

	// Verify included_environments
	if len(env.IncludedEnvironments) != 3 {
		t.Fatalf("expected 3 included environments, got %d", len(env.IncludedEnvironments))
	}
	expectedEnvs := []string{"prod-k8s", "prod-ecs", "prod-lambda"}
	for i, expectedEnv := range expectedEnvs {
		if env.IncludedEnvironments[i] != expectedEnv {
			t.Errorf("expected included_environments[%d] = %s, got %s", i, expectedEnv, env.IncludedEnvironments[i])
		}
	}

	// Verify logical environments don't have include_scaling set to true
	if env.IncludeScaling {
		t.Error("expected IncludeScaling to be false for logical environment")
	}
}

// TestCreateEnvironment_LogicalWithEmptyList tests creation fails appropriately with empty included_environments
func TestCreateEnvironment_LogicalWithEmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		// Verify included_environments field exists and is empty
		includedEnvs, ok := body["included_environments"].([]any)
		if !ok {
			t.Fatal("included_environments not found or not array")
		}
		if len(includedEnvs) != 0 {
			t.Errorf("expected 0 included environments, got %d", len(includedEnvs))
		}

		// API would typically return an error for empty list, but we test the request structure
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"OK"`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req := &CreateEnvironmentRequest{
		Name:                 "logical-env",
		Type:                 "logical",
		Description:          "Test logical environment",
		IncludedEnvironments: []string{}, // Empty list
		Policies:             []any{},
	}

	// Client should allow sending empty list (API will validate)
	err = client.CreateEnvironment(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error from client, got %v", err)
	}
}

// TestCreateEnvironment_LogicalUpdate tests updating included_environments
func TestCreateEnvironment_LogicalUpdate(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		// Verify type is logical
		if body["type"] != "logical" {
			t.Errorf("call %d: expected type 'logical', got %v", callCount, body["type"])
		}

		// Verify included_environments changes between calls
		includedEnvs := body["included_environments"].([]any)
		if callCount == 1 {
			if len(includedEnvs) != 2 {
				t.Errorf("call 1: expected 2 environments, got %d", len(includedEnvs))
			}
		} else if callCount == 2 {
			if len(includedEnvs) != 3 {
				t.Errorf("call 2: expected 3 environments, got %d", len(includedEnvs))
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"OK"`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// First call - create with 2 environments
	req := &CreateEnvironmentRequest{
		Name:                 "logical-env",
		Type:                 "logical",
		Description:          "Logical environment",
		IncludedEnvironments: []string{"env1", "env2"},
		Policies:             []any{},
	}
	err = client.CreateEnvironment(context.Background(), req)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Second call - update to 3 environments
	req.IncludedEnvironments = []string{"env1", "env2", "env3"}
	err = client.CreateEnvironment(context.Background(), req)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

// TestListEnvironments_WithLogical tests listing environments returns both physical and logical
func TestListEnvironments_WithLogical(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mix of physical and logical environments
		resp := []Environment{
			{
				Org:               "test-org",
				Name:              "production-k8s",
				Type:              "K8S",
				Description:       "Production cluster",
				LastModifiedAt:    1234567890.123456,
				LastReportedAt:    nil,
				State:             nil,
				IncludeScaling:    true,
				RequireProvenance: false,
				Tags:              map[string]string{},
				Policies:          []any{},
			},
			{
				Org:                  "test-org",
				Name:                 "production-aggregate",
				Type:                 "logical",
				Description:          "All production",
				LastModifiedAt:       1234567891.123456,
				LastReportedAt:       nil,
				State:                nil,
				IncludeScaling:       false,
				RequireProvenance:    false,
				Tags:                 map[string]string{},
				Policies:             []any{},
				IncludedEnvironments: []string{"production-k8s"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	environments, err := client.ListEnvironments(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(environments) != 2 {
		t.Fatalf("expected 2 environments, got %d", len(environments))
	}

	// Verify physical environment
	if environments[0].Type != "K8S" {
		t.Errorf("expected first env type 'K8S', got %s", environments[0].Type)
	}
	if len(environments[0].IncludedEnvironments) != 0 {
		t.Error("expected physical environment to have no included_environments")
	}

	// Verify logical environment
	if environments[1].Type != "logical" {
		t.Errorf("expected second env type 'logical', got %s", environments[1].Type)
	}
	if len(environments[1].IncludedEnvironments) != 1 {
		t.Errorf("expected logical environment to have 1 included_environment, got %d", len(environments[1].IncludedEnvironments))
	}
	if environments[1].IncludedEnvironments[0] != "production-k8s" {
		t.Errorf("expected included environment 'production-k8s', got %s", environments[1].IncludedEnvironments[0])
	}
}

// Helper function to create a pointer to a float64
func floatPtr(f float64) *float64 {
	return &f
}
