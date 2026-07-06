package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestServiceAccountResource_Metadata(t *testing.T) {
	r := &serviceAccountResource{}

	req := resource.MetadataRequest{ProviderTypeName: "kosli"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_service_account"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestServiceAccountResource_Schema(t *testing.T) {
	r := &serviceAccountResource{}

	resp := &resource.SchemaResponse{}
	r.Schema(context.TODO(), resource.SchemaRequest{}, resp)

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
	if !attrs["privilege"].IsRequired() {
		t.Error("Expected 'privilege' attribute to be required")
	}
	if !attrs["description"].IsOptional() {
		t.Error("Expected 'description' attribute to be optional")
	}
	if !attrs["created_at"].IsComputed() {
		t.Error("Expected 'created_at' attribute to be computed")
	}
}

func TestServiceAccountResource_Configure(t *testing.T) {
	r := &serviceAccountResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}
	if r.client != nil {
		t.Error("Expected client to remain nil when provider data is nil")
	}
}

func TestServiceAccountResource_Configure_WrongType(t *testing.T) {
	r := &serviceAccountResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), resource.ConfigureRequest{ProviderData: "wrong type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
}

func TestServiceAccountResource_Implements(t *testing.T) {
	var _ resource.Resource = &serviceAccountResource{}
	var _ resource.ResourceWithImportState = &serviceAccountResource{}
}

func TestNewServiceAccountResource(t *testing.T) {
	r := NewServiceAccountResource()
	if r == nil {
		t.Fatal("Expected non-nil resource")
	}
	if _, ok := r.(*serviceAccountResource); !ok {
		t.Error("Expected resource to be of type *serviceAccountResource")
	}
}

func TestMapServiceAccountToState(t *testing.T) {
	// Empty description maps to null to avoid drift.
	data := serviceAccountResourceModel{}
	mapServiceAccountToState(&client.ServiceAccount{
		Name:      "ci",
		Privilege: "member",
		CreatedAt: 1234567890,
	}, &data)

	if data.Name.ValueString() != "ci" {
		t.Errorf("expected name 'ci', got %q", data.Name.ValueString())
	}
	if !data.Description.IsNull() {
		t.Error("expected empty description to map to null")
	}
	if data.Privilege.ValueString() != "member" {
		t.Errorf("expected privilege 'member', got %q", data.Privilege.ValueString())
	}
	if data.CreatedAt.ValueFloat64() != 1234567890 {
		t.Errorf("expected created_at 1234567890, got %v", data.CreatedAt.ValueFloat64())
	}

	// Non-empty description should be preserved.
	data2 := serviceAccountResourceModel{}
	mapServiceAccountToState(&client.ServiceAccount{
		Name:        "ci",
		Description: "desc",
		Privilege:   "admin",
		ForWebhook:  true,
	}, &data2)
	if data2.Description.ValueString() != "desc" {
		t.Errorf("expected description 'desc', got %q", data2.Description.ValueString())
	}
	if !data2.ForWebhook.ValueBool() {
		t.Error("expected for_webhook to be true")
	}
}
