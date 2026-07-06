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

func TestListServiceAccounts_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/service-accounts/test-org") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := []ServiceAccount{
			{
				Name:           "ci-pipeline",
				DisplayName:    "CI Pipeline",
				Description:    "CI/CD service account",
				Privilege:      "member",
				CreatingUserID: "user-123",
				CreatedAt:      1234567890,
			},
			{
				Name:           "readonly",
				DisplayName:    "Read Only",
				Privilege:      "reader",
				CreatingUserID: "user-456",
				CreatedAt:      1234567891,
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

	accounts, err := client.ListServiceAccounts(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(accounts) != 2 {
		t.Fatalf("expected 2 service accounts, got %d", len(accounts))
	}
	if accounts[0].Name != "ci-pipeline" {
		t.Errorf("expected name 'ci-pipeline', got %s", accounts[0].Name)
	}
	if accounts[0].Privilege != "member" {
		t.Errorf("expected privilege 'member', got %s", accounts[0].Privilege)
	}
	if accounts[1].Privilege != "reader" {
		t.Errorf("expected privilege 'reader', got %s", accounts[1].Privilege)
	}
}

func TestGetServiceAccount_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/service-accounts/test-org/ci-pipeline") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := ServiceAccount{
			Name:           "ci-pipeline",
			DisplayName:    "CI Pipeline",
			Description:    "CI/CD service account",
			Privilege:      "member",
			CreatingUserID: "user-123",
			CreatedAt:      1234567890,
			ForWebhook:     true,
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

	account, err := client.GetServiceAccount(context.Background(), "ci-pipeline")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if account.Name != "ci-pipeline" {
		t.Errorf("expected name 'ci-pipeline', got %s", account.Name)
	}
	if account.Description != "CI/CD service account" {
		t.Errorf("unexpected description: %s", account.Description)
	}
	if !account.ForWebhook {
		t.Error("expected ForWebhook to be true")
	}
}

func TestGetServiceAccount_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Service account not found"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.GetServiceAccount(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected not-found error, got %v", err)
	}
}

func TestCreateServiceAccount_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/service-accounts/test-org") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req CreateServiceAccountRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Name != "ci-pipeline" {
			t.Errorf("expected name 'ci-pipeline', got %s", req.Name)
		}
		if req.Privilege != "member" {
			t.Errorf("expected privilege 'member', got %s", req.Privilege)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ServiceAccount{
			Name:           req.Name,
			DisplayName:    req.Name,
			Description:    req.Description,
			Privilege:      req.Privilege,
			CreatingUserID: "user-123",
			CreatedAt:      1234567890,
			ForWebhook:     false,
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

	account, err := client.CreateServiceAccount(context.Background(), &CreateServiceAccountRequest{
		Name:        "ci-pipeline",
		Description: "CI/CD service account",
		Privilege:   "member",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if account.Name != "ci-pipeline" {
		t.Errorf("expected name 'ci-pipeline', got %s", account.Name)
	}
	if account.Privilege != "member" {
		t.Errorf("expected privilege 'member', got %s", account.Privilege)
	}
}

func TestUpdateServiceAccount_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/service-accounts/test-org/ci-pipeline") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		// Verify the description field is present (so a clear sends null).
		if !strings.Contains(string(body), "description") {
			t.Errorf("expected description field in body, got %s", string(body))
		}
		var req UpdateServiceAccountRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Privilege != "admin" {
			t.Errorf("expected privilege 'admin', got %s", req.Privilege)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServiceAccount{
			Name:        "ci-pipeline",
			Description: "updated",
			Privilege:   req.Privilege,
			CreatedAt:   1234567890,
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

	desc := "updated"
	account, err := client.UpdateServiceAccount(context.Background(), "ci-pipeline", &UpdateServiceAccountRequest{
		Description: &desc,
		Privilege:   "admin",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if account.Privilege != "admin" {
		t.Errorf("expected privilege 'admin', got %s", account.Privilege)
	}
}

func TestUpdateServiceAccount_ClearDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// A nil Description pointer must marshal to JSON null to clear the field.
		if !strings.Contains(string(body), "\"description\":null") {
			t.Errorf("expected description:null in body, got %s", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServiceAccount{Name: "ci-pipeline", Privilege: "member"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.UpdateServiceAccount(context.Background(), "ci-pipeline", &UpdateServiceAccountRequest{
		Description: nil,
		Privilege:   "member",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteServiceAccount_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/service-accounts/test-org/ci-pipeline") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		// The API returns a bare JSON string on success.
		json.NewEncoder(w).Encode("deleted")
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if err := client.DeleteServiceAccount(context.Background(), "ci-pipeline"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
