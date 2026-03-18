package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestPolicyResource_Metadata(t *testing.T) {
	r := &policyResource{}
	req := resource.MetadataRequest{ProviderTypeName: "kosli"}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.TODO(), req, resp)

	if resp.TypeName != "kosli_policy" {
		t.Errorf("expected TypeName 'kosli_policy', got %q", resp.TypeName)
	}
}

func TestPolicyResource_Schema(t *testing.T) {
	r := &policyResource{}
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.TODO(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned errors: %v", resp.Diagnostics)
	}

	attrs := resp.Schema.Attributes

	// Required attributes
	name, ok := attrs["name"]
	if !ok {
		t.Fatal("missing 'name' attribute")
	}
	if !name.IsRequired() {
		t.Error("'name' should be required")
	}

	content, ok := attrs["content"]
	if !ok {
		t.Fatal("missing 'content' attribute")
	}
	if !content.IsRequired() {
		t.Error("'content' should be required")
	}

	// Optional attributes
	desc, ok := attrs["description"]
	if !ok {
		t.Fatal("missing 'description' attribute")
	}
	if !desc.IsOptional() {
		t.Error("'description' should be optional")
	}

	// Computed attributes
	latestVersion, ok := attrs["latest_version"]
	if !ok {
		t.Fatal("missing 'latest_version' attribute")
	}
	if !latestVersion.IsComputed() {
		t.Error("'latest_version' should be computed")
	}

	createdAt, ok := attrs["created_at"]
	if !ok {
		t.Fatal("missing 'created_at' attribute")
	}
	if !createdAt.IsComputed() {
		t.Error("'created_at' should be computed")
	}
}

func TestPolicyResource_Configure_NilProviderData(t *testing.T) {
	r := &policyResource{}
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

func TestPolicyResource_Configure_ValidClient(t *testing.T) {
	r := &policyResource{}
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

func TestPolicyResource_Configure_WrongType(t *testing.T) {
	r := &policyResource{}
	req := resource.ConfigureRequest{ProviderData: "not-a-client"}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type")
	}
}

func TestMapPolicyToModel_EmptyDescription(t *testing.T) {
	policy := &client.Policy{
		Name:        "test-policy",
		Description: "",
		CreatedAt:   1700000000.0,
		Versions: []client.PolicyVersion{
			{Version: 1, Content: "_schema: https://kosli.com/schemas/policy/environment/v1\n"},
		},
	}

	var data policyResourceModel
	mapPolicyToModel(policy, &data)

	if !data.Description.IsNull() {
		t.Errorf("expected description to be null, got %q", data.Description.ValueString())
	}
	if data.Name.ValueString() != "test-policy" {
		t.Errorf("expected name 'test-policy', got %q", data.Name.ValueString())
	}
	if data.LatestVersion.ValueInt64() != 1 {
		t.Errorf("expected latest_version 1, got %d", data.LatestVersion.ValueInt64())
	}
}

func TestMapPolicyToModel_WithDescription(t *testing.T) {
	policy := &client.Policy{
		Name:        "test-policy",
		Description: "My policy",
		CreatedAt:   1700000000.0,
		Versions: []client.PolicyVersion{
			{Version: 3, Content: "some yaml content"},
		},
	}

	var data policyResourceModel
	mapPolicyToModel(policy, &data)

	if data.Description.ValueString() != "My policy" {
		t.Errorf("expected description 'My policy', got %q", data.Description.ValueString())
	}
	if data.LatestVersion.ValueInt64() != 3 {
		t.Errorf("expected latest_version 3, got %d", data.LatestVersion.ValueInt64())
	}
	if data.Content.ValueString() != "some yaml content" {
		t.Errorf("expected content 'some yaml content', got %q", data.Content.ValueString())
	}
}

func TestMapPolicyToModel_NoVersions(t *testing.T) {
	policy := &client.Policy{
		Name:      "test-policy",
		CreatedAt: 1700000000.0,
		Versions:  []client.PolicyVersion{},
	}

	var data policyResourceModel
	mapPolicyToModel(policy, &data)

	// latest_version and content remain zero/null — no panic
	if data.LatestVersion.ValueInt64() != 0 {
		t.Errorf("expected latest_version 0 when no versions, got %d", data.LatestVersion.ValueInt64())
	}
}
