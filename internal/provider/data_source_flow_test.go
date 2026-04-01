package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestFlowDataSource_Metadata(t *testing.T) {
	d := &flowDataSource{}

	req := datasource.MetadataRequest{ProviderTypeName: "kosli"}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.TODO(), req, resp)

	if resp.TypeName != "kosli_flow" {
		t.Errorf("Expected TypeName %q, got %q", "kosli_flow", resp.TypeName)
	}
}

func TestFlowDataSource_Schema(t *testing.T) {
	d := &flowDataSource{}

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.TODO(), req, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	expectedAttrs := []string{"name", "description", "template"}
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
	if !attrs["template"].IsComputed() {
		t.Error("Expected 'template' to be computed")
	}
}

func TestFlowDataSource_Configure_NilProviderData(t *testing.T) {
	d := &flowDataSource{}

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

func TestFlowDataSource_Configure_WrongType(t *testing.T) {
	d := &flowDataSource{}

	req := datasource.ConfigureRequest{ProviderData: "wrong-type"}
	resp := &datasource.ConfigureResponse{}

	d.Configure(context.TODO(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
}

func TestNewFlowDataSource(t *testing.T) {
	d := NewFlowDataSource()
	if d == nil {
		t.Fatal("Expected non-nil data source")
	}
	if _, ok := d.(*flowDataSource); !ok {
		t.Error("Expected data source to be of type *flowDataSource")
	}
}
