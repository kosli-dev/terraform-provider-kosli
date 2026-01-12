package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCreateCustomAttestationType_Success tests successful creation with schema
func TestCreateCustomAttestationType_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/custom-attestation-types/test-org") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify Content-Type is multipart/form-data
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("expected multipart/form-data, got %s", ct)
		}

		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		// Verify data_json field
		dataJSON := r.FormValue("data_json")
		if dataJSON == "" {
			t.Error("data_json field is empty")
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
			t.Fatalf("failed to unmarshal data_json: %v", err)
		}

		// Verify evaluator transformation
		evaluator, ok := data["evaluator"].(map[string]any)
		if !ok {
			t.Fatal("evaluator not found or not a map")
		}
		if evaluator["content_type"] != "jq" {
			t.Errorf("expected content_type 'jq', got %v", evaluator["content_type"])
		}

		rules, ok := evaluator["rules"].([]any)
		if !ok || len(rules) != 1 {
			t.Errorf("expected 1 rule, got %v", rules)
		}

		// Verify type_schema file
		_, fileHeader, err := r.FormFile("type_schema")
		if err != nil {
			t.Errorf("failed to get type_schema file: %v", err)
		}
		if fileHeader.Filename != "schema.json" {
			t.Errorf("expected filename 'schema.json', got %s", fileHeader.Filename)
		}

		// Return 201 Created with "OK"
		w.WriteHeader(http.StatusCreated)
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

	req := &CreateCustomAttestationTypeRequest{
		Name:        "test-type",
		Description: "test description",
		Schema:      `{"type":"object"}`,
		JqRules:     []string{".age > 21"},
	}

	err = client.CreateCustomAttestationType(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestCreateCustomAttestationType_NoSchema tests creation without schema
func TestCreateCustomAttestationType_NoSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		// Verify type_schema file should not exist
		_, _, err := r.FormFile("type_schema")
		if err == nil {
			t.Error("expected no type_schema file, but found one")
		}

		// Return 201 Created
		w.WriteHeader(http.StatusCreated)
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

	req := &CreateCustomAttestationTypeRequest{
		Name:        "test-type",
		Description: "test description",
		Schema:      "", // Empty schema
		JqRules:     []string{".age > 21"},
	}

	err = client.CreateCustomAttestationType(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestCreateCustomAttestationType_BadRequest tests validation errors
func TestCreateCustomAttestationType_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request"}`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req := &CreateCustomAttestationTypeRequest{
		Name:        "",
		Description: "test",
		JqRules:     []string{},
	}

	err = client.CreateCustomAttestationType(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsBadRequest(err) {
		t.Errorf("expected bad request error, got %v", err)
	}
}

// TestCreateCustomAttestationType_Idempotent tests that the API is idempotent (creates versions)
// Note: The API always returns 201 Created, even for duplicate names.
// It creates a new version instead of returning a conflict error.
func TestCreateCustomAttestationType_Idempotent(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Always return 201, simulating the API's versioning behavior
		w.WriteHeader(http.StatusCreated)
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

	req := &CreateCustomAttestationTypeRequest{
		Name:        "test-type",
		Description: "test",
		JqRules:     []string{".age > 21"},
	}

	// First POST - should succeed
	err = client.CreateCustomAttestationType(context.Background(), req)
	if err != nil {
		t.Fatalf("first POST: expected no error, got %v", err)
	}

	// Second POST with same name - should also succeed (creates version)
	err = client.CreateCustomAttestationType(context.Background(), req)
	if err != nil {
		t.Fatalf("second POST: expected no error, got %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

// TestGetCustomAttestationType_Success tests successful retrieval
func TestGetCustomAttestationType_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/custom-attestation-types/test-org/test-type") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Return response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"name": "test-type",
			"description": "test description",
			"schema": "{\"type\":\"object\"}",
			"evaluator": {
				"content_type": "jq",
				"rules": [".age > 21"]
			},
			"archived": false,
			"org": "test-org"
		}`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	result, err := client.GetCustomAttestationType(context.Background(), "test-type", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify transformation from API format to user format
	if len(result.JqRules) != 1 || result.JqRules[0] != ".age > 21" {
		t.Errorf("expected jq_rules ['.age > 21'], got %v", result.JqRules)
	}
	if result.Evaluator != nil {
		t.Error("expected evaluator to be nil after transformation")
	}
	if result.Name != "test-type" {
		t.Errorf("expected name 'test-type', got %s", result.Name)
	}
	if result.Archived {
		t.Error("expected archived to be false")
	}
}

// TestGetCustomAttestationType_WithVersion tests retrieval with version parameter
func TestGetCustomAttestationType_WithVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify version query parameter
		version := r.URL.Query().Get("version")
		if version != "2" {
			t.Errorf("expected version '2', got %s", version)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"name": "test-type",
			"description": "test description",
			"evaluator": {"content_type": "jq", "rules": [".age > 21"]},
			"archived": false,
			"org": "test-org"
		}`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	opts := &GetCustomAttestationTypeOptions{Version: "2"}
	_, err = client.GetCustomAttestationType(context.Background(), "test-type", opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestGetCustomAttestationType_NotFound tests 404 error
func TestGetCustomAttestationType_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.GetCustomAttestationType(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected not found error, got %v", err)
	}
}

// TestListCustomAttestationTypes_Success tests successful list with multiple items
func TestListCustomAttestationTypes_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/custom-attestation-types/test-org") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Return array of attestation types
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{
				"name": "type-1",
				"description": "first type",
				"evaluator": {"content_type": "jq", "rules": [".x > 0"]},
				"archived": false,
				"org": "test-org"
			},
			{
				"name": "type-2",
				"description": "second type",
				"evaluator": {"content_type": "jq", "rules": [".y > 0"]},
				"archived": true,
				"org": "test-org"
			}
		]`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	result, err := client.ListCustomAttestationTypes(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify count
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}

	// Verify transformation for first item
	if len(result[0].JqRules) != 1 || result[0].JqRules[0] != ".x > 0" {
		t.Errorf("expected first item jq_rules ['.x > 0'], got %v", result[0].JqRules)
	}
	if result[0].Evaluator != nil {
		t.Error("expected evaluator to be nil after transformation")
	}

	// Verify transformation for second item
	if len(result[1].JqRules) != 1 || result[1].JqRules[0] != ".y > 0" {
		t.Errorf("expected second item jq_rules ['.y > 0'], got %v", result[1].JqRules)
	}
	if result[1].Evaluator != nil {
		t.Error("expected evaluator to be nil after transformation")
	}

	// Verify archived status
	if result[1].Archived != true {
		t.Error("expected second item to be archived")
	}
}

// TestListCustomAttestationTypes_Empty tests empty list
func TestListCustomAttestationTypes_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	result, err := client.ListCustomAttestationTypes(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty list, got %d items", len(result))
	}
}

// TestArchiveCustomAttestationType_Success tests successful archival
func TestArchiveCustomAttestationType_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/custom-attestation-types/test-org/test-type/archive") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify no body
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			t.Errorf("expected no body, got %s", string(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.ArchiveCustomAttestationType(context.Background(), "test-type")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestArchiveCustomAttestationType_NotFound tests 404 error
func TestArchiveCustomAttestationType_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.ArchiveCustomAttestationType(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected not found error, got %v", err)
	}
}

// TestTransformation_ToAPIFormat tests toAPIFormat transformation
func TestTransformation_ToAPIFormat(t *testing.T) {
	req := &CreateCustomAttestationTypeRequest{
		Name:        "test-type",
		Description: "test description",
		Schema:      `{"type":"object"}`,
		JqRules:     []string{".age > 21", ".name != null"},
	}

	result := req.toAPIFormat()

	// Verify name and description
	if result["name"] != "test-type" {
		t.Errorf("expected name 'test-type', got %v", result["name"])
	}
	if result["description"] != "test description" {
		t.Errorf("expected description 'test description', got %v", result["description"])
	}

	// Verify evaluator structure
	evaluator, ok := result["evaluator"].(map[string]any)
	if !ok {
		t.Fatal("evaluator not found or not a map")
	}

	if evaluator["content_type"] != "jq" {
		t.Errorf("expected content_type 'jq', got %v", evaluator["content_type"])
	}

	rules, ok := evaluator["rules"].([]string)
	if !ok {
		t.Fatal("rules not found or not a slice")
	}

	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0] != ".age > 21" {
		t.Errorf("expected first rule '.age > 21', got %s", rules[0])
	}
	if rules[1] != ".name != null" {
		t.Errorf("expected second rule '.name != null', got %s", rules[1])
	}
}

// TestTransformation_FromAPIFormat tests fromAPIFormat transformation
func TestTransformation_FromAPIFormat(t *testing.T) {
	at := &CustomAttestationType{
		Name:        "test-type",
		Description: "test description",
		Evaluator: &Evaluator{
			ContentType: "jq",
			Rules:       []string{".age > 21", ".name != null"},
		},
	}

	at.fromAPIFormat()

	// Verify transformation
	if len(at.JqRules) != 2 {
		t.Fatalf("expected 2 jq rules, got %d", len(at.JqRules))
	}
	if at.JqRules[0] != ".age > 21" {
		t.Errorf("expected first rule '.age > 21', got %s", at.JqRules[0])
	}
	if at.JqRules[1] != ".name != null" {
		t.Errorf("expected second rule '.name != null', got %s", at.JqRules[1])
	}

	// Verify evaluator is cleared
	if at.Evaluator != nil {
		t.Error("expected evaluator to be nil after transformation")
	}
}

// TestTransformation_FromAPIFormat_NonJQ tests transformation with non-jq content type
func TestTransformation_FromAPIFormat_NonJQ(t *testing.T) {
	at := &CustomAttestationType{
		Name:        "test-type",
		Description: "test description",
		Evaluator: &Evaluator{
			ContentType: "default",
			Rules:       nil,
		},
	}

	at.fromAPIFormat()

	// Verify no transformation for non-jq types
	if len(at.JqRules) != 0 {
		t.Errorf("expected empty jq_rules for non-jq type, got %v", at.JqRules)
	}
	// Evaluator should not be cleared for non-jq types
	if at.Evaluator == nil {
		t.Error("expected evaluator to remain for non-jq content type")
	}
}

// TestTransformation_EmptyRules tests handling of empty rules
func TestTransformation_EmptyRules(t *testing.T) {
	req := &CreateCustomAttestationTypeRequest{
		Name:        "test-type",
		Description: "test",
		JqRules:     []string{},
	}

	result := req.toAPIFormat()
	evaluator := result["evaluator"].(map[string]any)
	rules := evaluator["rules"].([]string)

	if len(rules) != 0 {
		t.Errorf("expected empty rules, got %v", rules)
	}
}
