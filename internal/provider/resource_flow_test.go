package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestFlowResource_Metadata(t *testing.T) {
	r := &flowResource{}

	req := resource.MetadataRequest{
		ProviderTypeName: "kosli",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_flow"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestFlowResource_Schema(t *testing.T) {
	r := &flowResource{}

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.TODO(), req, resp)

	// Verify schema has description
	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	// Verify expected attributes exist
	attrs := resp.Schema.Attributes
	expectedAttrs := []string{"name", "description", "template", "tags"}
	for _, attr := range expectedAttrs {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}

	// Verify name is required
	nameAttr := attrs["name"]
	if !nameAttr.IsRequired() {
		t.Error("Expected 'name' attribute to be required")
	}

	// A name change is an in-place rename (ADR 005), so name must not carry
	// RequiresReplace or any other plan modifier.
	nameStringAttr, ok := nameAttr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("Expected 'name' to be a schema.StringAttribute, got %T", nameAttr)
	}
	if len(nameStringAttr.PlanModifiers) != 0 {
		t.Errorf("Expected 'name' to have no plan modifiers (rename is in place), got %d", len(nameStringAttr.PlanModifiers))
	}

	// Verify description is optional
	descAttr := attrs["description"]
	if !descAttr.IsOptional() {
		t.Error("Expected 'description' attribute to be optional")
	}

	// Verify template is optional
	templateAttr := attrs["template"]
	if !templateAttr.IsOptional() {
		t.Error("Expected 'template' attribute to be optional")
	}

	// Verify tags is optional and computed
	tagsAttr := attrs["tags"]
	if !tagsAttr.IsOptional() {
		t.Error("Expected 'tags' attribute to be optional")
	}
	if !tagsAttr.IsComputed() {
		t.Error("Expected 'tags' attribute to be computed")
	}
}

func TestFlowResource_Configure(t *testing.T) {
	r := &flowResource{}

	// Test with nil provider data (should not error)
	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.TODO(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}

	if r.client != nil {
		t.Error("Expected client to remain nil when provider data is nil")
	}
}

func TestFlowResource_Configure_WrongType(t *testing.T) {
	r := &flowResource{}

	// Test with wrong type of provider data
	req := resource.ConfigureRequest{
		ProviderData: "wrong type",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.TODO(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}

	if r.client != nil {
		t.Error("Expected client to remain nil when provider data is wrong type")
	}
}

func TestFlowResourceModel_Structure(t *testing.T) {
	tagsMap, diags := types.MapValueFrom(context.TODO(), types.StringType, map[string]string{
		"managed-by": "terraform",
	})
	if diags.HasError() {
		t.Fatal("Failed to create tags map")
	}

	model := flowResourceModel{
		Name:        types.StringValue("my-flow"),
		Description: types.StringValue("CD pipeline for my app"),
		Template:    types.StringValue("version: 1\ntrail:\n  attestations: []\n"),
		Tags:        tagsMap,
	}

	if model.Name.ValueString() != "my-flow" {
		t.Error("Expected Name to be set correctly")
	}

	if model.Description.ValueString() != "CD pipeline for my app" {
		t.Error("Expected Description to be set correctly")
	}

	if model.Template.ValueString() == "" {
		t.Error("Expected Template to be set correctly")
	}

	if model.Tags.IsNull() || model.Tags.IsUnknown() {
		t.Error("Expected Tags to be set")
	}
}

func TestFlowResourceModel_WithNullValues(t *testing.T) {
	model := flowResourceModel{
		Name:        types.StringValue("my-flow"),
		Description: types.StringNull(),
		Template:    types.StringNull(),
		Tags:        types.MapNull(types.StringType),
	}

	if model.Name.ValueString() != "my-flow" {
		t.Error("Expected Name to be set correctly")
	}

	if !model.Description.IsNull() {
		t.Error("Expected Description to be null")
	}

	if !model.Template.IsNull() {
		t.Error("Expected Template to be null")
	}

	if !model.Tags.IsNull() {
		t.Error("Expected Tags to be null")
	}
}

func TestFlowResourceModel_WithTags(t *testing.T) {
	tagsMap, diags := types.MapValueFrom(context.TODO(), types.StringType, map[string]string{
		"env":  "production",
		"team": "platform",
	})
	if diags.HasError() {
		t.Fatal("Failed to create tags map")
	}

	model := flowResourceModel{
		Name: types.StringValue("my-flow"),
		Tags: tagsMap,
	}

	tagElems := model.Tags.Elements()
	if len(tagElems) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tagElems))
	}
}

func TestFlowResourceModel_WithEmptyTags(t *testing.T) {
	emptyMap, diags := types.MapValueFrom(context.TODO(), types.StringType, map[string]string{})
	if diags.HasError() {
		t.Fatal("Failed to create empty tags map")
	}

	model := flowResourceModel{
		Name: types.StringValue("my-flow"),
		Tags: emptyMap,
	}

	if model.Tags.IsNull() {
		t.Error("Expected Tags to be non-null (empty map)")
	}

	if len(model.Tags.Elements()) != 0 {
		t.Error("Expected Tags to have no elements")
	}
}

