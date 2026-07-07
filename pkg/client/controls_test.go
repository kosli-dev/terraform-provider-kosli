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

func TestListControls_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/controls/test-org") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Errorf("expected per_page=100, got %q", got)
		}

		resp := ControlsPage{
			Page:       1,
			PerPage:    100,
			TotalPages: 1,
			TotalCount: 2,
			Controls: []Control{
				{
					Identifier:  "SDLC-001",
					Name:        "Binary provenance",
					Description: "All artifacts must have provenance",
					Version:     1,
					CreatedAt:   1234567890,
					CreatedBy:   "user-123",
					Tags:        map[string]string{"framework": "finos-sdlc"},
				},
				{
					Identifier: "SDLC-002",
					Name:       "Peer review",
					Version:    3,
					CreatedAt:  1234567891,
					CreatedBy:  "user-456",
				},
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

	page, err := client.ListControls(context.Background(), &ListControlsOptions{PerPage: 100})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if page.TotalCount != 2 {
		t.Fatalf("expected total count 2, got %d", page.TotalCount)
	}
	if len(page.Controls) != 2 {
		t.Fatalf("expected 2 controls, got %d", len(page.Controls))
	}
	if page.Controls[0].Identifier != "SDLC-001" {
		t.Errorf("expected identifier 'SDLC-001', got %s", page.Controls[0].Identifier)
	}
	if page.Controls[0].Tags["framework"] != "finos-sdlc" {
		t.Errorf("unexpected tags: %v", page.Controls[0].Tags)
	}
	if page.Controls[1].Version != 3 {
		t.Errorf("expected version 3, got %d", page.Controls[1].Version)
	}
}

func TestGetControl_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/controls/test-org/SDLC-001") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := Control{
			Identifier:          "SDLC-001",
			Name:                "Binary provenance",
			Description:         "All artifacts must have provenance",
			Links:               map[string]string{"docs": "https://example.com/sdlc-001"},
			Version:             2,
			CreatedAt:           1234567890,
			CreatedBy:           "user-123",
			Tags:                map[string]string{"framework": "finos-sdlc"},
			Archived:            false,
			PoliciesReferencing: []string{"prod-policy"},
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

	control, err := client.GetControl(context.Background(), "SDLC-001")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if control.Identifier != "SDLC-001" {
		t.Errorf("expected identifier 'SDLC-001', got %s", control.Identifier)
	}
	if control.Name != "Binary provenance" {
		t.Errorf("expected name 'Binary provenance', got %s", control.Name)
	}
	if control.Links["docs"] != "https://example.com/sdlc-001" {
		t.Errorf("unexpected links: %v", control.Links)
	}
	if control.Version != 2 {
		t.Errorf("expected version 2, got %d", control.Version)
	}
	if len(control.PoliciesReferencing) != 1 || control.PoliciesReferencing[0] != "prod-policy" {
		t.Errorf("unexpected policies_referencing: %v", control.PoliciesReferencing)
	}
}

func TestGetControl_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Control not found"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.GetControl(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected not-found error, got %v", err)
	}
}

func TestCreateControl_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/controls/test-org") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req CreateControlRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Identifier != "SDLC-001" {
			t.Errorf("expected identifier 'SDLC-001', got %s", req.Identifier)
		}
		if req.Name != "Binary provenance" {
			t.Errorf("expected name 'Binary provenance', got %s", req.Name)
		}
		if req.Links["docs"] != "https://example.com/sdlc-001" {
			t.Errorf("unexpected links: %v", req.Links)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Control{
			Identifier:  req.Identifier,
			Name:        req.Name,
			Description: req.Description,
			Links:       req.Links,
			Version:     1,
			CreatedAt:   1234567890,
			CreatedBy:   "user-123",
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

	control, err := client.CreateControl(context.Background(), &CreateControlRequest{
		Identifier:  "SDLC-001",
		Name:        "Binary provenance",
		Description: "All artifacts must have provenance",
		Links:       map[string]string{"docs": "https://example.com/sdlc-001"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if control.Identifier != "SDLC-001" {
		t.Errorf("expected identifier 'SDLC-001', got %s", control.Identifier)
	}
	if control.Version != 1 {
		t.Errorf("expected version 1, got %d", control.Version)
	}
}

func TestCreateControl_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"message": "A control with this identifier already exists"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.CreateControl(context.Background(), &CreateControlRequest{
		Identifier: "SDLC-001",
		Name:       "Binary provenance",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsConflict(err) {
		t.Errorf("expected conflict error, got %v", err)
	}
}

func TestCreateControl_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"message": "You don't have permission to access this resource"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.CreateControl(context.Background(), &CreateControlRequest{
		Identifier: "SDLC-001",
		Name:       "Binary provenance",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsForbidden(err) {
		t.Errorf("expected forbidden error, got %v", err)
	}
}

func TestUpdateControl_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/controls/test-org/SDLC-001") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		// PUT replaces mutable fields wholesale, so every field must be present
		// even when cleared.
		if !strings.Contains(string(body), "\"description\"") {
			t.Errorf("expected description field in body, got %s", string(body))
		}
		if !strings.Contains(string(body), "\"links\"") {
			t.Errorf("expected links field in body, got %s", string(body))
		}
		var req UpdateControlRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Name != "Binary provenance v2" {
			t.Errorf("expected name 'Binary provenance v2', got %s", req.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Control{
			Identifier:  "SDLC-001",
			Name:        req.Name,
			Description: req.Description,
			Links:       req.Links,
			Version:     2,
			CreatedAt:   1234567890,
			CreatedBy:   "user-123",
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

	control, err := client.UpdateControl(context.Background(), "SDLC-001", &UpdateControlRequest{
		Name:        "Binary provenance v2",
		Description: "",
		Links:       map[string]string{},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if control.Name != "Binary provenance v2" {
		t.Errorf("expected name 'Binary provenance v2', got %s", control.Name)
	}
	if control.Version != 2 {
		t.Errorf("expected version 2, got %d", control.Version)
	}
}

func TestUpdateControl_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Control not found"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.UpdateControl(context.Background(), "missing", &UpdateControlRequest{Name: "x"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected not-found error, got %v", err)
	}
}

func TestArchiveControl_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/controls/test-org/SDLC-001/archive") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Control{
			Identifier: "SDLC-001",
			Name:       "Binary provenance",
			Version:    2,
			Archived:   true,
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

	control, err := client.ArchiveControl(context.Background(), "SDLC-001")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !control.Archived {
		t.Error("expected Archived to be true")
	}
}

func TestArchiveControl_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Control not found"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.ArchiveControl(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected not-found error, got %v", err)
	}
}
