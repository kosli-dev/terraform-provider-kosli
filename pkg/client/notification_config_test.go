package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newNotificationConfigTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

func TestGetNotificationConfig_Configured(t *testing.T) {
	client := newNotificationConfigTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/notification-config/test-org/api_key_expiry" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"notification_type": "api_key_expiry",
			"scope": null,
			"targets": [
				{"type": "EMAIL", "emails": ["platform@example.com", "security@example.com"]},
				{"type": "SLACK", "webhook": "https://hooks.slack.com/services/T0/B0/x"},
				{"type": "WEBHOOK", "webhook": "https://ops.example.com/hook", "payload_version": "1.0"}
			],
			"detection": {},
			"last_notified_at": 1759320000.5,
			"last_failure": {"failed_at": 1759310000.0, "error": "connection refused"}
		}`))
	})

	config, err := client.GetNotificationConfig(context.Background(), NotificationTypeAPIKeyExpiry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.NotificationType != "api_key_expiry" {
		t.Errorf("expected notification_type 'api_key_expiry', got %q", config.NotificationType)
	}
	if config.Scope != nil {
		t.Errorf("expected nil scope, got %q", *config.Scope)
	}
	if len(config.Targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(config.Targets))
	}
	if config.Targets[0].Type != NotificationTargetTypeEmail || len(config.Targets[0].Emails) != 2 {
		t.Errorf("unexpected email target: %+v", config.Targets[0])
	}
	if config.Targets[1].Type != NotificationTargetTypeSlack || config.Targets[1].Webhook != "https://hooks.slack.com/services/T0/B0/x" {
		t.Errorf("unexpected slack target: %+v", config.Targets[1])
	}
	if config.Targets[2].Type != NotificationTargetTypeWebhook || config.Targets[2].PayloadVersion != "1.0" {
		t.Errorf("unexpected webhook target: %+v", config.Targets[2])
	}
	if config.LastNotifiedAt == nil || *config.LastNotifiedAt != 1759320000.5 {
		t.Errorf("expected last_notified_at 1759320000.5, got %v", config.LastNotifiedAt)
	}
	if config.LastFailure == nil || config.LastFailure.Error != "connection refused" || config.LastFailure.FailedAt != 1759310000.0 {
		t.Errorf("unexpected last_failure: %+v", config.LastFailure)
	}
}

func TestGetNotificationConfig_Unconfigured(t *testing.T) {
	client := newNotificationConfigTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"notification_type": "api_key_expiry",
			"scope": null,
			"targets": [],
			"detection": {},
			"last_notified_at": null,
			"last_failure": null
		}`))
	})

	config, err := client.GetNotificationConfig(context.Background(), NotificationTypeAPIKeyExpiry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(config.Targets) != 0 {
		t.Errorf("expected no targets, got %d", len(config.Targets))
	}
	if config.LastNotifiedAt != nil {
		t.Errorf("expected nil last_notified_at, got %v", *config.LastNotifiedAt)
	}
	if config.LastFailure != nil {
		t.Errorf("expected nil last_failure, got %+v", config.LastFailure)
	}
}

func TestGetNotificationConfig_UnknownType(t *testing.T) {
	client := newNotificationConfigTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "Input payload validation failed", "errors": ["notification_type: Input should be 'api_key_expiry'"]}`))
	})

	_, err := client.GetNotificationConfig(context.Background(), "coffee_low")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 APIError, got %v", err)
	}
}

func TestSetNotificationConfig_Success(t *testing.T) {
	client := newNotificationConfigTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/notification-config/test-org/api_key_expiry" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)

		// Only the fields that apply to each target type are sent.
		var raw map[string][]map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		targets := raw["targets"]
		if len(targets) != 2 {
			t.Fatalf("expected 2 targets, got %d", len(targets))
		}
		if _, ok := targets[0]["webhook"]; ok {
			t.Errorf("expected EMAIL target to omit webhook, got %v", targets[0])
		}
		if _, ok := targets[1]["emails"]; ok {
			t.Errorf("expected SLACK target to omit emails, got %v", targets[1])
		}
		if _, ok := targets[1]["payload_version"]; ok {
			t.Errorf("expected SLACK target to omit payload_version, got %v", targets[1])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"notification_type": "api_key_expiry",
			"scope": null,
			"targets": [
				{"type": "EMAIL", "emails": ["platform@example.com"]},
				{"type": "SLACK", "webhook": "https://hooks.slack.com/services/T0/B0/x"}
			],
			"detection": {},
			"last_notified_at": null,
			"last_failure": null
		}`))
	})

	config, err := client.SetNotificationConfig(context.Background(), NotificationTypeAPIKeyExpiry, &NotificationConfigRequest{
		Targets: []NotificationTarget{
			{Type: NotificationTargetTypeEmail, Emails: []string{"platform@example.com"}},
			{Type: NotificationTargetTypeSlack, Webhook: "https://hooks.slack.com/services/T0/B0/x"},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(config.Targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(config.Targets))
	}
}

func TestSetNotificationConfig_Forbidden(t *testing.T) {
	client := newNotificationConfigTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "User 'ci' doesn't have permission to configure notifications in organization 'test-org'"}`))
	})

	_, err := client.SetNotificationConfig(context.Background(), NotificationTypeAPIKeyExpiry, &NotificationConfigRequest{
		Targets: []NotificationTarget{{Type: NotificationTargetTypeEmail, Emails: []string{"a@example.com"}}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 APIError, got %v", err)
	}
}

func TestDeleteNotificationConfig_Success(t *testing.T) {
	called := false
	client := newNotificationConfigTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/notification-config/test-org/api_key_expiry" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`"OK"`))
	})

	if err := client.DeleteNotificationConfig(context.Background(), NotificationTypeAPIKeyExpiry); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Error("expected DELETE request to be sent")
	}
}
