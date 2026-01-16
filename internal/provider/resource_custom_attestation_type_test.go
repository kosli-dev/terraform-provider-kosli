package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCustomAttestationTypeResource_Metadata(t *testing.T) {
	r := &customAttestationTypeResource{}

	// Test metadata
	req := resource.MetadataRequest{
		ProviderTypeName: "kosli",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_custom_attestation_type"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestCustomAttestationTypeResource_Schema(t *testing.T) {
	r := &customAttestationTypeResource{}

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.TODO(), req, resp)

	// Verify schema has description
	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := resp.Schema.Attributes
	requiredAttrs := []string{"name", "description", "schema", "jq_rules"}
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

	// Verify description is optional
	descAttr := attrs["description"]
	if descAttr.IsOptional() == false {
		t.Error("Expected 'description' attribute to be optional")
	}

	// Verify schema is required
	schemaAttr := attrs["schema"]
	if schemaAttr.IsRequired() == false {
		t.Error("Expected 'schema' attribute to be required")
	}

	// Verify jq_rules is required
	jqRulesAttr := attrs["jq_rules"]
	if jqRulesAttr.IsRequired() == false {
		t.Error("Expected 'jq_rules' attribute to be required")
	}
}

func TestCustomAttestationTypeResource_Configure(t *testing.T) {
	r := &customAttestationTypeResource{}

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

func TestCustomAttestationTypeResource_Configure_WrongType(t *testing.T) {
	r := &customAttestationTypeResource{}

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

func TestCustomAttestationTypeResourceModel_Structure(t *testing.T) {
	// Test that the model can be created with expected fields
	model := customAttestationTypeResourceModel{
		Name:        types.StringValue("test-attestation"),
		Description: types.StringValue("Test description"),
		Schema:      jsontypes.NewNormalizedValue(`{"type": "object"}`),
		JqRules:     types.ListNull(types.StringType),
	}

	if model.Name.ValueString() != "test-attestation" {
		t.Error("Expected Name to be set correctly")
	}

	if model.Description.ValueString() != "Test description" {
		t.Error("Expected Description to be set correctly")
	}

	if model.Schema.ValueString() != `{"type": "object"}` {
		t.Errorf("Expected Schema to be set correctly, got %q", model.Schema.ValueString())
	}
}

func TestNewCustomAttestationTypeResource(t *testing.T) {
	r := NewCustomAttestationTypeResource()

	if r == nil {
		t.Fatal("Expected non-nil resource")
	}

	_, ok := r.(*customAttestationTypeResource)
	if !ok {
		t.Error("Expected resource to be of type *customAttestationTypeResource")
	}
}

func TestCustomAttestationTypeResource_Implements(t *testing.T) {
	// Verify the resource implements required interfaces
	var _ resource.Resource = &customAttestationTypeResource{}
	var _ resource.ResourceWithImportState = &customAttestationTypeResource{}
}

// Note: Full CRUD operation tests require acceptance testing (issue #17)
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
