package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testAction = ActionResponse{
	Name:         "compliance-alerts",
	Type:         "env",
	Number:       1,
	Environments: []string{"production"},
	Triggers:     []string{"ON_NON_COMPLIANT_ENV", "ON_COMPLIANT_ENV"},
	Targets: []ActionTarget{
		{Type: "WEBHOOK", Webhook: "https://hooks.example.com/kosli", PayloadVersion: "1.0"},
	},
	CreatedBy:      "user@example.com",
	CreatedAt:      1633123456.0,
	LastModifiedAt: 1633123457.0,
}

func TestListActions_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/organizations/test-org/environments_notifications") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		actions := []ActionResponse{
			testAction,
			{
				Name:         "scaling-alerts",
				Type:         "env",
				Number:       2,
				Environments: []string{"staging"},
				Triggers:     []string{"ON_SCALED_ARTIFACT"},
				Targets: []ActionTarget{
					{Type: "WEBHOOK", Webhook: "https://hooks.example.com/scale", PayloadVersion: "1.0"},
				},
				CreatedBy:      "other@example.com",
				CreatedAt:      1633123460.0,
				LastModifiedAt: 1633123461.0,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(actions)
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	actions, err := client.ListActions(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	if actions[0].Name != "compliance-alerts" {
		t.Errorf("expected name 'compliance-alerts', got %s", actions[0].Name)
	}
	if actions[0].Number != 1 {
		t.Errorf("expected number 1, got %d", actions[0].Number)
	}
	if len(actions[0].Triggers) != 2 {
		t.Errorf("expected 2 triggers, got %d", len(actions[0].Triggers))
	}
	if len(actions[0].Targets) != 1 {
		t.Errorf("expected 1 target, got %d", len(actions[0].Targets))
	}
	if actions[0].Targets[0].Webhook != "https://hooks.example.com/kosli" {
		t.Errorf("unexpected webhook URL: %s", actions[0].Targets[0].Webhook)
	}
}

func TestListActions_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
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

	actions, err := client.ListActions(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(actions))
	}
}

func TestGetActionByNumber_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/organizations/test-org/environments_notifications/1") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testAction)
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	action, err := client.GetActionByNumber(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if action.Name != "compliance-alerts" {
		t.Errorf("expected name 'compliance-alerts', got %s", action.Name)
	}
	if action.Number != 1 {
		t.Errorf("expected number 1, got %d", action.Number)
	}
	if action.Type != "env" {
		t.Errorf("expected type 'env', got %s", action.Type)
	}
	if action.CreatedBy != "user@example.com" {
		t.Errorf("expected created_by 'user@example.com', got %s", action.CreatedBy)
	}
}

func TestGetActionByNumber_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "action not found",
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

	_, err = client.GetActionByNumber(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention 404, got: %v", err)
	}
}

func TestGetActionByName_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actions := []ActionResponse{testAction}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(actions)
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	action, err := client.GetActionByName(context.Background(), "compliance-alerts")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if action.Name != "compliance-alerts" {
		t.Errorf("expected name 'compliance-alerts', got %s", action.Name)
	}
	if action.Number != 1 {
		t.Errorf("expected number 1, got %d", action.Number)
	}
}

func TestGetActionByName_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
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

	_, err = client.GetActionByName(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to return true, got error: %v", err)
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to mention action name, got: %v", err)
	}
}

func TestCreateOrUpdateAction_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/organizations/test-org/environments_notifications") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			t.Errorf("expected application/json content-type, got %s", r.Header.Get("Content-Type"))
		}

		var body ActionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if body.Name != "compliance-alerts" {
			t.Errorf("expected name 'compliance-alerts', got %s", body.Name)
		}
		if body.Type != "env" {
			t.Errorf("expected type 'env', got %s", body.Type)
		}
		if len(body.Environments) != 1 || body.Environments[0] != "production" {
			t.Errorf("unexpected environments: %v", body.Environments)
		}
		if len(body.Triggers) != 1 || body.Triggers[0] != "ON_NON_COMPLIANT_ENV" {
			t.Errorf("unexpected triggers: %v", body.Triggers)
		}
		if len(body.Targets) != 1 || body.Targets[0].Type != "WEBHOOK" {
			t.Errorf("unexpected targets: %v", body.Targets)
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

	req := &ActionRequest{
		Name:         "compliance-alerts",
		Type:         "env",
		Environments: []string{"production"},
		Triggers:     []string{"ON_NON_COMPLIANT_ENV"},
		Targets: []ActionTarget{
			{Type: "WEBHOOK", Webhook: "https://hooks.example.com/kosli", PayloadVersion: "1.0"},
		},
	}

	err = client.CreateOrUpdateAction(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCreateOrUpdateAction_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid request"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req := &ActionRequest{
		Name:         "compliance-alerts",
		Type:         "env",
		Environments: []string{"production"},
		Triggers:     []string{"ON_NON_COMPLIANT_ENV"},
		Targets:      []ActionTarget{{Type: "WEBHOOK", Webhook: "https://hooks.example.com/kosli"}},
	}

	err = client.CreateOrUpdateAction(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to mention 400, got: %v", err)
	}
}

// TestCreateOrUpdateAction_CalledTwice verifies that the client can make two consecutive
// PUT calls without error. It does not verify server-side idempotency (which requires
// integration tests against a real API).
func TestCreateOrUpdateAction_CalledTwice(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method != http.MethodPut {
			t.Errorf("call %d: expected PUT, got %s", callCount, r.Method)
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

	req := &ActionRequest{
		Name:         "compliance-alerts",
		Type:         "env",
		Environments: []string{"production"},
		Triggers:     []string{"ON_NON_COMPLIANT_ENV"},
		Targets:      []ActionTarget{{Type: "WEBHOOK", Webhook: "https://hooks.example.com/kosli"}},
	}

	if err := client.CreateOrUpdateAction(context.Background(), req); err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if err := client.CreateOrUpdateAction(context.Background(), req); err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestUpdateAction_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		// Must use the numbered path to update in-place
		if !strings.Contains(r.URL.Path, "/organizations/test-org/environments_notifications/1") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body ActionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Name != "compliance-alerts" {
			t.Errorf("expected name 'compliance-alerts', got %s", body.Name)
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

	req := &ActionRequest{
		Name:         "compliance-alerts",
		Type:         "env",
		Environments: []string{"production"},
		Triggers:     []string{"ON_NON_COMPLIANT_ENV"},
		Targets:      []ActionTarget{{Type: "WEBHOOK", Webhook: "https://hooks.example.com/kosli", PayloadVersion: "1.0"}},
	}

	err = client.UpdateAction(context.Background(), 1, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUpdateAction_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "action not found"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req := &ActionRequest{
		Name:    "compliance-alerts",
		Type:    "env",
		Targets: []ActionTarget{{Type: "WEBHOOK", Webhook: "https://hooks.example.com/kosli"}},
	}

	err = client.UpdateAction(context.Background(), 99, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention 404, got: %v", err)
	}
}

func TestDeleteAction_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/organizations/test-org/environments_notifications/1") {
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

	err = client.DeleteAction(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteAction_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "action not found"})
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.DeleteAction(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention 404, got: %v", err)
	}
}
