package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestNotificationConfigResource_Metadata(t *testing.T) {
	r := &notificationConfigResource{}

	req := resource.MetadataRequest{ProviderTypeName: "kosli"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.TODO(), req, resp)

	if resp.TypeName != "kosli_notification_config" {
		t.Errorf("Expected TypeName %q, got %q", "kosli_notification_config", resp.TypeName)
	}
}

func TestNotificationConfigResource_Schema(t *testing.T) {
	r := &notificationConfigResource{}

	resp := &resource.SchemaResponse{}
	r.Schema(context.TODO(), resource.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	for _, attr := range []string{"notification_type", "emails", "slack_webhooks", "webhooks"} {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}
	if _, exists := attrs["payload_version"]; exists {
		t.Error("Expected 'payload_version' to not exist in schema")
	}

	if !attrs["notification_type"].IsRequired() {
		t.Error("Expected 'notification_type' to be required")
	}
	for _, attr := range []string{"emails", "slack_webhooks", "webhooks"} {
		if !attrs[attr].IsOptional() {
			t.Errorf("Expected %q to be optional", attr)
		}
	}
	if attrs["emails"].IsSensitive() {
		t.Error("Expected 'emails' to not be sensitive")
	}
	for _, attr := range []string{"slack_webhooks", "webhooks"} {
		if !attrs[attr].IsSensitive() {
			t.Errorf("Expected %q to be sensitive", attr)
		}
	}
}

func TestNotificationConfigResource_ConfigValidators(t *testing.T) {
	r := &notificationConfigResource{}
	if got := len(r.ConfigValidators(context.TODO())); got != 1 {
		t.Errorf("Expected 1 config validator, got %d", got)
	}
}

func TestNotificationConfigResource_Configure(t *testing.T) {
	r := &notificationConfigResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}
	if r.client != nil {
		t.Error("Expected client to remain nil when provider data is nil")
	}
}

func TestNotificationConfigResource_Configure_WrongType(t *testing.T) {
	r := &notificationConfigResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), resource.ConfigureRequest{ProviderData: "wrong type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
	if r.client != nil {
		t.Error("Expected client to remain nil when provider data is wrong type")
	}
}

func TestNewNotificationConfigResource(t *testing.T) {
	r := NewNotificationConfigResource()
	if r == nil {
		t.Fatal("Expected non-nil resource")
	}
	if _, ok := r.(*notificationConfigResource); !ok {
		t.Error("Expected resource to be of type *notificationConfigResource")
	}
}

func TestNotificationConfigResource_Implements(t *testing.T) {
	var _ resource.Resource = &notificationConfigResource{}
	var _ resource.ResourceWithImportState = &notificationConfigResource{}
	var _ resource.ResourceWithConfigValidators = &notificationConfigResource{}
}

// TestGroupNotificationTargets verifies the typed API target list flattens into
// per-type values, merging repeated EMAIL targets and skipping unknown types.
func TestGroupNotificationTargets(t *testing.T) {
	values := groupNotificationTargets([]client.NotificationTarget{
		{Type: "EMAIL", Emails: []string{"a@example.com"}},
		{Type: "SLACK", Webhook: "https://hooks.slack.com/services/T0/B0/x"},
		{Type: "EMAIL", Emails: []string{"b@example.com"}},
		{Type: "WEBHOOK", Webhook: "https://ops.example.com/hook", PayloadVersion: "1.0"},
		{Type: "PAGER", Webhook: "https://pager.example.com"},
	})

	if len(values.Emails) != 2 || values.Emails[0] != "a@example.com" || values.Emails[1] != "b@example.com" {
		t.Errorf("unexpected emails: %v", values.Emails)
	}
	if len(values.SlackWebhooks) != 1 || values.SlackWebhooks[0] != "https://hooks.slack.com/services/T0/B0/x" {
		t.Errorf("unexpected slack webhooks: %v", values.SlackWebhooks)
	}
	if len(values.Webhooks) != 1 || values.Webhooks[0] != "https://ops.example.com/hook" {
		t.Errorf("unexpected webhooks: %v", values.Webhooks)
	}
}