func TestNewFlowResource(t *testing.T) {
	r := NewFlowResource()

	if r == nil {
		t.Fatal("Expected non-nil resource")
	}

	_, ok := r.(*flowResource)
	if !ok {
		t.Error("Expected resource to be of type *flowResource")
	}
}

func TestFlowResource_Implements(t *testing.T) {
	// Verify the resource implements required interfaces
	var _ resource.Resource = &flowResource{}
	var _ resource.ResourceWithImportState = &flowResource{}
}

// TestMapFlowToModel_Tags exercises the three tag-normalization paths in
// mapFlowToModel: nil (must become empty map), empty map, and populated map.
func TestMapFlowToModel_Tags(t *testing.T) {
	tests := []struct {
		name         string
		apiTags      map[string]string
		wantNull     bool
		wantLen      int
		wantTagKey   string
		wantTagValue string
	}{
		{
			name:     "nil tags normalized to empty map",
			apiTags:  nil,
			wantNull: false,
			wantLen:  0,
		},
		{
			name:     "empty map preserved as empty map",
			apiTags:  map[string]string{},
			wantNull: false,
			wantLen:  0,
		},
		{
			name:         "populated tags mapped correctly",
			apiTags:      map[string]string{"managed-by": "terraform", "team": "platform"},
			wantNull:     false,
			wantLen:      2,
			wantTagKey:   "managed-by",
			wantTagValue: "terraform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flow := &client.Flow{Name: "my-flow", Tags: tt.apiTags}
			var data flowResourceModel
			var diags diag.Diagnostics

			mapFlowToModel(context.Background(), flow, &data, &diags)

			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if data.Tags.IsNull() != tt.wantNull {
				t.Errorf("Tags.IsNull() = %v, want %v", data.Tags.IsNull(), tt.wantNull)
			}
			if got := len(data.Tags.Elements()); got != tt.wantLen {
				t.Errorf("len(Tags.Elements()) = %d, want %d", got, tt.wantLen)
			}
			if tt.wantTagKey != "" {
				elems := data.Tags.Elements()
				v, ok := elems[tt.wantTagKey]
				if !ok {
					t.Errorf("tag key %q not found in %v", tt.wantTagKey, elems)
				} else if v.String() != `"`+tt.wantTagValue+`"` {
					t.Errorf("tag %q = %s, want %q", tt.wantTagKey, v.String(), tt.wantTagValue)
				}
			}
		})
	}
}

// recordedCall is one HTTP request seen by a recording test server.
type recordedCall struct {
	Method string
	Path   string
	Body   string
}

// recordingServer wraps handler so every request's method, path and body are
// appended to calls, in order, before handler runs.
func recordingServer(t *testing.T, calls *[]recordedCall, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		mu.Lock()
		*calls = append(*calls, recordedCall{Method: r.Method, Path: r.URL.Path, Body: string(body)})
		mu.Unlock()
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

// callSignatures returns "METHOD path" for each recorded call.
func callSignatures(calls []recordedCall) []string {
	out := make([]string, 0, len(calls))
	for _, c := range calls {
		out = append(out, c.Method+" "+c.Path)
	}
	return out
}

func assertCallSequence(t *testing.T, calls []recordedCall, want []string) {
	t.Helper()
	got := callSignatures(calls)
	if len(got) != len(want) {
		t.Fatalf("unexpected API call sequence:\n got %v\nwant %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected API call sequence:\n got %v\nwant %v", got, want)
		}
	}
}

// flowRaw builds a raw flow value (template unset, no tags) for plan/state in unit tests.
func flowRaw(t *testing.T, ctx context.Context, s schema.Schema, name, description string) tftypes.Value {
	t.Helper()
	desc := tftypes.NewValue(tftypes.String, nil)
	if description != "" {
		desc = tftypes.NewValue(tftypes.String, description)
	}
	return tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
		"name":        tftypes.NewValue(tftypes.String, name),
		"description": desc,
		"template":    tftypes.NewValue(tftypes.String, nil),
		"tags":        tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{}),
	})
}

// runFlowUpdate drives flowResource.Update from a prior state (priorName,
// priorDesc) to a plan (planName, planDesc) against server. As the framework
// does, the response state is seeded with the prior state.
func runFlowUpdate(t *testing.T, server *httptest.Server, priorName, priorDesc, planName, planDesc string) (*resource.UpdateResponse, flowResourceModel) {
	t.Helper()
	ctx := context.TODO()

	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	r := &flowResource{client: c}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: s, Raw: flowRaw(t, ctx, s, planName, planDesc)},
		State: tfsdk.State{Schema: s, Raw: flowRaw(t, ctx, s, priorName, priorDesc)},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: flowRaw(t, ctx, s, priorName, priorDesc)}}

	r.Update(ctx, req, resp)

	var state flowResourceModel
	if d := resp.State.Get(ctx, &state); d.HasError() {
		t.Fatalf("failed to read response state: %v", d)
	}
	return resp, state
}

