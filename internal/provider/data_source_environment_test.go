package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEnvironmentDataSource_Metadata(t *testing.T) {
	d := &environmentDataSource{}

	// Test metadata
	req := datasource.MetadataRequest{
		ProviderTypeName: "kosli",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_environment"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestEnvironmentDataSource_Schema(t *testing.T) {
	d := &environmentDataSource{}

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.TODO(), req, resp)

	// Verify schema has description
	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := resp.Schema.Attributes
	requiredAttrs := []string{"name", "type", "description", "include_scaling", "last_modified_at", "last_reported_at"}
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

	// Verify type is computed
	typeAttr := attrs["type"]
	if typeAttr.IsComputed() == false {
		t.Error("Expected 'type' attribute to be computed")
	}

	// Verify description is computed
	descAttr := attrs["description"]
	if descAttr.IsComputed() == false {
		t.Error("Expected 'description' attribute to be computed")
	}

	// Verify include_scaling is computed
	includeScalingAttr := attrs["include_scaling"]
	if includeScalingAttr.IsComputed() == false {
		t.Error("Expected 'include_scaling' attribute to be computed")
	}

	// Verify last_modified_at is computed
	lastModifiedAtAttr := attrs["last_modified_at"]
	if lastModifiedAtAttr.IsComputed() == false {
		t.Error("Expected 'last_modified_at' attribute to be computed")
	}

	// Verify last_reported_at is computed
	lastReportedAtAttr := attrs["last_reported_at"]
	if lastReportedAtAttr.IsComputed() == false {
		t.Error("Expected 'last_reported_at' attribute to be computed")
	}
}

func TestEnvironmentDataSource_Configure(t *testing.T) {
	d := &environmentDataSource{}

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

func TestEnvironmentDataSource_Configure_WrongType(t *testing.T) {
	d := &environmentDataSource{}

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

func TestEnvironmentDataSourceModel_Structure(t *testing.T) {
	// Test that the model can be created with expected fields
	model := environmentDataSourceModel{
		Name:           types.StringValue("production-k8s"),
		Type:           types.StringValue("K8S"),
		Description:    types.StringValue("Production cluster"),
		IncludeScaling: types.BoolValue(true),
		LastModifiedAt: types.Float64Value(1640000000.123456),
		LastReportedAt: types.Float64Value(1640000100.654321),
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
		t.Error("Expected IncludeScaling to be true")
	}

	if model.LastModifiedAt.ValueFloat64() != 1640000000.123456 {
		t.Errorf("Expected LastModifiedAt to be 1640000000.123456, got %f", model.LastModifiedAt.ValueFloat64())
	}

	if model.LastReportedAt.ValueFloat64() != 1640000100.654321 {
		t.Errorf("Expected LastReportedAt to be 1640000100.654321, got %f", model.LastReportedAt.ValueFloat64())
	}
}

func TestEnvironmentDataSourceModel_WithNullValues(t *testing.T) {
	// Test that the model handles null values correctly
	model := environmentDataSourceModel{
		Name:           types.StringValue("test-env"),
		Type:           types.StringValue("docker"),
		Description:    types.StringNull(),
		IncludeScaling: types.BoolValue(false),
		LastModifiedAt: types.Float64Value(1640000000.0),
		LastReportedAt: types.Float64Null(),
	}

	if model.Name.ValueString() != "test-env" {
		t.Error("Expected Name to be set correctly")
	}

	if !model.Description.IsNull() {
		t.Error("Expected Description to be null")
	}

	if !model.LastReportedAt.IsNull() {
		t.Error("Expected LastReportedAt to be null")
	}

	if model.IncludeScaling.ValueBool() != false {
		t.Error("Expected IncludeScaling to be false")
	}
}

func TestNewEnvironmentDataSource(t *testing.T) {
	d := NewEnvironmentDataSource()

	if d == nil {
		t.Fatal("Expected non-nil data source")
	}

	_, ok := d.(*environmentDataSource)
	if !ok {
		t.Error("Expected data source to be of type *environmentDataSource")
	}
}

func TestEnvironmentDataSource_Implements(t *testing.T) {
	// Verify the data source implements required interfaces
	var _ datasource.DataSource = &environmentDataSource{}
}

// Note: Full Read operation tests require acceptance testing (issue #72)
// These tests verify the data source structure and basic configuration,
// while acceptance tests will verify the full read operation against a real API.
//
// The Read method has 0% coverage in unit tests because it requires:
// - A real or mock Kosli API client
// - Terraform framework context (config, state)
// - Complex setup with request/response mocking
//
// This will be thoroughly tested in acceptance tests where:
// - Real environments are queried from a test Kosli organization
// - The full Terraform data source lifecycle is exercised
// - API integration is validated end-to-end
