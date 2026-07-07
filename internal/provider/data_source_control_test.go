package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestControlDataSource_Metadata(t *testing.T) {
	d := &controlDataSource{}

	req := datasource.MetadataRequest{ProviderTypeName: "kosli"}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_control"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestControlDataSource_Schema(t *testing.T) {
	d := &controlDataSource{}

	resp := &datasource.SchemaResponse{}
	d.Schema(context.TODO(), datasource.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	for _, attr := range []string{"identifier", "name", "description", "links", "version", "created_at", "created_by", "tags", "archived", "policies_referencing"} {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}

	if !attrs["identifier"].IsRequired() {
		t.Error("Expected 'identifier' attribute to be required")
	}
	if !attrs["archived"].IsOptional() {
		t.Error("Expected 'archived' attribute to be optional")
	}
	for _, attr := range []string{"name", "description", "links", "version", "created_at", "created_by", "tags", "archived", "policies_referencing"} {
		if !attrs[attr].IsComputed() {
			t.Errorf("Expected %q attribute to be computed", attr)
		}
	}
}

func TestControlDataSource_Configure(t *testing.T) {
	d := &controlDataSource{}

	resp := &datasource.ConfigureResponse{}
	d.Configure(context.TODO(), datasource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}
	if d.client != nil {
		t.Error("Expected client to remain nil when provider data is nil")
	}
}

func TestControlDataSource_Configure_WrongType(t *testing.T) {
	d := &controlDataSource{}

	resp := &datasource.ConfigureResponse{}
	d.Configure(context.TODO(), datasource.ConfigureRequest{ProviderData: "wrong type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
}

func TestControlDataSource_Implements(t *testing.T) {
	var _ datasource.DataSource = &controlDataSource{}
}

func TestNewControlDataSource(t *testing.T) {
	d := NewControlDataSource()
	if d == nil {
		t.Fatal("Expected non-nil data source")
	}
	if _, ok := d.(*controlDataSource); !ok {
		t.Error("Expected data source to be of type *controlDataSource")
	}
}
