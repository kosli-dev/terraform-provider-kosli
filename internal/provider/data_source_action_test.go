package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestActionDataSource_Metadata(t *testing.T) {
	d := &actionDataSource{}

	req := datasource.MetadataRequest{ProviderTypeName: "kosli"}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.TODO(), req, resp)

	if resp.TypeName != "kosli_action" {
		t.Errorf("Expected TypeName %q, got %q", "kosli_action", resp.TypeName)
	}
}

func TestActionDataSource_Schema(t *testing.T) {
	d := &actionDataSource{}

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.TODO(), req, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	expectedAttrs := []string{"name", "environments", "triggers", "number", "created_by", "last_modified_at"}
	for _, attr := range expectedAttrs {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}

	if !attrs["name"].IsRequired() {
		t.Error("Expected 'name' to be required")
	}
	if !attrs["environments"].IsComputed() {
		t.Error("Expected 'environments' to be computed")
	}
	if !attrs["triggers"].IsComputed() {
		t.Error("Expected 'triggers' to be computed")
	}
	if !attrs["number"].IsComputed() {
		t.Error("Expected 'number' to be computed")
	}
	if !attrs["created_by"].IsComputed() {
		t.Error("Expected 'created_by' to be computed")
	}
	if !attrs["last_modified_at"].IsComputed() {
		t.Error("Expected 'last_modified_at' to be computed")
	}
}

func TestActionDataSource_Configure(t *testing.T) {
	d := &actionDataSource{}

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

func TestActionDataSource_Configure_WrongType(t *testing.T) {
	d := &actionDataSource{}

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

func TestActionDataSourceModel_Structure(t *testing.T) {
	envList, _ := types.ListValueFrom(context.TODO(), types.StringType, []string{"production"})
	trigList, _ := types.ListValueFrom(context.TODO(), types.StringType, []string{"ON_NON_COMPLIANT_ENV"})

	model := actionDataSourceModel{
		Name:           types.StringValue("compliance-alerts"),
		Environments:   envList,
		Triggers:       trigList,
		Number:         types.Int64Value(42),
		CreatedBy:      types.StringValue("user@example.com"),
		LastModifiedAt: types.Float64Value(1633123457.123),
	}

	if model.Name.ValueString() != "compliance-alerts" {
		t.Error("Expected Name to be set correctly")
	}
	if model.Number.ValueInt64() != 42 {
		t.Error("Expected Number to be 42")
	}
	if model.CreatedBy.ValueString() != "user@example.com" {
		t.Error("Expected CreatedBy to be set correctly")
	}
	if model.LastModifiedAt.ValueFloat64() != 1633123457.123 {
		t.Errorf("Expected LastModifiedAt 1633123457.123, got %f", model.LastModifiedAt.ValueFloat64())
	}
}

func TestNewActionDataSource(t *testing.T) {
	d := NewActionDataSource()
	if d == nil {
		t.Fatal("Expected non-nil data source")
	}
	if _, ok := d.(*actionDataSource); !ok {
		t.Error("Expected data source to be of type *actionDataSource")
	}
}

func TestActionDataSource_Implements(t *testing.T) {
	var _ datasource.DataSource = &actionDataSource{}
}
