package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLogicalEnvironmentDataSource_Metadata(t *testing.T) {
	d := &logicalEnvironmentDataSource{}

	// Test metadata
	req := datasource.MetadataRequest{
		ProviderTypeName: "kosli",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_logical_environment"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestLogicalEnvironmentDataSource_Schema(t *testing.T) {
	d := &logicalEnvironmentDataSource{}

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.TODO(), req, resp)

	// Verify schema has description
	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := resp.Schema.Attributes
	requiredAttrs := []string{"name", "type", "description", "included_environments", "last_modified_at"}
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

	// Verify included_environments is computed
	includedEnvsAttr := attrs["included_environments"]
	if includedEnvsAttr.IsComputed() == false {
		t.Error("Expected 'included_environments' attribute to be computed")
	}

	// Verify last_modified_at is computed
	lastModifiedAtAttr := attrs["last_modified_at"]
	if lastModifiedAtAttr.IsComputed() == false {
		t.Error("Expected 'last_modified_at' attribute to be computed")
	}
}

func TestLogicalEnvironmentDataSource_Configure(t *testing.T) {
	d := &logicalEnvironmentDataSource{}

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

func TestLogicalEnvironmentDataSource_Configure_WrongType(t *testing.T) {
	d := &logicalEnvironmentDataSource{}

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

func TestLogicalEnvironmentDataSourceModel_Structure(t *testing.T) {
	// Test that the model can be created with expected fields
	includedEnvs, diags := types.ListValueFrom(context.TODO(), types.StringType, []string{"prod-k8s", "prod-ecs"})
	if diags.HasError() {
		t.Fatalf("Unexpected diagnostics creating list: %v", diags)
	}

	model := logicalEnvironmentDataSourceModel{
		Name:                 types.StringValue("production-aggregate"),
		Type:                 types.StringValue("logical"),
		Description:          types.StringValue("All production environments"),
		IncludedEnvironments: includedEnvs,
		LastModifiedAt:       types.Float64Value(1640000000.123456),
	}

	if model.Name.ValueString() != "production-aggregate" {
		t.Error("Expected Name to be set correctly")
	}

	if model.Type.ValueString() != "logical" {
		t.Error("Expected Type to be set correctly")
	}

	if model.Description.ValueString() != "All production environments" {
		t.Error("Expected Description to be set correctly")
	}

	if model.IncludedEnvironments.IsNull() {
		t.Error("Expected IncludedEnvironments to be set")
	}

	// Verify list contents
	var envs []string
	diags = model.IncludedEnvironments.ElementsAs(context.TODO(), &envs, false)
	if diags.HasError() {
		t.Fatalf("Unexpected diagnostics converting IncludedEnvironments: %v", diags)
	}
	if len(envs) != 2 {
		t.Errorf("Expected 2 included environments, got %d", len(envs))
	}
	if envs[0] != "prod-k8s" || envs[1] != "prod-ecs" {
		t.Errorf("Expected ['prod-k8s', 'prod-ecs'], got %v", envs)
	}

	if model.LastModifiedAt.ValueFloat64() != 1640000000.123456 {
		t.Errorf("Expected LastModifiedAt to be 1640000000.123456, got %f", model.LastModifiedAt.ValueFloat64())
	}
}

func TestLogicalEnvironmentDataSourceModel_WithNullValues(t *testing.T) {
	// Test that the model handles null values correctly
	emptyList, diags := types.ListValueFrom(context.TODO(), types.StringType, []string{})
	if diags.HasError() {
		t.Fatalf("Unexpected diagnostics creating empty list: %v", diags)
	}

	model := logicalEnvironmentDataSourceModel{
		Name:                 types.StringValue("test-logical"),
		Type:                 types.StringValue("logical"),
		Description:          types.StringNull(),
		IncludedEnvironments: emptyList,
		LastModifiedAt:       types.Float64Value(1640000000.0),
	}

	if model.Name.ValueString() != "test-logical" {
		t.Error("Expected Name to be set correctly")
	}

	if !model.Description.IsNull() {
		t.Error("Expected Description to be null")
	}

	if model.IncludedEnvironments.IsNull() {
		t.Error("Expected IncludedEnvironments to not be null (but can be empty)")
	}

	// Verify empty list
	var envs []string
	diags = model.IncludedEnvironments.ElementsAs(context.TODO(), &envs, false)
	if diags.HasError() {
		t.Fatalf("Unexpected diagnostics converting IncludedEnvironments: %v", diags)
	}
	if len(envs) != 0 {
		t.Errorf("Expected 0 included environments, got %d", len(envs))
	}
}

func TestNewLogicalEnvironmentDataSource(t *testing.T) {
	d := NewLogicalEnvironmentDataSource()

	if d == nil {
		t.Fatal("Expected non-nil data source")
	}

	_, ok := d.(*logicalEnvironmentDataSource)
	if !ok {
		t.Error("Expected data source to be of type *logicalEnvironmentDataSource")
	}
}

func TestLogicalEnvironmentDataSource_Implements(t *testing.T) {
	// Verify the data source implements required interfaces
	var _ datasource.DataSource = &logicalEnvironmentDataSource{}
}

// Note: Full Read operation tests require acceptance testing
// These tests verify the data source structure and basic configuration,
// while acceptance tests will verify the full read operation against a real API.
//
// The Read method has 0% coverage in unit tests because it requires:
// - A real or mock Kosli API client
// - Terraform framework context (config, state)
// - Complex setup with request/response mocking
//
// This will be thoroughly tested in acceptance tests where:
// - Real logical environments are queried from a test Kosli organization
// - The full Terraform data source lifecycle is exercised
// - API integration is validated end-to-end
