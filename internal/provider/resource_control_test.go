package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestControlResource_Metadata(t *testing.T) {
	r := &controlResource{}

	req := resource.MetadataRequest{ProviderTypeName: "kosli"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_control"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestControlResource_Schema(t *testing.T) {
	r := &controlResource{}

	resp := &resource.SchemaResponse{}
	r.Schema(context.TODO(), resource.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	for _, attr := range []string{"identifier", "name", "description", "links", "version", "created_at", "created_by", "tags", "policies_referencing"} {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}

	if !attrs["identifier"].IsRequired() {
		t.Error("Expected 'identifier' attribute to be required")
	}
	if !attrs["name"].IsRequired() {
		t.Error("Expected 'name' attribute to be required")
	}
	if !attrs["description"].IsOptional() {
		t.Error("Expected 'description' attribute to be optional")
	}
	if !attrs["links"].IsOptional() {
		t.Error("Expected 'links' attribute to be optional")
	}
	for _, attr := range []string{"version", "created_at", "created_by", "tags", "policies_referencing"} {
		if !attrs[attr].IsComputed() {
			t.Errorf("Expected %q attribute to be computed", attr)
		}
	}
}

func TestControlResource_Configure(t *testing.T) {
	r := &controlResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}
	if r.client != nil {
		t.Error("Expected client to remain nil when provider data is nil")
	}
}

func TestControlResource_Configure_WrongType(t *testing.T) {
	r := &controlResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), resource.ConfigureRequest{ProviderData: "wrong type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
}

func TestControlResource_Implements(t *testing.T) {
	var _ resource.Resource = &controlResource{}
	var _ resource.ResourceWithImportState = &controlResource{}
}

func TestNewControlResource(t *testing.T) {
	r := NewControlResource()
	if r == nil {
		t.Fatal("Expected non-nil resource")
	}
	if _, ok := r.(*controlResource); !ok {
		t.Error("Expected resource to be of type *controlResource")
	}
}

func TestControlIdentifierRegexp(t *testing.T) {
	valid := []string{"SDLC-001", "a", "1control", "sdlc.review_v1~beta"}
	for _, id := range valid {
		if !controlIdentifierRegexp.MatchString(id) {
			t.Errorf("expected identifier %q to be valid", id)
		}
	}

	invalid := []string{"-leading-hyphen", ".leading-dot", "has space", "has/slash", ""}
	for _, id := range invalid {
		if controlIdentifierRegexp.MatchString(id) {
			t.Errorf("expected identifier %q to be invalid", id)
		}
	}
}

func TestMapControlToState(t *testing.T) {
	ctx := context.Background()

	// Empty description and links map to null to avoid drift; nil computed
	// collections normalize to empty.
	data := controlResourceModel{}
	var diags diag.Diagnostics
	mapControlToState(ctx, &client.Control{
		Identifier: "SDLC-001",
		Name:       "Binary provenance",
		Version:    1,
		CreatedAt:  1234567890,
		CreatedBy:  "user-123",
	}, &data, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.Identifier.ValueString() != "SDLC-001" {
		t.Errorf("expected identifier 'SDLC-001', got %q", data.Identifier.ValueString())
	}
	if data.Name.ValueString() != "Binary provenance" {
		t.Errorf("expected name 'Binary provenance', got %q", data.Name.ValueString())
	}
	if !data.Description.IsNull() {
		t.Error("expected empty description to map to null")
	}
	if !data.Links.IsNull() {
		t.Error("expected empty links to map to null")
	}
	if data.Version.ValueInt64() != 1 {
		t.Errorf("expected version 1, got %d", data.Version.ValueInt64())
	}
	if data.CreatedAt.ValueString() != "2009-02-13T23:31:30Z" {
		t.Errorf("expected created_at '2009-02-13T23:31:30Z', got %q", data.CreatedAt.ValueString())
	}
	if data.Tags.IsNull() || len(data.Tags.Elements()) != 0 {
		t.Errorf("expected empty tags map, got %v", data.Tags)
	}
	if data.PoliciesReferencing.IsNull() || len(data.PoliciesReferencing.Elements()) != 0 {
		t.Errorf("expected empty policies_referencing list, got %v", data.PoliciesReferencing)
	}

	// Non-empty values should be preserved.
	data2 := controlResourceModel{}
	var diags2 diag.Diagnostics
	mapControlToState(ctx, &client.Control{
		Identifier:          "SDLC-001",
		Name:                "Binary provenance",
		Description:         "desc",
		Links:               map[string]string{"docs": "https://example.com"},
		Version:             2,
		Tags:                map[string]string{"framework": "finos-sdlc"},
		PoliciesReferencing: []string{"prod-policy"},
	}, &data2, &diags2)
	if diags2.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags2)
	}
	if data2.Description.ValueString() != "desc" {
		t.Errorf("expected description 'desc', got %q", data2.Description.ValueString())
	}
	if len(data2.Links.Elements()) != 1 {
		t.Errorf("expected 1 link, got %v", data2.Links)
	}
	if len(data2.Tags.Elements()) != 1 {
		t.Errorf("expected 1 tag, got %v", data2.Tags)
	}
	if len(data2.PoliciesReferencing.Elements()) != 1 {
		t.Errorf("expected 1 referencing policy, got %v", data2.PoliciesReferencing)
	}

	// Explicitly configured empty values must round-trip as-is (not be
	// normalized to null), or Terraform reports an inconsistent result.
	emptyLinks, d := types.MapValueFrom(ctx, types.StringType, map[string]string{})
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}
	data3 := controlResourceModel{
		Description: types.StringValue(""),
		Links:       emptyLinks,
	}
	var diags3 diag.Diagnostics
	mapControlToState(ctx, &client.Control{
		Identifier: "SDLC-001",
		Name:       "Binary provenance",
	}, &data3, &diags3)
	if diags3.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags3)
	}
	if data3.Description.IsNull() {
		t.Error("expected explicitly configured empty description to stay \"\", got null")
	}
	if data3.Links.IsNull() {
		t.Error("expected explicitly configured empty links to stay {}, got null")
	}

	// Unknown prior values must normalize to null, not be treated as set.
	data4 := controlResourceModel{
		Description: types.StringUnknown(),
		Links:       types.MapUnknown(types.StringType),
	}
	var diags4 diag.Diagnostics
	mapControlToState(ctx, &client.Control{
		Identifier: "SDLC-001",
		Name:       "Binary provenance",
	}, &data4, &diags4)
	if diags4.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags4)
	}
	if !data4.Description.IsNull() {
		t.Error("expected unknown prior description to normalize to null")
	}
	if !data4.Links.IsNull() {
		t.Error("expected unknown prior links to normalize to null")
	}
}
