package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestNotificationConfigDataSource_Metadata(t *testing.T) {
	d := &notificationConfigDataSource{}

	req := datasource.MetadataRequest{ProviderTypeName: "kosli"}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.TODO(), req, resp)

	if resp.TypeName != "kosli_notification_config" {
		t.Errorf("Expected TypeName %q, got %q", "kosli_notification_config", resp.TypeName)
	}
}

func TestNotificationConfigDataSource_Schema(t *testing.T) {
	d := &notificationConfigDataSource{}

	resp := &datasource.SchemaResponse{}
	d.Schema(context.TODO(), datasource.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	if !attrs["notification_type"].IsRequired() {
		t.Error("Expected 'notification_type' to be required")
	}

	computed := []string{"configured", "emails", "slack_webhooks", "webhooks", "last_notified_at", "last_failure_at", "last_failure_error"}
	for _, attr := range computed {
		a, exists := attrs[attr]
		if !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("Expected %q to be computed", attr)
		}
	}

	for _, attr := range []string{"slack_webhooks", "webhooks"} {
		if !attrs[attr].IsSensitive() {
			t.Errorf("Expected %q to be sensitive", attr)
		}
	}
}

func TestNotificationConfigDataSource_Configure(t *testing.T) {
	d := &notificationConfigDataSource{}

	resp := &datasource.ConfigureResponse{}
	d.Configure(context.TODO(), datasource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}
	if d.client != nil {
		t.Error("Expected client to remain nil when provider data is nil")
	}
}

func TestNotificationConfigDataSource_Configure_WrongType(t *testing.T) {
	d := &notificationConfigDataSource{}

	resp := &datasource.ConfigureResponse{}
	d.Configure(context.TODO(), datasource.ConfigureRequest{ProviderData: "wrong type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
}

func TestNewNotificationConfigDataSource(t *testing.T) {
	d := NewNotificationConfigDataSource()
	if d == nil {
		t.Fatal("Expected non-nil data source")
	}
	if _, ok := d.(*notificationConfigDataSource); !ok {
		t.Error("Expected data source to be of type *notificationConfigDataSource")
	}
}

// TestMapNotificationConfigToDataSourceModel_Configured verifies targets and
// delivery status map into the data source model.
func TestMapNotificationConfigToDataSourceModel_Configured(t *testing.T) {
	ctx := context.TODO()
	lastNotified := 1759320000.0

	config := &client.NotificationConfigResponse{
		NotificationType: "api_key_expiry",
		Targets: []client.NotificationTarget{
			{Type: "EMAIL", Emails: []string{"platform@example.com"}},
			{Type: "WEBHOOK", Webhook: "https://ops.example.com/hook", PayloadVersion: "1.0"},
		},
		LastNotifiedAt: &lastNotified,
		LastFailure:    &client.NotificationFailure{FailedAt: 1759330000.0, Error: "connection refused"},
	}

	var data notificationConfigDataSourceModel
	if diags := mapNotificationConfigToDataSourceModel(ctx, config, &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !data.Configured.ValueBool() {
		t.Error("expected configured to be true")
	}
	if len(data.Emails.Elements()) != 1 {
		t.Errorf("expected 1 email, got %v", data.Emails)
	}
	if data.SlackWebhooks.IsNull() || len(data.SlackWebhooks.Elements()) != 0 {
		t.Errorf("expected slack_webhooks to be an empty set, got %v", data.SlackWebhooks)
	}
	if len(data.Webhooks.Elements()) != 1 {
		t.Errorf("expected 1 webhook, got %v", data.Webhooks)
	}
	if data.LastNotifiedAt.ValueString() != "2025-10-01T12:00:00Z" {
		t.Errorf("expected last_notified_at 2025-10-01T12:00:00Z, got %q", data.LastNotifiedAt.ValueString())
	}
	if data.LastFailureAt.ValueString() != "2025-10-01T14:46:40Z" {
		t.Errorf("expected last_failure_at 2025-10-01T14:46:40Z, got %q", data.LastFailureAt.ValueString())
	}
	if data.LastFailureError.ValueString() != "connection refused" {
		t.Errorf("expected last_failure_error 'connection refused', got %q", data.LastFailureError.ValueString())
	}
}

// TestMapNotificationConfigToDataSourceModel_Unconfigured verifies an
// unconfigured type reads as configured = false with empty sets and null
// delivery status, rather than as an error.
func TestMapNotificationConfigToDataSourceModel_Unconfigured(t *testing.T) {
	ctx := context.TODO()

	config := &client.NotificationConfigResponse{
		NotificationType: "api_key_expiry",
		Targets:          []client.NotificationTarget{},
	}

	var data notificationConfigDataSourceModel
	if diags := mapNotificationConfigToDataSourceModel(ctx, config, &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.Configured.ValueBool() {
		t.Error("expected configured to be false")
	}
	for name, set := range map[string]interface{ IsNull() bool }{
		"emails":         data.Emails,
		"slack_webhooks": data.SlackWebhooks,
		"webhooks":       data.Webhooks,
	} {
		if set.IsNull() {
			t.Errorf("expected %s to be an empty set, not null", name)
		}
	}
	if !data.LastNotifiedAt.IsNull() || !data.LastFailureAt.IsNull() || !data.LastFailureError.IsNull() {
		t.Error("expected delivery status attributes to be null")
	}
}
