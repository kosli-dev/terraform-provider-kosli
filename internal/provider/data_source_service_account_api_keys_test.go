package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestServiceAccountAPIKeysDataSource_Metadata(t *testing.T) {
	d := &serviceAccountAPIKeysDataSource{}

	resp := &datasource.MetadataResponse{}
	d.Metadata(context.TODO(), datasource.MetadataRequest{ProviderTypeName: "kosli"}, resp)

	expectedTypeName := "kosli_service_account_api_keys"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestServiceAccountAPIKeysDataSource_Schema(t *testing.T) {
	d := &serviceAccountAPIKeysDataSource{}

	resp := &datasource.SchemaResponse{}
	d.Schema(context.TODO(), datasource.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	for _, attr := range []string{"service_account_name", "keys"} {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}

	if !attrs["service_account_name"].IsRequired() {
		t.Error("Expected 'service_account_name' attribute to be required")
	}
	if !attrs["keys"].IsComputed() {
		t.Error("Expected 'keys' attribute to be computed")
	}
}

func TestServiceAccountAPIKeysDataSource_Configure(t *testing.T) {
	d := &serviceAccountAPIKeysDataSource{}

	resp := &datasource.ConfigureResponse{}
	d.Configure(context.TODO(), datasource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}
}

func TestServiceAccountAPIKeysDataSource_Configure_WrongType(t *testing.T) {
	d := &serviceAccountAPIKeysDataSource{}

	resp := &datasource.ConfigureResponse{}
	d.Configure(context.TODO(), datasource.ConfigureRequest{ProviderData: "wrong type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
}

func TestServiceAccountAPIKeysDataSource_Implements(t *testing.T) {
	var _ datasource.DataSource = &serviceAccountAPIKeysDataSource{}
}

func TestNewServiceAccountAPIKeysDataSource(t *testing.T) {
	d := NewServiceAccountAPIKeysDataSource()
	if d == nil {
		t.Fatal("Expected non-nil data source")
	}
	if _, ok := d.(*serviceAccountAPIKeysDataSource); !ok {
		t.Error("Expected data source to be of type *serviceAccountAPIKeysDataSource")
	}
}