// TestNotificationTargetValues_Targets verifies the API payload: one EMAIL
// target carrying every address, then one target per Slack and generic webhook.
func TestNotificationTargetValues_Targets(t *testing.T) {
	values := notificationTargetValues{
		Emails:        []string{"a@example.com", "b@example.com"},
		SlackWebhooks: []string{"https://hooks.slack.com/services/T0/B0/x"},
		Webhooks:      []string{"https://ops.example.com/one", "https://ops.example.com/two"},
	}

	targets := values.targets()
	if len(targets) != 4 {
		t.Fatalf("expected 4 targets, got %d: %+v", len(targets), targets)
	}
	if targets[0].Type != "EMAIL" || len(targets[0].Emails) != 2 || targets[0].Webhook != "" {
		t.Errorf("unexpected email target: %+v", targets[0])
	}
	if targets[1].Type != "SLACK" || targets[1].Webhook != "https://hooks.slack.com/services/T0/B0/x" {
		t.Errorf("unexpected slack target: %+v", targets[1])
	}
	if targets[2].Type != "WEBHOOK" || targets[2].Webhook != "https://ops.example.com/one" {
		t.Errorf("unexpected webhook target: %+v", targets[2])
	}
	if targets[3].Type != "WEBHOOK" || targets[3].Webhook != "https://ops.example.com/two" {
		t.Errorf("unexpected webhook target: %+v", targets[3])
	}
}

func TestNotificationTargetValues_Targets_EmailsOnly(t *testing.T) {
	targets := notificationTargetValues{Emails: []string{"a@example.com"}}.targets()
	if len(targets) != 1 || targets[0].Type != "EMAIL" {
		t.Errorf("expected a single EMAIL target, got %+v", targets)
	}
}

