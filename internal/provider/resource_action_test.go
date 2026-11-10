package provider

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestActionResource_Metadata(t *testing.T) {
	r := &actionResource{}

	req := resource.MetadataRequest{ProviderTypeName: "kosli"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.TODO(), req, resp)

	if resp.TypeName != "kosli_action" {
		t.Errorf("Expected TypeName %q, got %q", "kosli_action", resp.TypeName)
	}
}

func TestActionResource_Schema(t *testing.T) {
	r := &actionResource{}

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.TODO(), req, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	expectedAttrs := []string{"name", "environments", "triggers", "webhook_url", "number", "created_by", "last_modified_at"}
	for _, attr := range expectedAttrs {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}
	if _, exists := attrs["payload_version"]; exists {
		t.Error("Expected 'payload_version' to not exist in schema")
	}

	if !attrs["name"].IsRequired() {
		t.Error("Expected 'name' to be required")
	}
	if !attrs["environments"].IsRequired() {
		t.Error("Expected 'environments' to be required")
	}
	if !attrs["triggers"].IsRequired() {
		t.Error("Expected 'triggers' to be required")
	}
	for _, name := range []string{"webhook_url", "emails", "slack_webhooks", "webhooks"} {
		attr, exists := attrs[name]
		if !exists {
			t.Errorf("Expected attribute %q to exist in schema", name)
			continue
		}
		if !attr.IsOptional() {
			t.Errorf("Expected %q to be optional", name)
		}
	}
	for _, name := range []string{"webhook_url", "slack_webhooks", "webhooks"} {
		if attr, exists := attrs[name]; exists && !attr.IsSensitive() {
			t.Errorf("Expected %q to be sensitive", name)
		}
	}
	if attr, exists := attrs["emails"]; exists && attr.IsSensitive() {
		t.Error("Expected 'emails' to not be sensitive")
	}
	if !attrs["number"].IsComputed() {
		t.Error("Expected 'number' to be computed")
	}
	if !attrs["created_by"].IsComputed() {
		t.Error("Expected 'created_by' to be computed")
	}
	if !attrs["last_modified_at"].IsComputed() {
		t.Error("Expected 'last_modified_at' to be computed")
	}
}

func TestActionResource_Configure(t *testing.T) {
	r := &actionResource{}

	req := resource.ConfigureRequest{ProviderData: nil}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.TODO(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}
	if r.client != nil {
		t.Error("Expected client to remain nil when provider data is nil")
	}
}

