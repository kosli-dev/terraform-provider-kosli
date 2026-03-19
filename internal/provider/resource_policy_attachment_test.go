package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestPolicyAttachmentResource_Metadata(t *testing.T) {
	r := &policyAttachmentResource{}
	req := resource.MetadataRequest{ProviderTypeName: "kosli"}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.TODO(), req, resp)

	if resp.TypeName != "kosli_policy_attachment" {
		t.Errorf("expected TypeName 'kosli_policy_attachment', got %q", resp.TypeName)
	}
}

func TestPolicyAttachmentResource_Schema(t *testing.T) {
	r := &policyAttachmentResource{}
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.TODO(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned errors: %v", resp.Diagnostics)
	}

	attrs := resp.Schema.Attributes

	// environment_name: required
	envName, ok := attrs["environment_name"]
	if !ok {
		t.Fatal("missing 'environment_name' attribute")
	}
	if !envName.IsRequired() {
		t.Error("'environment_name' should be required")
	}

	// policy_name: required
	policyName, ok := attrs["policy_name"]
	if !ok {
		t.Fatal("missing 'policy_name' attribute")
	}
	if !policyName.IsRequired() {
		t.Error("'policy_name' should be required")
	}
}

func TestPolicyAttachmentResource_Configure_NilProviderData(t *testing.T) {
	r := &policyAttachmentResource{}
	req := resource.ConfigureRequest{ProviderData: nil}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error with nil provider data: %v", resp.Diagnostics)
	}
	if r.client != nil {
		t.Error("client should remain nil when provider data is nil")
	}
}

func TestPolicyAttachmentResource_Configure_ValidClient(t *testing.T) {
	r := &policyAttachmentResource{}
	c, err := client.NewClient("token", "org")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req := resource.ConfigureRequest{ProviderData: c}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
	if r.client == nil {
		t.Error("client should be set after configure")
	}
}

func TestPolicyAttachmentResource_Configure_WrongType(t *testing.T) {
	r := &policyAttachmentResource{}
	req := resource.ConfigureRequest{ProviderData: "not-a-client"}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type")
	}
}

func TestPolicyAttachmentResourceModel_Fields(t *testing.T) {
	data := policyAttachmentResourceModel{
		EnvironmentName: types.StringValue("my-env"),
		PolicyName:      types.StringValue("my-policy"),
	}

	if data.EnvironmentName.ValueString() != "my-env" {
		t.Errorf("expected EnvironmentName 'my-env', got %q", data.EnvironmentName.ValueString())
	}
	if data.PolicyName.ValueString() != "my-policy" {
		t.Errorf("expected PolicyName 'my-policy', got %q", data.PolicyName.ValueString())
	}
}
