package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCustomAttestationTypeDataSource_Metadata(t *testing.T) {
	d := &customAttestationTypeDataSource{}

	// Test metadata
	req := datasource.MetadataRequest{
		ProviderTypeName: "kosli",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_custom_attestation_type"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestCustomAttestationTypeDataSource_Schema(t *testing.T) {
	d := &customAttestationTypeDataSource{}

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.TODO(), req, resp)

	// Verify schema has description
	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := resp.Schema.Attributes
	requiredAttrs := []string{"name", "description", "schema", "jq_rules", "archived", "org"}
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

	// Verify description is computed
	descAttr := attrs["description"]
	if descAttr.IsComputed() == false {
		t.Error("Expected 'description' attribute to be computed")
	}

	// Verify schema is computed
	schemaAttr := attrs["schema"]
	if schemaAttr.IsComputed() == false {
		t.Error("Expected 'schema' attribute to be computed")
	}

	// Verify jq_rules is computed
	jqRulesAttr := attrs["jq_rules"]
	if jqRulesAttr.IsComputed() == false {
		t.Error("Expected 'jq_rules' attribute to be computed")
	}

	// Verify archived is computed
	archivedAttr := attrs["archived"]
	if archivedAttr.IsComputed() == false {
		t.Error("Expected 'archived' attribute to be computed")
	}

	// Verify org is computed
	orgAttr := attrs["org"]
	if orgAttr.IsComputed() == false {
		t.Error("Expected 'org' attribute to be computed")
	}
}

func TestCustomAttestationTypeDataSource_Configure(t *testing.T) {
	d := &customAttestationTypeDataSource{}

	// Test with nil provider data (should not error)
	req := datasource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &datasource.ConfigureResponse{}

	d.Configure(context.TODO(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}

	if d.client != nil {
		t.Error("Expected client to remain nil when provider data is nil")
	}
}

func TestCustomAttestationTypeDataSource_Configure_WrongType(t *testing.T) {
	d := &customAttestationTypeDataSource{}

	// Test with wrong type of provider data
	req := datasource.ConfigureRequest{
		ProviderData: "wrong type",
	}
	resp := &datasource.ConfigureResponse{}

	d.Configure(context.TODO(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}

	if d.client != nil {
		t.Error("Expected client to remain nil when provider data is wrong type")
	}
}

func TestCustomAttestationTypeDataSourceModel_Structure(t *testing.T) {
	// Test that the model can be created with expected fields
	model := customAttestationTypeDataSourceModel{
		Name:        types.StringValue("test-attestation"),
		Description: types.StringValue("Test description"),
		Schema:      types.StringValue(`{"type": "object"}`),
		JqRules:     types.ListNull(types.StringType),
		Archived:    types.BoolValue(false),
		Org:         types.StringValue("test-org"),
	}

	if model.Name.ValueString() != "test-attestation" {
		t.Error("Expected Name to be set correctly")
	}

	if model.Description.ValueString() != "Test description" {
		t.Error("Expected Description to be set correctly")
	}

	if model.Schema.ValueString() != `{"type": "object"}` {
		t.Error("Expected Schema to be set correctly")
	}

	if model.Archived.ValueBool() != false {
		t.Error("Expected Archived to be false")
	}

	if model.Org.ValueString() != "test-org" {
		t.Error("Expected Org to be set correctly")
	}
}

func TestNewCustomAttestationTypeDataSource(t *testing.T) {
	d := NewCustomAttestationTypeDataSource()

	if d == nil {
		t.Fatal("Expected non-nil data source")
	}

	_, ok := d.(*customAttestationTypeDataSource)
	if !ok {
		t.Error("Expected data source to be of type *customAttestationTypeDataSource")
	}
}

func TestCustomAttestationTypeDataSource_Implements(t *testing.T) {
	// Verify the data source implements required interfaces
	var _ datasource.DataSource = &customAttestationTypeDataSource{}
}

// Note: Full Read operation tests require acceptance testing (issue #17)
// These tests verify the data source structure and basic configuration,
// while acceptance tests will verify the full read operation against a real API.
//
// The Read method has 0% coverage in unit tests because it requires:
// - A real or mock Kosli API client
// - Terraform framework context (config, state)
// - Complex setup with request/response mocking
//
// This will be thoroughly tested in acceptance tests where:
// - Real attestation types are queried from a test Kosli organization
// - The full Terraform data source lifecycle is exercised
// - API integration is validated end-to-end