func TestActionResource_Configure_WrongType(t *testing.T) {
	r := &actionResource{}

	req := resource.ConfigureRequest{ProviderData: "wrong type"}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.TODO(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
	if r.client != nil {
		t.Error("Expected client to remain nil when provider data is wrong type")
	}
}

func TestActionResourceModel_Structure(t *testing.T) {
	envList, _ := types.ListValueFrom(context.TODO(), types.StringType, []string{"production"})
	trigList, _ := types.ListValueFrom(context.TODO(), types.StringType, []string{"ON_NON_COMPLIANT_ENV"})

	model := actionResourceModel{
		Name:           types.StringValue("compliance-alerts"),
		Environments:   envList,
		Triggers:       trigList,
		WebhookURL:     types.StringValue("https://hooks.example.com/kosli"),
		Number:         types.Int64Value(1),
		CreatedBy:      types.StringValue("user@example.com"),
		LastModifiedAt: types.Float64Value(1633123457.0),
	}

	if model.Name.ValueString() != "compliance-alerts" {
		t.Error("Expected Name to be set correctly")
	}
	if model.Number.ValueInt64() != 1 {
		t.Error("Expected Number to be 1")
	}
	if model.WebhookURL.ValueString() != "https://hooks.example.com/kosli" {
		t.Error("Expected WebhookURL to be set correctly")
	}
}

func TestNewActionResource(t *testing.T) {
	r := NewActionResource()
	if r == nil {
		t.Fatal("Expected non-nil resource")
	}
	if _, ok := r.(*actionResource); !ok {
		t.Error("Expected resource to be of type *actionResource")
	}
}

func TestActionResource_Implements(t *testing.T) {
	var _ resource.Resource = &actionResource{}
	var _ resource.ResourceWithImportState = &actionResource{}
}

// TestMapActionResponseToModel_PreservesWebhookURLWhenAPIReturnsEmpty verifies that
// webhook_url in state is not overwritten when the API redacts it (returns "").
// This guards against the bug where every terraform refresh would blank the webhook URL.
func TestMapActionResponseToModel_PreservesWebhookURLWhenAPIReturnsEmpty(t *testing.T) {
	ctx := context.TODO()
	existingURL := "https://hooks.example.com/my-secret-webhook"

	data := actionResourceModel{
		WebhookURL: types.StringValue(existingURL),
	}

	action := &client.ActionResponse{
		Name:   "my-action",
		Number: 1,
		Targets: []client.ActionTarget{
			{Type: "WEBHOOK", Webhook: ""},
		},
		Environments: []string{"prod"},
		Triggers:     []string{"ON_NON_COMPLIANT_ENV"},
	}

	diags := mapActionResponseToModel(ctx, action, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.WebhookURL.ValueString() != existingURL {
		t.Errorf("expected webhook_url %q to be preserved, got %q", existingURL, data.WebhookURL.ValueString())
	}
}

// TestMapActionResponseToModel_UpdatesWebhookURLWhenAPIReturnsValue verifies that
// webhook_url is updated when the API does return a value.
func TestMapActionResponseToModel_UpdatesWebhookURLWhenAPIReturnsValue(t *testing.T) {
	ctx := context.TODO()

	data := actionResourceModel{
		WebhookURL: types.StringValue("https://old.example.com/hook"),
	}

	newURL := "https://new.example.com/hook"
	action := &client.ActionResponse{
		Name:   "my-action",
		Number: 1,
		Targets: []client.ActionTarget{
			{Type: "WEBHOOK", Webhook: newURL},
		},
		Environments: []string{"prod"},
		Triggers:     []string{"ON_NON_COMPLIANT_ENV"},
	}

	diags := mapActionResponseToModel(ctx, action, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.WebhookURL.ValueString() != newURL {
		t.Errorf("expected webhook_url %q, got %q", newURL, data.WebhookURL.ValueString())
	}
}

// TestMapActionResponseToModel_PreservesWebhookURLWhenNoTargets verifies that
// webhook_url is preserved when the API returns no targets.
func TestMapActionResponseToModel_PreservesWebhookURLWhenNoTargets(t *testing.T) {
	ctx := context.TODO()
	existingURL := "https://hooks.example.com/my-webhook"

	data := actionResourceModel{
		WebhookURL: types.StringValue(existingURL),
	}

	action := &client.ActionResponse{
		Name:         "my-action",
		Number:       1,
		Targets:      []client.ActionTarget{},
		Environments: []string{"prod"},
		Triggers:     []string{"ON_NON_COMPLIANT_ENV"},
	}

	diags := mapActionResponseToModel(ctx, action, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.WebhookURL.ValueString() != existingURL {
		t.Errorf("expected webhook_url %q to be preserved, got %q", existingURL, data.WebhookURL.ValueString())
	}
}

// TestMapActionResponseToModel_MapsAllFields verifies that non-sensitive fields are
// mapped correctly from the API response.
func TestMapActionResponseToModel_MapsAllFields(t *testing.T) {
	ctx := context.TODO()
	data := actionResourceModel{}

	action := &client.ActionResponse{
		Name:           "compliance-alerts",
		Number:         42,
		CreatedBy:      "user@example.com",
		LastModifiedAt: 1633123457.0,
		Environments:   []string{"prod", "staging"},
		Triggers:       []string{"ON_NON_COMPLIANT_ENV", "ON_COMPLIANT_ENV"},
		Targets: []client.ActionTarget{
			{Type: "WEBHOOK", Webhook: "https://hooks.example.com/kosli"},
		},
	}

	diags := mapActionResponseToModel(ctx, action, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.Name.ValueString() != "compliance-alerts" {
		t.Errorf("expected Name %q, got %q", "compliance-alerts", data.Name.ValueString())
	}
	if data.Number.ValueInt64() != 42 {
		t.Errorf("expected Number 42, got %d", data.Number.ValueInt64())
	}
	if data.CreatedBy.ValueString() != "user@example.com" {
		t.Errorf("expected CreatedBy %q, got %q", "user@example.com", data.CreatedBy.ValueString())
	}
	if data.LastModifiedAt.ValueFloat64() != 1633123457.0 {
		t.Errorf("expected LastModifiedAt 1633123457.0, got %f", data.LastModifiedAt.ValueFloat64())
	}
	if data.WebhookURL.ValueString() != "https://hooks.example.com/kosli" {
		t.Errorf("expected WebhookURL to be set, got %q", data.WebhookURL.ValueString())
	}

	var envs []string
	data.Environments.ElementsAs(ctx, &envs, false)
	if len(envs) != 2 || envs[0] != "prod" || envs[1] != "staging" {
		t.Errorf("expected Environments [prod staging], got %v", envs)
	}
}

// TestBuildActionRequest verifies that buildActionRequest constructs the correct
// API payload, including the hardcoded "env" type and "WEBHOOK" target type.
func TestBuildActionRequest(t *testing.T) {
	ctx := context.TODO()

	envList, _ := types.ListValueFrom(ctx, types.StringType, []string{"prod", "staging"})
	trigList, _ := types.ListValueFrom(ctx, types.StringType, []string{"ON_NON_COMPLIANT_ENV"})

	data := actionResourceModel{
		Name:         types.StringValue("my-action"),
		Environments: envList,
		Triggers:     trigList,
		WebhookURL:   types.StringValue("https://hooks.example.com/kosli"),
	}

	req, diags := buildActionRequest(ctx, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if req.Name != "my-action" {
		t.Errorf("expected Name %q, got %q", "my-action", req.Name)
	}
	if req.Type != "env" {
		t.Errorf("expected Type %q, got %q", "env", req.Type)
	}
	if len(req.Environments) != 2 || req.Environments[0] != "prod" {
		t.Errorf("expected Environments [prod staging], got %v", req.Environments)
	}
	if len(req.Triggers) != 1 || req.Triggers[0] != "ON_NON_COMPLIANT_ENV" {
		t.Errorf("expected Triggers [ON_NON_COMPLIANT_ENV], got %v", req.Triggers)
	}
	if len(req.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(req.Targets))
	}
	if req.Targets[0].Type != "WEBHOOK" {
		t.Errorf("expected target Type %q, got %q", "WEBHOOK", req.Targets[0].Type)
	}
	if req.Targets[0].Webhook != "https://hooks.example.com/kosli" {
		t.Errorf("expected Webhook URL, got %q", req.Targets[0].Webhook)
	}
}

func stringSet(t *testing.T, values ...string) types.Set {
	t.Helper()
	set, diags := types.SetValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		t.Fatalf("building set: %v", diags)
	}
	return set
}

func setElements(t *testing.T, set types.Set) []string {
	t.Helper()
	var values []string
	if set.IsNull() {
		return nil
	}
	if diags := set.ElementsAs(context.Background(), &values, false); diags.HasError() {
		t.Fatalf("reading set: %v", diags)
	}
	sort.Strings(values)
	return values
}

func TestActionResource_ConfigValidators(t *testing.T) {
	r := &actionResource{}
	if got := len(r.ConfigValidators(context.Background())); got != 2 {
		t.Errorf("expected 2 config validators (at least one target, webhook_url vs webhooks), got %d", got)
	}
}

func TestBuildActionRequest_AllTargetTypes(t *testing.T) {
	ctx := context.Background()
	envList, _ := types.ListValueFrom(ctx, types.StringType, []string{"prod"})
	trigList, _ := types.ListValueFrom(ctx, types.StringType, []string{"ON_NON_COMPLIANT_ENV"})

	data := actionResourceModel{
		Name:          types.StringValue("my-action"),
		Environments:  envList,
		Triggers:      trigList,
		WebhookURL:    types.StringNull(),
		Emails:        stringSet(t, "a@example.com", "b@example.com"),
		SlackWebhooks: stringSet(t, "https://hooks.slack.com/services/T/B/X"),
		Webhooks:      stringSet(t, "https://hooks.example.com/one"),
	}

	req, diags := buildActionRequest(ctx, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want := []client.ActionTarget{
		{Type: "EMAIL", Emails: []string{"a@example.com", "b@example.com"}},
		{Type: "SLACK", Webhook: "https://hooks.slack.com/services/T/B/X"},
		{Type: "WEBHOOK", Webhook: "https://hooks.example.com/one", PayloadVersion: "1.0"},
	}
	if !reflect.DeepEqual(req.Targets, want) {
		t.Errorf("targets:\n got %+v\nwant %+v", req.Targets, want)
	}
}

func TestBuildActionRequest_EmailsOnly(t *testing.T) {
	ctx := context.Background()
	envList, _ := types.ListValueFrom(ctx, types.StringType, []string{"prod"})
	trigList, _ := types.ListValueFrom(ctx, types.StringType, []string{"ON_NON_COMPLIANT_ENV"})

	data := actionResourceModel{
		Name:          types.StringValue("my-action"),
		Environments:  envList,
		Triggers:      trigList,
		Emails:        stringSet(t, "a@example.com"),
		SlackWebhooks: types.SetNull(types.StringType),
		Webhooks:      types.SetNull(types.StringType),
	}

	req, diags := buildActionRequest(ctx, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(req.Targets) != 1 || req.Targets[0].Type != "EMAIL" {
		t.Errorf("expected a single EMAIL target, got %+v", req.Targets)
	}
}

func TestMapActionResponseToModel_Targets(t *testing.T) {
	const (
		hookA = "https://hooks.example.com/a"
		hookB = "https://hooks.example.com/b"
		slack = "https://hooks.slack.com/services/T/B/X"
	)
	null := types.SetNull(types.StringType)

	tests := []struct {
		name           string
		prior          actionResourceModel
		targets        []client.ActionTarget
		wantWebhookURL types.String
		wantEmails     []string
		wantSlack      []string
		wantWebhooks   []string
	}{
		{
			name:  "attributes in use keep each target type in its own attribute",
			prior: actionResourceModel{Emails: stringSet(t, "a@example.com"), SlackWebhooks: stringSet(t, slack), Webhooks: stringSet(t, hookA)},
			targets: []client.ActionTarget{
				{Type: "EMAIL", Emails: []string{"a@example.com"}},
				{Type: "SLACK", Webhook: slack},
				{Type: "WEBHOOK", Webhook: hookA, PayloadVersion: "1.0"},
			},
			wantWebhookURL: types.StringNull(),
			wantEmails:     []string{"a@example.com"},
			wantSlack:      []string{slack},
			wantWebhooks:   []string{hookA},
		},
		{
			name:  "emails from several EMAIL targets merge",
			prior: actionResourceModel{Emails: stringSet(t, "a@example.com"), SlackWebhooks: null, Webhooks: null},
			targets: []client.ActionTarget{
				{Type: "EMAIL", Emails: []string{"a@example.com"}},
				{Type: "EMAIL", Emails: []string{"b@example.com"}},
			},
			wantWebhookURL: types.StringNull(),
			wantEmails:     []string{"a@example.com", "b@example.com"},
		},
		{
			name:  "webhook_url in use keeps its target and reports the others as drift",
			prior: actionResourceModel{WebhookURL: types.StringValue(hookB), Emails: null, SlackWebhooks: null, Webhooks: null},
			targets: []client.ActionTarget{
				{Type: "WEBHOOK", Webhook: hookA},
				{Type: "WEBHOOK", Webhook: hookB},
				{Type: "SLACK", Webhook: slack},
			},
			wantWebhookURL: types.StringValue(hookB),
			wantSlack:      []string{slack},
			wantWebhooks:   []string{hookA},
		},
		{
			name:           "webhook_url keeps the configured spelling of a normalized URL",
			prior:          actionResourceModel{WebhookURL: types.StringValue("https://Hooks.Example.com/a/"), Emails: null, SlackWebhooks: null, Webhooks: null},
			targets:        []client.ActionTarget{{Type: "WEBHOOK", Webhook: hookA}},
			wantWebhookURL: types.StringValue("https://Hooks.Example.com/a/"),
		},
		{
			name:           "import of a single webhook fills webhook_url",
			prior:          actionResourceModel{},
			targets:        []client.ActionTarget{{Type: "WEBHOOK", Webhook: hookA}, {Type: "EMAIL", Emails: []string{"a@example.com"}}},
			wantWebhookURL: types.StringValue(hookA),
			wantEmails:     []string{"a@example.com"},
		},
		{
			name:           "import of several webhooks fills webhooks",
			prior:          actionResourceModel{},
			targets:        []client.ActionTarget{{Type: "WEBHOOK", Webhook: hookA}, {Type: "WEBHOOK", Webhook: hookB}},
			wantWebhookURL: types.StringNull(),
			wantWebhooks:   []string{hookA, hookB},
		},
		{
			name:           "a Slack target never lands in webhook_url",
			prior:          actionResourceModel{},
			targets:        []client.ActionTarget{{Type: "SLACK", Webhook: slack}},
			wantWebhookURL: types.StringNull(),
			wantSlack:      []string{slack},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.prior
			action := &client.ActionResponse{
				Name: "my-action", Number: 1, Targets: tt.targets,
				Environments: []string{"prod"}, Triggers: []string{"ON_NON_COMPLIANT_ENV"},
			}
			if diags := mapActionResponseToModel(context.Background(), action, &data); diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if !data.WebhookURL.Equal(tt.wantWebhookURL) {
				t.Errorf("webhook_url: got %v, want %v", data.WebhookURL, tt.wantWebhookURL)
			}
			if got := setElements(t, data.Emails); !reflect.DeepEqual(got, tt.wantEmails) {
				t.Errorf("emails: got %v, want %v", got, tt.wantEmails)
			}
			if got := setElements(t, data.SlackWebhooks); !reflect.DeepEqual(got, tt.wantSlack) {
				t.Errorf("slack_webhooks: got %v, want %v", got, tt.wantSlack)
			}
			if got := setElements(t, data.Webhooks); !reflect.DeepEqual(got, tt.wantWebhooks) {
				t.Errorf("webhooks: got %v, want %v", got, tt.wantWebhooks)
			}
		})
	}
}

func TestSlackAppActionImportError(t *testing.T) {
	if msg := slackAppActionImportError(&client.ActionResponse{Name: "plain"}); msg != "" {
		t.Errorf("expected no error for an ordinary action, got %q", msg)
	}
	if msg := slackAppActionImportError(&client.ActionResponse{Name: "from-slack", IsCreatedFromSlackApp: true}); msg == "" {
		t.Error("expected an error for an action created from the Slack app")
	}
}
