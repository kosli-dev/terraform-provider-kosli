package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPolicyDataSource_Metadata(t *testing.T) {
	d := &policyDataSource{}

	req := datasource.MetadataRequest{ProviderTypeName: "kosli"}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.TODO(), req, resp)

	if resp.TypeName != "kosli_policy" {
		t.Errorf("Expected TypeName %q, got %q", "kosli_policy", resp.TypeName)
	}
}

func TestPolicyDataSource_Schema(t *testing.T) {
	d := &policyDataSource{}

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.TODO(), req, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	expectedAttrs := []string{"name", "description", "content", "latest_version", "created_at"}
	for _, attr := range expectedAttrs {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}

	if !attrs["name"].IsRequired() {
		t.Error("Expected 'name' to be required")
	}
	if !attrs["description"].IsComputed() {
		t.Error("Expected 'description' to be computed")
	}
	if !attrs["content"].IsComputed() {
		t.Error("Expected 'content' to be computed")
	}
	if !attrs["latest_version"].IsComputed() {
		t.Error("Expected 'latest_version' to be computed")
	}
	if !attrs["created_at"].IsComputed() {
		t.Error("Expected 'created_at' to be computed")
	}
}

func TestPolicyDataSource_Configure(t *testing.T) {
	d := &policyDataSource{}

	req := datasource.ConfigureRequest{ProviderData: nil}
	resp := &datasource.ConfigureResponse{}

	d.Configure(context.TODO(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}
	if d.client != nil {
		t.Error("Expected client to remain nil when provider data is nil")
	}
}

func TestPolicyDataSource_Configure_WrongType(t *testing.T) {
	d := &policyDataSource{}

	req := datasource.ConfigureRequest{ProviderData: "wrong type"}
	resp := &datasource.ConfigureResponse{}

	d.Configure(context.TODO(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
	if d.client != nil {
		t.Error("Expected client to remain nil when provider data is wrong type")
	}
}

func TestPolicyDataSourceModel_Structure(t *testing.T) {
	model := policyDataSourceModel{
		Name:          types.StringValue("prod-requirements"),
		Description:   types.StringValue("Production policy"),
		Content:       types.StringValue("_schema: https://kosli.com/schemas/policy/environment/v1\n"),
		LatestVersion: types.Int64Value(3),
		CreatedAt:     types.Float64Value(1633123457.123),
	}

	if model.Name.ValueString() != "prod-requirements" {
		t.Error("Expected Name to be set correctly")
	}
	if model.Description.ValueString() != "Production policy" {
		t.Error("Expected Description to be set correctly")
	}
	if model.LatestVersion.ValueInt64() != 3 {
		t.Error("Expected LatestVersion to be 3")
	}
	if model.CreatedAt.ValueFloat64() != 1633123457.123 {
		t.Errorf("Expected CreatedAt 1633123457.123, got %f", model.CreatedAt.ValueFloat64())
	}
}

func TestNewPolicyDataSource(t *testing.T) {
	d := NewPolicyDataSource()
	if d == nil {
		t.Fatal("Expected non-nil data source")
	}
	if _, ok := d.(*policyDataSource); !ok {
		t.Error("Expected data source to be of type *policyDataSource")
	}
}

func TestPolicyDataSource_Implements(t *testing.T) {
	var _ datasource.DataSource = &policyDataSource{}
}
