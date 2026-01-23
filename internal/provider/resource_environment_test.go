package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEnvironmentResource_Metadata(t *testing.T) {
	r := &environmentResource{}

	// Test metadata
	req := resource.MetadataRequest{
		ProviderTypeName: "kosli",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_environment"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestEnvironmentResource_Schema(t *testing.T) {
	r := &environmentResource{}

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.TODO(), req, resp)

	// Verify schema has description
	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := resp.Schema.Attributes
	requiredAttrs := []string{"name", "type", "description", "include_scaling"}
	for _, attr := range requiredAttrs {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}

	// Verify name is required
	nameAttr := attrs["name"]
	if nameAttr.IsRequired() == false {
		t.Error("Expected 'name' attribute to be required")
	}

	// Verify type is required
	typeAttr := attrs["type"]
	if typeAttr.IsRequired() == false {
		t.Error("Expected 'type' attribute to be required")
	}

	// Verify description is optional
	descAttr := attrs["description"]
	if descAttr.IsOptional() == false {
		t.Error("Expected 'description' attribute to be optional")
	}

	// Verify include_scaling is optional and computed
	includeScalingAttr := attrs["include_scaling"]
	if includeScalingAttr.IsOptional() == false {
		t.Error("Expected 'include_scaling' attribute to be optional")
	}
	if includeScalingAttr.IsComputed() == false {
		t.Error("Expected 'include_scaling' attribute to be computed")
	}
}

func TestEnvironmentResource_Configure(t *testing.T) {
	r := &environmentResource{}

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

func TestEnvironmentResource_Configure_WrongType(t *testing.T) {
	r := &environmentResource{}

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

func TestEnvironmentResourceModel_Structure(t *testing.T) {
	// Test that the model can be created with expected fields
	model := environmentResourceModel{
		Name:           types.StringValue("production-k8s"),
		Type:           types.StringValue("K8S"),
		Description:    types.StringValue("Production cluster"),
		IncludeScaling: types.BoolValue(true),
	}

	if model.Name.ValueString() != "production-k8s" {
		t.Error("Expected Name to be set correctly")
	}

	if model.Type.ValueString() != "K8S" {
		t.Error("Expected Type to be set correctly")
	}

	if model.Description.ValueString() != "Production cluster" {
		t.Error("Expected Description to be set correctly")
	}

	if model.IncludeScaling.ValueBool() != true {
		t.Error("Expected IncludeScaling to be set correctly")
	}
}

func TestEnvironmentResourceModel_WithNullValues(t *testing.T) {
	// Test that the model handles null values correctly
	model := environmentResourceModel{
		Name:           types.StringValue("test-env"),
		Type:           types.StringValue("docker"),
		Description:    types.StringNull(),
		IncludeScaling: types.BoolValue(false),
	}

	if model.Name.ValueString() != "test-env" {
		t.Error("Expected Name to be set correctly")
	}

	if !model.Description.IsNull() {
		t.Error("Expected Description to be null")
	}

	if model.IncludeScaling.ValueBool() != false {
		t.Error("Expected IncludeScaling to be false")
	}
}

func TestNewEnvironmentResource(t *testing.T) {
	r := NewEnvironmentResource()

	if r == nil {
		t.Fatal("Expected non-nil resource")
	}

	_, ok := r.(*environmentResource)
	if !ok {
		t.Error("Expected resource to be of type *environmentResource")
	}
}

func TestEnvironmentResource_Implements(t *testing.T) {
	// Verify the resource implements required interfaces
	var _ resource.Resource = &environmentResource{}
	var _ resource.ResourceWithImportState = &environmentResource{}
}

// Note: Full CRUD operation tests require acceptance testing (issue #71)
// These tests verify the resource structure and basic configuration,
// while acceptance tests will verify the full lifecycle against a real API.
//
// The CRUD methods (Create, Read, Update, Delete) have 0% coverage in unit tests
// because they require:
// - A real or mock Kosli API client
// - Terraform framework context (plans, state)
// - Complex setup with request/response mocking
//
// These will be thoroughly tested in acceptance tests where:
// - Real resources are created/updated/deleted in a test Kosli organization
// - The full Terraform lifecycle is exercised
// - API integration is validated end-to-end
