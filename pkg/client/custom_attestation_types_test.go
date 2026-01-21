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

		// Return response matching actual API structure
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"name": "test-type",
			"description": "test description",
			"archived": false,
			"versions": [
				{
					"version": 1,
					"timestamp": 1768247330.112509,
					"type_schema": "{\"type\":\"object\"}",
					"evaluator": {
						"content_type": "jq",
						"rules": [".age > 21"]
					},
					"created_by": "Test User"
				}
			],
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
	if result.Schema != `{"type":"object"}` {
		t.Errorf("expected schema '{\"type\":\"object\"}', got %s", result.Schema)
	}
	if len(result.JqRules) != 1 || result.JqRules[0] != ".age > 21" {
		t.Errorf("expected jq_rules ['.age > 21'], got %v", result.JqRules)
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

		// Return array of attestation types matching actual API structure
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{
				"name": "type-1",
				"description": "first type",
				"archived": false,
				"versions": [
					{
						"version": 1,
						"timestamp": 1768247330.112509,
						"type_schema": "{\"type\":\"object\",\"properties\":{\"x\":{\"type\":\"number\"}}}",
						"evaluator": {"content_type": "jq", "rules": [".x > 0"]},
						"created_by": "Test User"
					}
				],
				"org": "test-org"
			},
			{
				"name": "type-2",
				"description": "second type",
				"archived": true,
				"versions": [
					{
						"version": 1,
						"timestamp": 1768247330.112509,
						"type_schema": "{\"type\":\"object\",\"properties\":{\"y\":{\"type\":\"number\"}}}",
						"evaluator": {"content_type": "jq", "rules": [".y > 0"]},
						"created_by": "Test User"
					}
				],
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

	// Verify transformation for first item (schema is normalized from API)
	// Note: json.Marshal may reorder properties, so we unmarshal and compare
	var schema1 map[string]any
	if err := json.Unmarshal([]byte(result[0].Schema), &schema1); err != nil {
		t.Errorf("failed to unmarshal first item schema: %v", err)
	}
	if schema1["type"] != "object" {
		t.Errorf("expected type 'object', got %v", schema1["type"])
	}
	if len(result[0].JqRules) != 1 || result[0].JqRules[0] != ".x > 0" {
		t.Errorf("expected first item jq_rules ['.x > 0'], got %v", result[0].JqRules)
	}

	// Verify transformation for second item (schema is normalized from API)
	var schema2 map[string]any
	if err := json.Unmarshal([]byte(result[1].Schema), &schema2); err != nil {
		t.Errorf("failed to unmarshal second item schema: %v", err)
	}
	if schema2["type"] != "object" {
		t.Errorf("expected type 'object', got %v", schema2["type"])
	}
	if len(result[1].JqRules) != 1 || result[1].JqRules[0] != ".y > 0" {
		t.Errorf("expected second item jq_rules ['.y > 0'], got %v", result[1].JqRules)
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
		Versions: []Version{
			{
				Version:    1,
				TypeSchema: `{"type": "object"}`,
				Evaluator: &Evaluator{
					ContentType: "jq",
					Rules:       []string{".age > 21", ".name != null"},
				},
			},
		},
	}

	at.fromAPIFormat()

	// Verify schema extraction (normalized from API)
	// Note: json.Marshal removes extra whitespace and may reorder properties
	expected := `{"type":"object"}`
	if at.Schema != expected {
		t.Errorf("expected normalized schema %s, got %s", expected, at.Schema)
	}

	// Verify jq_rules extraction
	if len(at.JqRules) != 2 {
		t.Fatalf("expected 2 jq rules, got %d", len(at.JqRules))
	}
	if at.JqRules[0] != ".age > 21" {
		t.Errorf("expected first rule '.age > 21', got %s", at.JqRules[0])
	}
	if at.JqRules[1] != ".name != null" {
		t.Errorf("expected second rule '.name != null', got %s", at.JqRules[1])
	}
}

// TestTransformation_FromAPIFormat_NonJQ tests transformation with non-jq content type
func TestTransformation_FromAPIFormat_NonJQ(t *testing.T) {
	at := &CustomAttestationType{
		Name:        "test-type",
		Description: "test description",
		Versions: []Version{
			{
				Version:    1,
				TypeSchema: `{"type": "object"}`,
				Evaluator: &Evaluator{
					ContentType: "default",
					Rules:       nil,
				},
			},
		},
	}

	at.fromAPIFormat()

	// Verify schema is normalized even for non-jq types
	expected := `{"type":"object"}`
	if at.Schema != expected {
		t.Errorf("expected normalized schema %s, got %s", expected, at.Schema)
	}

	// Verify no jq_rules for non-jq types
	if len(at.JqRules) != 0 {
		t.Errorf("expected empty jq_rules for non-jq type, got %v", at.JqRules)
	}
}

// TestTransformation_PythonStyleSchema tests that Python-style schema is normalized to JSON
func TestTransformation_PythonStyleSchema(t *testing.T) {
	// This tests that we normalize Python-style dict notation from the API
	// The API may return Python-style dict notation with single quotes
	at := &CustomAttestationType{
		Name:        "test-type",
		Description: "test description",
		Versions: []Version{
			{
				Version: 1,
				// Python-style dict notation with single quotes (what the API might return)
				TypeSchema: `{'type': 'object', 'properties': {'x': {'type': 'number'}}}`,
				Evaluator: &Evaluator{
					ContentType: "jq",
					Rules:       []string{".x > 0"},
				},
			},
		},
	}

	err := at.fromAPIFormat()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Schema is normalized to RFC 7159 JSON format (double quotes, no whitespace)
	// Note: json.Marshal may reorder properties, so we unmarshal and verify structure
	var schema map[string]any
	if err := json.Unmarshal([]byte(at.Schema), &schema); err != nil {
		t.Fatalf("failed to unmarshal normalized schema: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("expected type 'object', got %v", schema["type"])
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Errorf("expected properties to be an object")
	}

	x, ok := properties["x"].(map[string]any)
	if !ok {
		t.Errorf("expected x property to be an object")
	}

	if x["type"] != "number" {
		t.Errorf("expected x.type to be 'number', got %v", x["type"])
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

	// When jq_rules is empty, evaluator should not be included
	if _, ok := result["evaluator"]; ok {
		t.Error("expected evaluator to be absent when jq_rules is empty")
	}
}
