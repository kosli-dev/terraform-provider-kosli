package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	expectedAttrs := []string{"name", "environments", "triggers", "webhook_url", "payload_version", "number", "created_by", "last_modified_at"}
	for _, attr := range expectedAttrs {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
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
	if !attrs["payload_version"].IsOptional() {
		t.Error("Expected 'payload_version' to be optional")
	}
	if !attrs["payload_version"].IsComputed() {
		t.Error("Expected 'payload_version' to be computed")
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
		PayloadVersion: types.StringValue("1.0"),
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
	if model.PayloadVersion.ValueString() != "1.0" {
		t.Error("Expected PayloadVersion to be 1.0")
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
