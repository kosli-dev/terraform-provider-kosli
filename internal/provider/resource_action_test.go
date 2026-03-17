package provider

import (
	"context"
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
	if !attrs["webhook_url"].IsRequired() {
		t.Error("Expected 'webhook_url' to be required")
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
