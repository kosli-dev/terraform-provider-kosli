package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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
		LastUsedAt:  0,
	}, &data)

	if data.Key.ValueString() != "secret-from-create" {
		t.Errorf("expected key to be preserved, got %q", data.Key.ValueString())
	}
	if data.ID.ValueString() != "key-1" {
		t.Errorf("expected id 'key-1', got %q", data.ID.ValueString())
	}
	// Timestamps are rendered RFC3339 UTC; 0 means "never" and maps to null.
	if data.ExpiresAt.ValueString() != "2100-01-01T00:00:00Z" {
		t.Errorf("expected expires_at '2100-01-01T00:00:00Z', got %q", data.ExpiresAt.ValueString())
	}
	if data.CreatedAt.ValueString() != "2009-02-13T23:31:30Z" {
		t.Errorf("expected created_at '2009-02-13T23:31:30Z', got %q", data.CreatedAt.ValueString())
	}
	if !data.LastUsedAt.IsNull() {
		t.Errorf("expected never-used last_used_at to be null, got %q", data.LastUsedAt.ValueString())
	}
}

// newAPIKeyCreateReqResp builds a Create request whose plan carries the given
// expires_at, plus a response with an empty state of the right schema.
func newAPIKeyCreateReqResp(t *testing.T, ctx context.Context, expiresAt tftypes.Value) (resource.CreateRequest, *resource.CreateResponse) {
	t.Helper()

	schemaResp := &resource.SchemaResponse{}
	(&serviceAccountAPIKeyResource{}).Schema(ctx, resource.SchemaRequest{}, schemaResp)

	// Derived from the schema rather than restated, so the helper stays valid
	// when an attribute is added to the resource without touching this file.
	objType := schemaResp.Schema.Type().TerraformType(ctx)

	plan := tftypes.NewValue(objType, map[string]tftypes.Value{
		"service_account_name": tftypes.NewValue(tftypes.String, "ci"),
		"description":          tftypes.NewValue(tftypes.String, "test key"),
		"expires_at":           expiresAt,
		"id":                   tftypes.NewValue(tftypes.String, nil),
		"key":                  tftypes.NewValue(tftypes.String, nil),
		"created_at":           tftypes.NewValue(tftypes.String, nil),
		"last_used_at":         tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: plan}}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	return req, resp
}

// apiKeyCreateServer serves a create response that issues the key with
// issuedExpiry, whatever expiry was requested.
func apiKeyCreateServer(t *testing.T, issuedExpiry int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id": "key-1", "key": "raw-secret", "description": "test key",
			"created_at": 1700000000, "expires_at": %d, "last_used_at": 0}`, issuedExpiry)
	}))
}

// TestServiceAccountAPIKeyResource_Create_ExpiryShortened covers the server
// silently capping a requested expiry: the apply must fail with an actionable
// diagnostic rather than Terraform's generic "inconsistent result" error, and
// the key the server did issue must still be recorded in state.
func TestServiceAccountAPIKeyResource_Create_ExpiryShortened(t *testing.T) {
	ctx := context.TODO()

	// Requested 2100-01-01; the server issues a far earlier expiry instead.
	const issued = 1800000000
	server := apiKeyCreateServer(t, issued)
	defer server.Close()

	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	r := &serviceAccountAPIKeyResource{client: c}
	req, resp := newAPIKeyCreateReqResp(t, ctx,
		tftypes.NewValue(tftypes.String, "2100-01-01T00:00:00Z"))

	r.Create(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic when the server shortens the expiry")
	}
	summary := resp.Diagnostics.Errors()[0].Summary()
	if !strings.Contains(summary, "Not Honoured") {
		t.Errorf("expected a summary naming the unhonoured expiry, got %q", summary)
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	for _, want := range []string{"2100-01-01T00:00:00Z", "expires_at"} {
		if !strings.Contains(detail, want) {
			t.Errorf("expected detail to mention %q, got %q", want, detail)
		}
	}

	// The key exists server-side, so it must be in state rather than orphaned.
	var id *string
	if diags := resp.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
		t.Fatalf("unexpected error reading id from state: %v", diags)
	}
	if id == nil || *id != "key-1" {
		t.Errorf("expected the issued key to be saved to state, got id %v", id)
	}
}

// TestServiceAccountAPIKeyResource_Create_ExpiryHonoured is the companion case:
// an expiry the server accepts unchanged must not raise the diagnostic.
func TestServiceAccountAPIKeyResource_Create_ExpiryHonoured(t *testing.T) {
	ctx := context.TODO()

	// 2027-01-01T00:00:00Z, echoed back unchanged.
	const issued = 1798761600
	server := apiKeyCreateServer(t, issued)
	defer server.Close()

	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	r := &serviceAccountAPIKeyResource{client: c}
	req, resp := newAPIKeyCreateReqResp(t, ctx,
		tftypes.NewValue(tftypes.String, "2027-01-01T00:00:00Z"))

	r.Create(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no error when the expiry is honoured, got: %v", resp.Diagnostics.Errors())
	}
}

// TestServiceAccountAPIKeyResource_Create_ExpiryOmitted verifies the check does
// not fire when expires_at is absent: the server picks the expiry, so whatever
// it returns is by definition not a mismatch.
func TestServiceAccountAPIKeyResource_Create_ExpiryOmitted(t *testing.T) {
	ctx := context.TODO()

	server := apiKeyCreateServer(t, 1800000000)
	defer server.Close()

	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	r := &serviceAccountAPIKeyResource{client: c}
	req, resp := newAPIKeyCreateReqResp(t, ctx, tftypes.NewValue(tftypes.String, nil))

	r.Create(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no error when expires_at is omitted, got: %v", resp.Diagnostics.Errors())
	}
}
