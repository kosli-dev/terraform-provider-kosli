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

func TestListControls_FollowsPagination(t *testing.T) {
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

		page := ControlsPage{PerPage: 100, TotalPages: 2, TotalCount: 3}
		switch r.URL.Query().Get("page") {
		case "1":
			page.Page = 1
			page.Controls = []Control{
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
			}
		case "2":
			page.Page = 2
			page.Controls = []Control{{Identifier: "SDLC-003", Name: "SBOM"}}
		default:
			t.Errorf("unexpected page: %q", r.URL.Query().Get("page"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(page)
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	controls, err := client.ListControls(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(controls) != 3 {
		t.Fatalf("expected 3 controls across pages, got %d", len(controls))
	}
	if controls[0].Identifier != "SDLC-001" {
		t.Errorf("expected identifier 'SDLC-001', got %s", controls[0].Identifier)
	}
	if controls[0].Tags["framework"] != "finos-sdlc" {
		t.Errorf("unexpected tags: %v", controls[0].Tags)
	}
	if controls[1].Version != 3 {
		t.Errorf("expected version 3, got %d", controls[1].Version)
	}
	if controls[2].Identifier != "SDLC-003" {
		t.Errorf("expected identifier 'SDLC-003' from page 2, got %s", controls[2].Identifier)
	}
}

func TestListControls_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ControlsPage{Page: 1, PerPage: 100, TotalPages: 0, TotalCount: 0})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	controls, err := client.ListControls(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(controls) != 0 {
		t.Errorf("expected no controls, got %d", len(controls))
	}
}

func TestListControls_Filters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("search"); got != "sdlc" {
			t.Errorf("expected search=sdlc, got %q", got)
		}
		if got := r.URL.Query().Get("archived"); got != "true" {
			t.Errorf("expected archived=true, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ControlsPage{
			Page:       1,
			PerPage:    100,
			TotalPages: 1,
			TotalCount: 1,
			Controls:   []Control{{Identifier: "SDLC-001", Archived: true}},
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

	controls, err := client.ListControls(context.Background(), &ListControlsOptions{Search: "sdlc", Archived: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(controls) != 1 {
		t.Fatalf("expected 1 control, got %d", len(controls))
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
		// even when cleared. Cleared links must be sent as {} — not JSON null,
		// which the server may treat as "leave unchanged".
		if !strings.Contains(string(body), "\"description\"") {
			t.Errorf("expected description field in body, got %s", string(body))
		}
		if !strings.Contains(string(body), "\"links\":{}") {
			t.Errorf("expected links:{} in body, got %s", string(body))
		}
		if strings.Contains(string(body), "\"links\":null") {
			t.Errorf("cleared links must not be sent as null, got %s", string(body))
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

func TestGetControlVersion_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/controls/test-org/SDLC-001/versions/2") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := ControlVersion{
			Identifier:  "SDLC-001",
			Version:     2,
			Name:        "Binary provenance",
			Description: "Second revision",
			Links:       map[string]string{"docs": "https://example.com/v2"},
			CreatedAt:   1234567890,
			CreatedBy:   "user-123",
			Status:      "created",
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

	version, err := client.GetControlVersion(context.Background(), "SDLC-001", 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if version.Version != 2 {
		t.Errorf("expected version 2, got %d", version.Version)
	}
	if version.Description != "Second revision" {
		t.Errorf("expected description 'Second revision', got %s", version.Description)
	}
	if version.Status != "created" {
		t.Errorf("expected status 'created', got %s", version.Status)
	}
}

func TestGetControlVersion_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Control or version not found"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.GetControlVersion(context.Background(), "SDLC-001", 99)
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
