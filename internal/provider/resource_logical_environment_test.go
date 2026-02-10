package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLogicalEnvironmentResource_Metadata(t *testing.T) {
	r := &logicalEnvironmentResource{}

	// Test metadata
	req := resource.MetadataRequest{
		ProviderTypeName: "kosli",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_logical_environment"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestLogicalEnvironmentResource_Schema(t *testing.T) {
	r := &logicalEnvironmentResource{}

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.TODO(), req, resp)

	// Verify schema has description
	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := resp.Schema.Attributes
	requiredAttrs := []string{"name", "type", "description", "included_environments"}
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

	// Verify type is computed (not configurable by user)
	typeAttr := attrs["type"]
	if typeAttr.IsComputed() == false {
		t.Error("Expected 'type' attribute to be computed")
	}
	if typeAttr.IsRequired() == true {
		t.Error("Expected 'type' attribute to not be required (it's computed)")
	}

	// Verify description is optional
	descAttr := attrs["description"]
	if descAttr.IsOptional() == false {
		t.Error("Expected 'description' attribute to be optional")
	}

	// Verify included_environments is required
	includedEnvsAttr := attrs["included_environments"]
	if includedEnvsAttr.IsRequired() == false {
		t.Error("Expected 'included_environments' attribute to be required")
	}
}

func TestLogicalEnvironmentResource_Configure(t *testing.T) {
	r := &logicalEnvironmentResource{}

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

func TestLogicalEnvironmentResource_Configure_WrongType(t *testing.T) {
	r := &logicalEnvironmentResource{}

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

func TestLogicalEnvironmentResourceModel_Structure(t *testing.T) {
	// Test that the model can be created with expected fields
	includedEnvs, _ := types.ListValueFrom(context.TODO(), types.StringType, []string{"prod-k8s", "prod-ecs"})

	model := logicalEnvironmentResourceModel{
		Name:                 types.StringValue("production-aggregate"),
		Type:                 types.StringValue("logical"),
		Description:          types.StringValue("All production environments"),
		IncludedEnvironments: includedEnvs,
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
	model.IncludedEnvironments.ElementsAs(context.TODO(), &envs, false)
	if len(envs) != 2 {
		t.Errorf("Expected 2 included environments, got %d", len(envs))
	}
	if envs[0] != "prod-k8s" || envs[1] != "prod-ecs" {
		t.Errorf("Expected ['prod-k8s', 'prod-ecs'], got %v", envs)
	}
}

func TestLogicalEnvironmentResourceModel_WithNullValues(t *testing.T) {
	// Test that the model handles null values correctly
	includedEnvs, _ := types.ListValueFrom(context.TODO(), types.StringType, []string{"test-env"})

	model := logicalEnvironmentResourceModel{
		Name:                 types.StringValue("test-logical"),
		Type:                 types.StringValue("logical"),
		Description:          types.StringNull(),
		IncludedEnvironments: includedEnvs,
	}

	if model.Name.ValueString() != "test-logical" {
		t.Error("Expected Name to be set correctly")
	}

	if !model.Description.IsNull() {
		t.Error("Expected Description to be null")
	}

	if model.IncludedEnvironments.IsNull() {
		t.Error("Expected IncludedEnvironments to not be null")
	}
}

func TestLogicalEnvironmentResourceModel_EmptyList(t *testing.T) {
	// Test handling of empty included_environments list
	emptyList, _ := types.ListValueFrom(context.TODO(), types.StringType, []string{})

	model := logicalEnvironmentResourceModel{
		Name:                 types.StringValue("test-env"),
		Type:                 types.StringValue("logical"),
		Description:          types.StringValue("Test"),
		IncludedEnvironments: emptyList,
	}

	if model.IncludedEnvironments.IsNull() {
		t.Error("Expected IncludedEnvironments to not be null even when empty")
	}

	var envs []string
	model.IncludedEnvironments.ElementsAs(context.TODO(), &envs, false)
	if len(envs) != 0 {
		t.Errorf("Expected 0 included environments, got %d", len(envs))
	}
}

func TestNewLogicalEnvironmentResource(t *testing.T) {
	r := NewLogicalEnvironmentResource()

	if r == nil {
		t.Fatal("Expected non-nil resource")
	}

	_, ok := r.(*logicalEnvironmentResource)
	if !ok {
		t.Error("Expected resource to be of type *logicalEnvironmentResource")
	}
}

func TestLogicalEnvironmentResource_Implements(t *testing.T) {
	// Verify the resource implements required interfaces
	var _ resource.Resource = &logicalEnvironmentResource{}
	var _ resource.ResourceWithImportState = &logicalEnvironmentResource{}
}

// Note: Full CRUD operation tests require acceptance testing
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
// - Validation of logical environment constraints is tested