// flowUpdateHandler serves a flow rename-and-update sequence. renameStatus and
// putStatus control the rename and PUT responses; GET only answers for
// currentName so a read by a stale name fails loudly.
func flowUpdateHandler(t *testing.T, renameStatus, putStatus int, currentName, description string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/flows/test-org/old-flow/rename":
			w.WriteHeader(renameStatus)
			w.Write([]byte(`"OK"`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/flows/test-org/template_file":
			w.WriteHeader(putStatus)
			w.Write([]byte(`"OK"`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/flows/test-org/"+currentName:
			json.NewEncoder(w).Encode(map[string]any{
				"name": currentName, "description": description, "visibility": "private", "template": "", "tags": map[string]string{},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "not found"}`))
		}
	}
}

// TestFlowResource_Update_RenamesInPlace verifies a name change calls the rename
// endpoint with the prior-state name before the PUT, which then targets the new
// name, and that state ends up with the new name.
func TestFlowResource_Update_RenamesInPlace(t *testing.T) {
	var calls []recordedCall
	server := recordingServer(t, &calls, flowUpdateHandler(t, http.StatusOK, http.StatusOK, "new-flow", "after"))

	resp, state := runFlowUpdate(t, server, "old-flow", "before", "new-flow", "after")

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics.Errors())
	}
	assertCallSequence(t, calls, []string{
		"PUT /api/v2/flows/test-org/old-flow/rename",
		"PUT /api/v2/flows/test-org/template_file",
		"GET /api/v2/flows/test-org/new-flow",
	})

	var renameBody map[string]any
	if err := json.Unmarshal([]byte(calls[0].Body), &renameBody); err != nil {
		t.Fatalf("failed to decode rename body: %v", err)
	}
	if renameBody["new_name"] != "new-flow" {
		t.Errorf("expected rename body new_name=new-flow, got %v", renameBody)
	}

	if state.Name.ValueString() != "new-flow" {
		t.Errorf("expected state name new-flow, got %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != "after" {
		t.Errorf("expected state description after, got %q", state.Description.ValueString())
	}
}

// TestFlowResource_Update_NoRenameWhenNameUnchanged verifies an update that
// keeps the name never touches the rename endpoint.
func TestFlowResource_Update_NoRenameWhenNameUnchanged(t *testing.T) {
	var calls []recordedCall
	server := recordingServer(t, &calls, flowUpdateHandler(t, http.StatusOK, http.StatusOK, "old-flow", "after"))

	resp, state := runFlowUpdate(t, server, "old-flow", "before", "old-flow", "after")

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics.Errors())
	}
	assertCallSequence(t, calls, []string{
		"PUT /api/v2/flows/test-org/template_file",
		"GET /api/v2/flows/test-org/old-flow",
	})
	if state.Name.ValueString() != "old-flow" {
		t.Errorf("expected state name old-flow, got %q", state.Name.ValueString())
	}
}

// TestFlowResource_Update_RenameFails_KeepsPriorState verifies a rejected rename
// (e.g. the new name is taken) stops the update before any other call and
// leaves the prior state, including the old name, untouched.
func TestFlowResource_Update_RenameFails_KeepsPriorState(t *testing.T) {
	var calls []recordedCall
	server := recordingServer(t, &calls, flowUpdateHandler(t, http.StatusBadRequest, http.StatusOK, "old-flow", "before"))

	resp, state := runFlowUpdate(t, server, "old-flow", "before", "new-flow", "after")

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Error Renaming Flow" {
		t.Errorf("expected summary 'Error Renaming Flow', got %q", got)
	}
	assertCallSequence(t, calls, []string{"PUT /api/v2/flows/test-org/old-flow/rename"})
	if state.Name.ValueString() != "old-flow" {
		t.Errorf("expected state to keep old-flow, got %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != "before" {
		t.Errorf("expected state to keep prior description, got %q", state.Description.ValueString())
	}
}

// TestFlowResource_Update_UpdateFailsAfterRename_KeepsNewName verifies that
// when the rename succeeds but the following update fails, state records the
// new name (the rename happened) while keeping the prior values otherwise.
func TestFlowResource_Update_UpdateFailsAfterRename_KeepsNewName(t *testing.T) {
	var calls []recordedCall
	server := recordingServer(t, &calls, flowUpdateHandler(t, http.StatusOK, http.StatusBadRequest, "new-flow", "before"))

	resp, state := runFlowUpdate(t, server, "old-flow", "before", "new-flow", "after")

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Error Updating Flow" {
		t.Errorf("expected summary 'Error Updating Flow', got %q", got)
	}
	assertCallSequence(t, calls, []string{
		"PUT /api/v2/flows/test-org/old-flow/rename",
		"PUT /api/v2/flows/test-org/template_file",
	})
	if state.Name.ValueString() != "new-flow" {
		t.Errorf("expected state to record the applied rename new-flow, got %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != "before" {
		t.Errorf("expected state to keep prior description, got %q", state.Description.ValueString())
	}
}
