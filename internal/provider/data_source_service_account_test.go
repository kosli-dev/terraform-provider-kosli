package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestServiceAccountDataSource_Metadata(t *testing.T) {
	d := &serviceAccountDataSource{}

	resp := &datasource.MetadataResponse{}
	d.Metadata(context.TODO(), datasource.MetadataRequest{ProviderTypeName: "kosli"}, resp)

	expectedTypeName := "kosli_service_account"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestServiceAccountDataSource_Schema(t *testing.T) {
	d := &serviceAccountDataSource{}

	resp := &datasource.SchemaResponse{}
	d.Schema(context.TODO(), datasource.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	for _, attr := range []string{"name", "description", "privilege", "display_name", "creating_user_id", "created_at", "for_webhook"} {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}

	if !attrs["name"].IsRequired() {
		t.Error("Expected 'name' attribute to be required")
	}
	if !attrs["privilege"].IsComputed() {
		t.Error("Expected 'privilege' attribute to be computed")
	}
}

func TestServiceAccountDataSource_Configure(t *testing.T) {
	d := &serviceAccountDataSource{}

	resp := &datasource.ConfigureResponse{}
	d.Configure(context.TODO(), datasource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}
}

func TestServiceAccountDataSource_Configure_WrongType(t *testing.T) {
	d := &serviceAccountDataSource{}

	resp := &datasource.ConfigureResponse{}
	d.Configure(context.TODO(), datasource.ConfigureRequest{ProviderData: "wrong type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
}

func TestServiceAccountDataSource_Implements(t *testing.T) {
	var _ datasource.DataSource = &serviceAccountDataSource{}
}

func TestNewServiceAccountDataSource(t *testing.T) {
	d := NewServiceAccountDataSource()
	if d == nil {
		t.Fatal("Expected non-nil data source")
	}
	if _, ok := d.(*serviceAccountDataSource); !ok {
		t.Error("Expected data source to be of type *serviceAccountDataSource")
	}
}
