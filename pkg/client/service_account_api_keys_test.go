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

func TestListServiceAccountAPIKeys_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/service-accounts/test-org/ci-pipeline/api-keys") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		// Raw JSON with float-formatted timestamps: the API serializes numbers
		// as floats (like its other timestamp fields), which must decode fine.
		_, _ = w.Write([]byte(`[
			{"id": "key-1", "description": "prod key", "created_at": 1234567890.5, "expires_at": 0.0, "last_used_at": 1234567900.5},
			{"id": "key-2", "description": "temp key", "created_at": 1234567891.0, "expires_at": 4102444800.0, "last_used_at": 0}
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

	keys, err := client.ListServiceAccountAPIKeys(context.Background(), "ci-pipeline")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].ID != "key-1" {
		t.Errorf("expected id 'key-1', got %s", keys[0].ID)
	}
	if keys[0].Key != "" {
		t.Errorf("expected empty raw key from list, got %q", keys[0].Key)
	}
	if keys[1].ExpiresAt != 4102444800 {
		t.Errorf("expected expires_at 4102444800, got %v", keys[1].ExpiresAt)
	}
}

func TestGetServiceAccountAPIKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/service-accounts/test-org/ci-pipeline/api-keys/key-2") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServiceAccountAPIKey{
			ID: "key-2", Description: "temp key", CreatedAt: 1234567891, ExpiresAt: 4102444800, LastUsedAt: 1234567895,
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

	key, err := client.GetServiceAccountAPIKey(context.Background(), "ci-pipeline", "key-2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if key.Description != "temp key" {
		t.Errorf("expected description 'temp key', got %s", key.Description)
	}
	if key.Key != "" {
		t.Errorf("expected empty raw key from get, got %q", key.Key)
	}
	if key.ExpiresAt != 4102444800 {
		t.Errorf("expected expires_at 4102444800, got %v", key.ExpiresAt)
	}
}

func TestGetServiceAccountAPIKey_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "API key not found"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.GetServiceAccountAPIKey(context.Background(), "ci-pipeline", "key-missing")
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected not-found error, got %v", err)
	}
}

func TestCreateServiceAccountAPIKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/service-accounts/test-org/ci-pipeline/api-keys") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req CreateAPIKeyRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Description != "prod key" {
			t.Errorf("expected description 'prod key', got %s", req.Description)
		}
		if req.ExpiresAt != 4102444800 {
			t.Errorf("expected expires_at 4102444800, got %d", req.ExpiresAt)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ServiceAccountAPIKey{
			ID:          "key-new",
			Key:         "raw-secret-value",
			Description: req.Description,
			CreatedAt:   1234567890,
			ExpiresAt:   float64(req.ExpiresAt),
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

	key, err := client.CreateServiceAccountAPIKey(context.Background(), "ci-pipeline", &CreateAPIKeyRequest{
		Description: "prod key",
		ExpiresAt:   4102444800,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if key.ID != "key-new" {
		t.Errorf("expected id 'key-new', got %s", key.ID)
	}
	if key.Key != "raw-secret-value" {
		t.Errorf("expected raw key value, got %q", key.Key)
	}
}

func TestCreateServiceAccountAPIKey_OmitsZeroExpiry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// expires_at has omitempty, so a zero value must not appear in the body.
		if strings.Contains(string(body), "expires_at") {
			t.Errorf("expected expires_at to be omitted, got %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ServiceAccountAPIKey{ID: "key-new", Key: "raw", Description: "no expiry"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.CreateServiceAccountAPIKey(context.Background(), "ci-pipeline", &CreateAPIKeyRequest{
		Description: "no expiry",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRevokeServiceAccountAPIKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/service-accounts/test-org/ci-pipeline/api-keys/key-1") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode("revoked")
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if err := client.RevokeServiceAccountAPIKey(context.Background(), "ci-pipeline", "key-1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
