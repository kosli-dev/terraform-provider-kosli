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
	expectedAttrs := []string{"name", "description", "template", "tags"}
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

	// Verify template is optional
	templateAttr := attrs["template"]
	if !templateAttr.IsOptional() {
		t.Error("Expected 'template' attribute to be optional")
	}

	// Verify tags is optional and computed
	tagsAttr := attrs["tags"]
	if !tagsAttr.IsOptional() {
		t.Error("Expected 'tags' attribute to be optional")
	}
	if !tagsAttr.IsComputed() {
		t.Error("Expected 'tags' attribute to be computed")
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
	tagsMap, diags := types.MapValueFrom(context.TODO(), types.StringType, map[string]string{
		"managed-by": "terraform",
	})
	if diags.HasError() {
		t.Fatal("Failed to create tags map")
	}

	model := flowResourceModel{
		Name:        types.StringValue("my-flow"),
		Description: types.StringValue("CD pipeline for my app"),
		Template:    types.StringValue("version: 1\ntrail:\n  attestations: []\n"),
		Tags:        tagsMap,
	}

	if model.Name.ValueString() != "my-flow" {
		t.Error("Expected Name to be set correctly")
	}

	if model.Description.ValueString() != "CD pipeline for my app" {
		t.Error("Expected Description to be set correctly")
	}

	if model.Template.ValueString() == "" {
		t.Error("Expected Template to be set correctly")
	}

	if model.Tags.IsNull() || model.Tags.IsUnknown() {
		t.Error("Expected Tags to be set")
	}
}

func TestFlowResourceModel_WithNullValues(t *testing.T) {
	model := flowResourceModel{
		Name:        types.StringValue("my-flow"),
		Description: types.StringNull(),
		Template:    types.StringNull(),
		Tags:        types.MapNull(types.StringType),
	}

	if model.Name.ValueString() != "my-flow" {
		t.Error("Expected Name to be set correctly")
	}

	if !model.Description.IsNull() {
		t.Error("Expected Description to be null")
	}

	if !model.Template.IsNull() {
		t.Error("Expected Template to be null")
	}

	if !model.Tags.IsNull() {
		t.Error("Expected Tags to be null")
	}
}

func TestFlowResourceModel_WithTags(t *testing.T) {
	tagsMap, diags := types.MapValueFrom(context.TODO(), types.StringType, map[string]string{
		"env":  "production",
		"team": "platform",
	})
	if diags.HasError() {
		t.Fatal("Failed to create tags map")
	}

	model := flowResourceModel{
		Name: types.StringValue("my-flow"),
		Tags: tagsMap,
	}

	tagElems := model.Tags.Elements()
	if len(tagElems) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tagElems))
	}
}

func TestFlowResourceModel_WithEmptyTags(t *testing.T) {
	emptyMap, diags := types.MapValueFrom(context.TODO(), types.StringType, map[string]string{})
	if diags.HasError() {
		t.Fatal("Failed to create empty tags map")
	}

	model := flowResourceModel{
		Name: types.StringValue("my-flow"),
		Tags: emptyMap,
	}

	if model.Tags.IsNull() {
		t.Error("Expected Tags to be non-null (empty map)")
	}

	if len(model.Tags.Elements()) != 0 {
		t.Error("Expected Tags to have no elements")
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
