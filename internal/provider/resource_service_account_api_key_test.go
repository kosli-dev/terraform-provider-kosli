package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestServiceAccountAPIKeyResource_Metadata(t *testing.T) {
	r := &serviceAccountAPIKeyResource{}

	resp := &resource.MetadataResponse{}
	r.Metadata(context.TODO(), resource.MetadataRequest{ProviderTypeName: "kosli"}, resp)

	expectedTypeName := "kosli_service_account_api_key"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestServiceAccountAPIKeyResource_Schema(t *testing.T) {
	r := &serviceAccountAPIKeyResource{}

	resp := &resource.SchemaResponse{}
	r.Schema(context.TODO(), resource.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	for _, attr := range []string{"service_account_name", "description", "expires_at", "id", "key", "created_at", "last_used_at"} {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}

	if !attrs["service_account_name"].IsRequired() {
		t.Error("Expected 'service_account_name' attribute to be required")
	}
	if !attrs["description"].IsRequired() {
		t.Error("Expected 'description' attribute to be required")
	}
	if !attrs["expires_at"].IsOptional() {
		t.Error("Expected 'expires_at' attribute to be optional")
	}
	if !attrs["key"].IsComputed() {
		t.Error("Expected 'key' attribute to be computed")
	}
	if !attrs["key"].IsSensitive() {
		t.Error("Expected 'key' attribute to be sensitive")
	}
}

func TestServiceAccountAPIKeyResource_Configure(t *testing.T) {
	r := &serviceAccountAPIKeyResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}
}

func TestServiceAccountAPIKeyResource_Configure_WrongType(t *testing.T) {
	r := &serviceAccountAPIKeyResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), resource.ConfigureRequest{ProviderData: 42}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
}

func TestServiceAccountAPIKeyResource_Implements(t *testing.T) {
	var _ resource.Resource = &serviceAccountAPIKeyResource{}
	var _ resource.ResourceWithImportState = &serviceAccountAPIKeyResource{}
}

func TestNewServiceAccountAPIKeyResource(t *testing.T) {
	r := NewServiceAccountAPIKeyResource()
	if r == nil {
		t.Fatal("Expected non-nil resource")
	}
	if _, ok := r.(*serviceAccountAPIKeyResource); !ok {
		t.Error("Expected resource to be of type *serviceAccountAPIKeyResource")
	}
}

func TestMapAPIKeyToState_PreservesKey(t *testing.T) {
	// The Key field must never be overwritten by the mapping helper (the raw
	// key is only available at creation time).
	data := serviceAccountAPIKeyResourceModel{}
	data.Key = types.StringValue("secret-from-create")

	mapAPIKeyToState(&client.ServiceAccountAPIKey{
		ID:          "key-1",
		Description: "prod key",
		CreatedAt:   1234567890,
		ExpiresAt:   4102444800,
		LastUsedAt:  1234567900,
	}, &data)

	if data.Key.ValueString() != "secret-from-create" {
		t.Errorf("expected key to be preserved, got %q", data.Key.ValueString())
	}
	if data.ID.ValueString() != "key-1" {
		t.Errorf("expected id 'key-1', got %q", data.ID.ValueString())
	}
	if data.ExpiresAt.ValueInt64() != 4102444800 {
		t.Errorf("expected expires_at 4102444800, got %d", data.ExpiresAt.ValueInt64())
	}
}
