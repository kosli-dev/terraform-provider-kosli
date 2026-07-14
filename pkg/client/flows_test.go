package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateFlow_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/flows/test-org/template_file") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("expected multipart/form-data, got %s", ct)
		}

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
		if data["name"] != "test-flow" {
			t.Errorf("expected name 'test-flow', got %v", data["name"])
		}
		if data["visibility"] != "private" {
			t.Errorf("expected visibility 'private', got %v", data["visibility"])
		}

		// Verify template_file field
		_, fileHeader, err := r.FormFile("template_file")
		if err != nil {
			t.Errorf("failed to get template_file: %v", err)
		}
		if fileHeader.Filename != "template.yml" {
			t.Errorf("expected filename 'template.yml', got %s", fileHeader.Filename)
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

	err = c.CreateFlow(context.Background(), &CreateFlowRequest{
		Name:        "test-flow",
		Description: "test description",
		Visibility:  "private",
		Template:    "version: 1\ntrail:\n  attestations: []\n",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateFlow_NoTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		// template_file should be omitted when Template is empty
		_, _, err := r.FormFile("template_file")
		if err == nil {
			t.Error("expected no template_file field, but found one")
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

	err = c.CreateFlow(context.Background(), &CreateFlowRequest{
		Name:       "test-flow",
		Visibility: "private",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateFlow_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request"}`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = c.CreateFlow(context.Background(), &CreateFlowRequest{Name: ""})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsBadRequest(err) {
		t.Errorf("expected bad request error, got %v", err)
	}
}

func TestGetFlow_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/flows/test-org/test-flow") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"name": "test-flow",
			"description": "test description",
			"visibility": "private",
			"template": "",
			"tags": {}
		}`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	flow, err := c.GetFlow(context.Background(), "test-flow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow.Name != "test-flow" {
		t.Errorf("expected name 'test-flow', got %q", flow.Name)
	}
	if flow.Visibility != "private" {
		t.Errorf("expected visibility 'private', got %q", flow.Visibility)
	}
}

func TestGetFlow_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"flow not found"}`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = c.GetFlow(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected not found error, got %v", err)
	}
}

func TestListFlows_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/flows/test-org") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{"name": "flow-a", "description": "First flow"},
			{"name": "flow-b", "description": "Second flow"}
		]`))
	}))
	defer server.Close()

	c, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	flows, err := c.ListFlows(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flows) != 2 {
		t.Fatalf("expected 2 flows, got %d", len(flows))
	}
	if flows[0].Name != "flow-a" {
		t.Errorf("expected first flow name 'flow-a', got %q", flows[0].Name)
	}
}

func TestArchiveFlow_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/flows/test-org/test-flow/archive") {
			t.Errorf("unexpected path: %s", r.URL.Path)
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

	err = c.ArchiveFlow(context.Background(), "test-flow")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestCreateFlowRequest_MarshalMultipart_Nil tests that a nil request
// returns an error instead of panicking.
func TestCreateFlowRequest_MarshalMultipart_Nil(t *testing.T) {
	var req *CreateFlowRequest
	_, _, err := req.MarshalMultipart()
	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
}

func TestCreateFlowRequest_MarshalMultipart_NoTemplate(t *testing.T) {
	req := &CreateFlowRequest{Name: "test", Visibility: "private"}
	body, ct, err := req.MarshalMultipart()
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
