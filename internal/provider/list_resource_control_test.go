package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestControlListResource_Metadata(t *testing.T) {
	l := &controlListResource{}

	req := resource.MetadataRequest{ProviderTypeName: "kosli"}
	resp := &resource.MetadataResponse{}

	l.Metadata(context.TODO(), req, resp)

	// The list resource type name must match the managed resource type name.
	expectedTypeName := "kosli_control"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestControlListResource_Schema(t *testing.T) {
	l := &controlListResource{}

	resp := &list.ListResourceSchemaResponse{}
	l.ListResourceConfigSchema(context.TODO(), list.ListResourceSchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	for _, attr := range []string{"search", "archived"} {
		a, exists := attrs[attr]
		if !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
			continue
		}
		if !a.IsOptional() {
			t.Errorf("Expected %q attribute to be optional", attr)
		}
		if a.GetMarkdownDescription() == "" {
			t.Errorf("Expected non-empty description on %q attribute", attr)
		}
	}
}

func TestControlListResource_Configure(t *testing.T) {
	l := &controlListResource{}

	resp := &resource.ConfigureResponse{}
	l.Configure(context.TODO(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}
	if l.client != nil {
		t.Error("Expected client to remain nil when provider data is nil")
	}
}

func TestControlListResource_Configure_WrongType(t *testing.T) {
	l := &controlListResource{}

	resp := &resource.ConfigureResponse{}
	l.Configure(context.TODO(), resource.ConfigureRequest{ProviderData: "wrong type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
}

func TestControlListResource_Implements(t *testing.T) {
	var _ list.ListResource = &controlListResource{}
	var _ list.ListResourceWithConfigure = &controlListResource{}
}

func TestNewControlListResource(t *testing.T) {
	l := NewControlListResource()
	if l == nil {
		t.Fatal("Expected non-nil list resource")
	}
	if _, ok := l.(*controlListResource); !ok {
		t.Error("Expected list resource to be of type *controlListResource")
	}
}

// newControlListRequest builds a list.ListRequest with the given filter config
// against the real control schemas, mirroring what the framework server does.
func newControlListRequest(t *testing.T, ctx context.Context, search, archived tftypes.Value, includeResource bool, limit int64) list.ListRequest {
	t.Helper()

	listSchemaResp := &list.ListResourceSchemaResponse{}
	(&controlListResource{}).ListResourceConfigSchema(ctx, list.ListResourceSchemaRequest{}, listSchemaResp)

	schemaResp := &resource.SchemaResponse{}
	(&controlResource{}).Schema(ctx, resource.SchemaRequest{}, schemaResp)

	identityResp := &resource.IdentitySchemaResponse{}
	(&controlResource{}).IdentitySchema(ctx, resource.IdentitySchemaRequest{}, identityResp)

	configType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"search":   tftypes.String,
		"archived": tftypes.Bool,
	}}

	return list.ListRequest{
		Config: tfsdk.Config{
			Schema: listSchemaResp.Schema,
			Raw: tftypes.NewValue(configType, map[string]tftypes.Value{
				"search":   search,
				"archived": archived,
			}),
		},
		IncludeResource:        includeResource,
		Limit:                  limit,
		ResourceSchema:         schemaResp.Schema,
		ResourceIdentitySchema: identityResp.IdentitySchema,
	}
}

// collectListResults drains the results stream into a slice.
func collectListResults(stream *list.ListResultsStream) []list.ListResult {
	var results []list.ListResult
	for result := range stream.Results {
		results = append(results, result)
	}
	return results
}

func TestControlListResource_List(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search") != "SDLC" {
			t.Errorf("expected search query param 'SDLC', got %q", r.URL.Query().Get("search"))
		}
		if r.URL.Query().Get("archived") != "" {
			t.Errorf("expected archived query param to be omitted, got %q", r.URL.Query().Get("archived"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"page": 1, "per_page": 100, "total_pages": 1, "total_count": 2,
			"controls": [
				{"identifier": "SDLC-001", "name": "Binary provenance", "version": 1, "created_at": 1234567890, "created_by": "user-1"},
				{"identifier": "SDLC-002", "name": "Peer review", "version": 2, "created_at": 1234567890, "created_by": "user-2"}
			]
		}`)
	}))
	defer server.Close()

	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	l := &controlListResource{client: c}
	req := newControlListRequest(t, ctx,
		tftypes.NewValue(tftypes.String, "SDLC"),
		tftypes.NewValue(tftypes.Bool, nil),
		true, 0)

	stream := &list.ListResultsStream{}
	l.List(ctx, req, stream)
	results := collectListResults(stream)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for i, expected := range []struct{ identifier, name string }{
		{"SDLC-001", "Binary provenance"},
		{"SDLC-002", "Peer review"},
	} {
		result := results[i]
		if result.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics on result %d: %v", i, result.Diagnostics)
		}
		if result.DisplayName != expected.name {
			t.Errorf("expected display name %q, got %q", expected.name, result.DisplayName)
		}

		var identity controlResourceIdentityModel
		if diags := result.Identity.Get(ctx, &identity); diags.HasError() {
			t.Fatalf("unexpected diagnostics reading identity: %v", diags)
		}
		if identity.Identifier.ValueString() != expected.identifier {
			t.Errorf("expected identity identifier %q, got %q", expected.identifier, identity.Identifier.ValueString())
		}

		var data controlResourceModel
		if diags := result.Resource.Get(ctx, &data); diags.HasError() {
			t.Fatalf("unexpected diagnostics reading resource data: %v", diags)
		}
		if data.Name.ValueString() != expected.name {
			t.Errorf("expected resource name %q, got %q", expected.name, data.Name.ValueString())
		}
		// The list endpoint never returns policies_referencing; it must be
		// normalized to an empty (known) list.
		if data.PoliciesReferencing.IsNull() || len(data.PoliciesReferencing.Elements()) != 0 {
			t.Errorf("expected empty policies_referencing, got %v", data.PoliciesReferencing)
		}
	}
}

func TestControlListResource_List_Limit(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"page": 1, "per_page": 100, "total_pages": 1, "total_count": 3,
			"controls": [
				{"identifier": "SDLC-001", "name": "One"},
				{"identifier": "SDLC-002", "name": "Two"},
				{"identifier": "SDLC-003", "name": "Three"}
			]
		}`)
	}))
	defer server.Close()

	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	l := &controlListResource{client: c}
	req := newControlListRequest(t, ctx,
		tftypes.NewValue(tftypes.String, nil),
		tftypes.NewValue(tftypes.Bool, nil),
		false, 2)

	stream := &list.ListResultsStream{}
	l.List(ctx, req, stream)
	results := collectListResults(stream)

	if len(results) != 2 {
		t.Fatalf("expected limit to cap results at 2, got %d", len(results))
	}
}

func TestControlListResource_List_Forbidden(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message": "Controls not enabled"}`)
	}))
	defer server.Close()

	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	l := &controlListResource{client: c}
	req := newControlListRequest(t, ctx,
		tftypes.NewValue(tftypes.String, nil),
		tftypes.NewValue(tftypes.Bool, nil),
		false, 0)

	stream := &list.ListResultsStream{}
	l.List(ctx, req, stream)
	results := collectListResults(stream)

	if len(results) != 1 {
		t.Fatalf("expected a single diagnostics result, got %d results", len(results))
	}
	if !results[0].Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	found := false
	for _, d := range results[0].Diagnostics.Errors() {
		if strings.Contains(d.Detail(), "beta feature") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected the beta hint in the error detail, got %v", results[0].Diagnostics)
	}
}