func TestNotificationValuesEquivalent(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"https://hooks.example.com/x", "https://hooks.example.com/x", true},
		{"https://Hooks.Example.com/x/", "https://hooks.example.com/x", true},
		{"Platform@Example.COM", "Platform@example.com", true},
		{"https://hooks.example.com/x", "https://hooks.example.com/y", false},
		{"a@example.com", "b@example.com", false},
	}
	for _, tc := range cases {
		if got := notificationValuesEquivalent(tc.a, tc.b); got != tc.want {
			t.Errorf("notificationValuesEquivalent(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestPreferPriorSpelling(t *testing.T) {
	got := preferPriorSpelling(
		[]string{"https://hooks.example.com/x", "https://new.example.com/y"},
		[]string{"https://Hooks.Example.com/x/"},
	)
	if len(got) != 2 || got[0] != "https://Hooks.Example.com/x/" || got[1] != "https://new.example.com/y" {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestUniqueStrings(t *testing.T) {
	got := uniqueStrings([]string{"a", "b", "a", "c", "b"})
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("unexpected result: %v", got)
	}
}

// TestMapNotificationConfigToModel_MapsTargets verifies target types the API
// does not return map to null, so an unset attribute stays unset.
func TestMapNotificationConfigToModel_MapsTargets(t *testing.T) {
	ctx := context.TODO()
	data := notificationConfigResourceModel{
		Emails:        types.SetNull(types.StringType),
		SlackWebhooks: types.SetNull(types.StringType),
		Webhooks:      types.SetNull(types.StringType),
	}

	config := &client.NotificationConfigResponse{
		NotificationType: "api_key_expiry",
		Targets: []client.NotificationTarget{
			{Type: "EMAIL", Emails: []string{"platform@example.com"}},
			{Type: "SLACK", Webhook: "https://hooks.slack.com/services/T0/B0/x"},
		},
	}

	if diags := mapNotificationConfigToModel(ctx, config, &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.NotificationType.ValueString() != "api_key_expiry" {
		t.Errorf("expected notification_type 'api_key_expiry', got %q", data.NotificationType.ValueString())
	}
	var emails []string
	data.Emails.ElementsAs(ctx, &emails, false)
	if len(emails) != 1 || emails[0] != "platform@example.com" {
		t.Errorf("unexpected emails: %v", emails)
	}
	var slack []string
	data.SlackWebhooks.ElementsAs(ctx, &slack, false)
	if len(slack) != 1 || slack[0] != "https://hooks.slack.com/services/T0/B0/x" {
		t.Errorf("unexpected slack webhooks: %v", slack)
	}
	if !data.Webhooks.IsNull() {
		t.Errorf("expected webhooks to be null, got %v", data.Webhooks)
	}
}

// TestMapNotificationConfigToModel_KeepsConfiguredSpelling verifies values the
// API normalized (lowercased host, stripped trailing slash) keep the spelling
// already in the model, so applies stay consistent and plans stay clean.
func TestMapNotificationConfigToModel_KeepsConfiguredSpelling(t *testing.T) {
	ctx := context.TODO()
	emails, _ := types.SetValueFrom(ctx, types.StringType, []string{"Platform@Example.COM"})
	webhooks, _ := types.SetValueFrom(ctx, types.StringType, []string{"https://Ops.Example.com/hook/"})
	data := notificationConfigResourceModel{
		NotificationType: types.StringValue("api_key_expiry"),
		Emails:           emails,
		SlackWebhooks:    types.SetNull(types.StringType),
		Webhooks:         webhooks,
	}

	config := &client.NotificationConfigResponse{
		NotificationType: "api_key_expiry",
		Targets: []client.NotificationTarget{
			{Type: "EMAIL", Emails: []string{"Platform@example.com"}},
			{Type: "WEBHOOK", Webhook: "https://ops.example.com/hook", PayloadVersion: "1.0"},
		},
	}

	if diags := mapNotificationConfigToModel(ctx, config, &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !data.Emails.Equal(emails) {
		t.Errorf("expected emails %v to be kept, got %v", emails, data.Emails)
	}
	if !data.Webhooks.Equal(webhooks) {
		t.Errorf("expected webhooks %v to be kept, got %v", webhooks, data.Webhooks)
	}
}

// TestMapNotificationConfigToModel_DetectsDrift verifies a value changed
// outside Terraform replaces the stale one in state.
func TestMapNotificationConfigToModel_DetectsDrift(t *testing.T) {
	ctx := context.TODO()
	emails, _ := types.SetValueFrom(ctx, types.StringType, []string{"old@example.com"})
	data := notificationConfigResourceModel{
		Emails:        emails,
		SlackWebhooks: types.SetNull(types.StringType),
		Webhooks:      types.SetNull(types.StringType),
	}

	config := &client.NotificationConfigResponse{
		NotificationType: "api_key_expiry",
		Targets: []client.NotificationTarget{
			{Type: "EMAIL", Emails: []string{"new@example.com"}},
		},
	}

	if diags := mapNotificationConfigToModel(ctx, config, &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var got []string
	data.Emails.ElementsAs(ctx, &got, false)
	if len(got) != 1 || got[0] != "new@example.com" {
		t.Errorf("expected emails [new@example.com], got %v", got)
	}
}

// notificationConfigState builds resource state holding the given emails.
func notificationConfigState(t *testing.T, ctx context.Context, emails ...string) tfsdk.State {
	t.Helper()

	r := &notificationConfigResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	return tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    notificationConfigRaw(t, ctx, schemaResp.Schema, emails),
	}
}

func notificationConfigRaw(t *testing.T, ctx context.Context, s schema.Schema, emails []string) tftypes.Value {
	t.Helper()

	setType := tftypes.Set{ElementType: tftypes.String}
	emailValues := make([]tftypes.Value, 0, len(emails))
	for _, email := range emails {
		emailValues = append(emailValues, tftypes.NewValue(tftypes.String, email))
	}

	return tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
		"notification_type": tftypes.NewValue(tftypes.String, "api_key_expiry"),
		"emails":            tftypes.NewValue(setType, emailValues),
		"slack_webhooks":    tftypes.NewValue(setType, nil),
		"webhooks":          tftypes.NewValue(setType, nil),
	})
}

// TestNotificationConfigResource_Read_RemovesWhenUnconfigured verifies a config
// deleted outside Terraform (the API then answers 200 with no targets) drops
// out of state so the next plan recreates it.
func TestNotificationConfigResource_Read_RemovesWhenUnconfigured(t *testing.T) {
	ctx := context.TODO()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"notification_type": "api_key_expiry", "scope": null, "targets": [],
			"detection": {}, "last_notified_at": null, "last_failure": null}`))
	}))
	defer server.Close()

	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	r := &notificationConfigResource{client: c}
	state := notificationConfigState(t, ctx, "platform@example.com")
	resp := &resource.ReadResponse{State: state}

	r.Read(ctx, resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected resource to be removed from state")
	}
}

// TestNotificationConfigResource_Read_RefreshesTargets verifies a configured
// type refreshes state from the API.
func TestNotificationConfigResource_Read_RefreshesTargets(t *testing.T) {
	ctx := context.TODO()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/notification-config/test-org/api_key_expiry" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"notification_type": "api_key_expiry", "scope": null,
			"targets": [{"type": "EMAIL", "emails": ["platform@example.com", "security@example.com"]}],
			"detection": {}, "last_notified_at": null, "last_failure": null}`))
	}))
	defer server.Close()

	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	r := &notificationConfigResource{client: c}
	state := notificationConfigState(t, ctx, "platform@example.com")
	resp := &resource.ReadResponse{State: state}

	r.Read(ctx, resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}

	var data notificationConfigResourceModel
	if diags := resp.State.Get(ctx, &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics reading state: %v", diags)
	}
	var emails []string
	data.Emails.ElementsAs(ctx, &emails, false)
	if len(emails) != 2 {
		t.Errorf("expected 2 emails in state, got %v", emails)
	}
}
