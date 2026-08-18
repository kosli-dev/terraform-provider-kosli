package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
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
	requiredAttrs := []string{"name", "description", "schema", "jq_rules", "summary"}
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

	// Verify schema is optional
	schemaAttr := attrs["schema"]
	if schemaAttr.IsOptional() == false {
		t.Error("Expected 'schema' attribute to be optional")
	}

	// Verify jq_rules is optional
	jqRulesAttr := attrs["jq_rules"]
	if jqRulesAttr.IsOptional() == false {
		t.Error("Expected 'jq_rules' attribute to be optional")
	}

	// Verify summary is optional and uses the JSON custom type so that
	// formatting differences don't produce perpetual diffs
	summaryAttr := attrs["summary"]
	if summaryAttr.IsOptional() == false {
		t.Error("Expected 'summary' attribute to be optional")
	}
	if _, ok := summaryAttr.GetType().(jsontypes.NormalizedType); !ok {
		t.Errorf("Expected 'summary' to use jsontypes.NormalizedType, got %T", summaryAttr.GetType())
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

func TestCustomAttestationTypeResource_ToCreateRequest(t *testing.T) {
	jqRules, diags := types.ListValueFrom(context.TODO(), types.StringType, []string{".coverage >= 80"})
	if diags.HasError() {
		t.Fatalf("failed to build jq_rules list: %v", diags)
	}

	tests := []struct {
		name            string
		model           customAttestationTypeResourceModel
		expectedSummary string
	}{
		{
			name: "summary set",
			model: customAttestationTypeResourceModel{
				Name:        types.StringValue("test-type"),
				Description: types.StringValue("desc"),
				Schema:      jsontypes.NewNormalizedValue(`{"type":"object"}`),
				JqRules:     jqRules,
				Summary:     jsontypes.NewNormalizedValue(`[{"name":"Coverage","expression":".coverage"}]`),
			},
			expectedSummary: `[{"name":"Coverage","expression":".coverage"}]`,
		},
		{
			name: "summary null",
			model: customAttestationTypeResourceModel{
				Name:    types.StringValue("test-type"),
				JqRules: types.ListNull(types.StringType),
				Schema:  jsontypes.NewNormalizedNull(),
				Summary: jsontypes.NewNormalizedNull(),
			},
			expectedSummary: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diags diag.Diagnostics
			req := tt.model.toCreateRequest(context.TODO(), &diags)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if req.Summary != tt.expectedSummary {
				t.Errorf("expected Summary %q, got %q", tt.expectedSummary, req.Summary)
			}
			if req.Name != "test-type" {
				t.Errorf("expected Name 'test-type', got %q", req.Name)
			}
		})
	}
}

func TestCustomAttestationTypeResource_ApplyAPIResponse_Summary(t *testing.T) {
	tests := []struct {
		name        string
		apiSummary  string
		expectNull  bool
		expectValue string
	}{
		{name: "summary present", apiSummary: `[{"name":"Coverage","expression":".coverage"}]`, expectValue: `[{"name":"Coverage","expression":".coverage"}]`},
		{name: "summary absent", apiSummary: "", expectNull: true},
		{name: "summary empty array", apiSummary: "[]", expectValue: "[]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &customAttestationTypeResourceModel{Name: types.StringValue("test-type")}
			var diags diag.Diagnostics

			model.applyAPIResponse(context.TODO(), &client.CustomAttestationType{
				Name:    "test-type",
				Summary: tt.apiSummary,
			}, &diags)

			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if tt.expectNull {
				if !model.Summary.IsNull() {
					t.Errorf("expected Summary to be null, got %q", model.Summary.ValueString())
				}
				return
			}
			if model.Summary.IsNull() {
				t.Fatal("expected Summary to be set, got null")
			}
			if model.Summary.ValueString() != tt.expectValue {
				t.Errorf("expected Summary %q, got %q", tt.expectValue, model.Summary.ValueString())
			}
		})
	}
}
