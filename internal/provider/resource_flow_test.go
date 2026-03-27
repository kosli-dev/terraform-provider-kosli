package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlowResource_Metadata(t *testing.T) {
	r := &flowResource{}

	req := resource.MetadataRequest{
		ProviderTypeName: "kosli",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_flow"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestFlowResource_Schema(t *testing.T) {
	r := &flowResource{}

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.TODO(), req, resp)

	// Verify schema has description
	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	// Verify expected attributes exist
	attrs := resp.Schema.Attributes
	expectedAttrs := []string{"name", "description", "visibility", "template"}
	for _, attr := range expectedAttrs {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}

	// Verify name is required
	nameAttr := attrs["name"]
	if !nameAttr.IsRequired() {
		t.Error("Expected 'name' attribute to be required")
	}

	// Verify description is optional
	descAttr := attrs["description"]
	if !descAttr.IsOptional() {
		t.Error("Expected 'description' attribute to be optional")
	}

	// Verify visibility is optional and computed
	visibilityAttr := attrs["visibility"]
	if !visibilityAttr.IsOptional() {
		t.Error("Expected 'visibility' attribute to be optional")
	}
	if !visibilityAttr.IsComputed() {
		t.Error("Expected 'visibility' attribute to be computed")
	}

	// Verify template is optional
	templateAttr := attrs["template"]
	if !templateAttr.IsOptional() {
		t.Error("Expected 'template' attribute to be optional")
	}
}

func TestFlowResource_Configure(t *testing.T) {
	r := &flowResource{}

	// Test with nil provider data (should not error)
	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.TODO(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}

	if r.client != nil {
		t.Error("Expected client to remain nil when provider data is nil")
	}
}

func TestFlowResource_Configure_WrongType(t *testing.T) {
	r := &flowResource{}

	// Test with wrong type of provider data
	req := resource.ConfigureRequest{
		ProviderData: "wrong type",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.TODO(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}

	if r.client != nil {
		t.Error("Expected client to remain nil when provider data is wrong type")
	}
}

func TestFlowResourceModel_Structure(t *testing.T) {
	model := flowResourceModel{
		Name:        types.StringValue("my-flow"),
		Description: types.StringValue("CD pipeline for my app"),
		Visibility:  types.StringValue("public"),
		Template:    types.StringValue("version: 1\ntrail:\n  attestations: []\n"),
	}

	if model.Name.ValueString() != "my-flow" {
		t.Error("Expected Name to be set correctly")
	}

	if model.Description.ValueString() != "CD pipeline for my app" {
		t.Error("Expected Description to be set correctly")
	}

	if model.Visibility.ValueString() != "public" {
		t.Error("Expected Visibility to be set correctly")
	}

	if model.Template.ValueString() == "" {
		t.Error("Expected Template to be set correctly")
	}
}

func TestFlowResourceModel_WithNullValues(t *testing.T) {
	model := flowResourceModel{
		Name:        types.StringValue("my-flow"),
		Description: types.StringNull(),
		Visibility:  types.StringValue("private"),
		Template:    types.StringNull(),
	}

	if model.Name.ValueString() != "my-flow" {
		t.Error("Expected Name to be set correctly")
	}

	if !model.Description.IsNull() {
		t.Error("Expected Description to be null")
	}

	if model.Visibility.ValueString() != "private" {
		t.Error("Expected Visibility to be private")
	}

	if !model.Template.IsNull() {
		t.Error("Expected Template to be null")
	}
}

func TestNewFlowResource(t *testing.T) {
	r := NewFlowResource()

	if r == nil {
		t.Fatal("Expected non-nil resource")
	}

	_, ok := r.(*flowResource)
	if !ok {
		t.Error("Expected resource to be of type *flowResource")
	}
}

func TestFlowResource_Implements(t *testing.T) {
	// Verify the resource implements required interfaces
	var _ resource.Resource = &flowResource{}
	var _ resource.ResourceWithImportState = &flowResource{}
}
