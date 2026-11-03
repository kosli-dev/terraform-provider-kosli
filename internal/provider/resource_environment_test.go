package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestEnvironmentResource_Metadata(t *testing.T) {
	r := &environmentResource{}

	// Test metadata
	req := resource.MetadataRequest{
		ProviderTypeName: "kosli",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_environment"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestEnvironmentResource_Schema(t *testing.T) {
	r := &environmentResource{}

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.TODO(), req, resp)

	// Verify schema has description
	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := resp.Schema.Attributes
	requiredAttrs := []string{"name", "type", "description", "tags"}
	for _, attr := range requiredAttrs {
		if _, exists := attrs[attr]; !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
		}
	}

	// Verify name is required
	nameAttr := attrs["name"]
	if nameAttr.IsRequired() == false {
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

	// Verify type is required
	typeAttr := attrs["type"]
	if typeAttr.IsRequired() == false {
		t.Error("Expected 'type' attribute to be required")
	}

	// Type cannot be changed by the API, so it still forces replacement.
	typeStringAttr, ok := typeAttr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("Expected 'type' to be a schema.StringAttribute, got %T", typeAttr)
	}
	if len(typeStringAttr.PlanModifiers) == 0 {
		t.Error("Expected 'type' to keep its RequiresReplace plan modifier")
	}

	// Verify description is optional
	descAttr := attrs["description"]
	if descAttr.IsOptional() == false {
		t.Error("Expected 'description' attribute to be optional")
	}

	// include_scaling was removed from the API and must not be in the schema (issue #235)
	if _, exists := attrs["include_scaling"]; exists {
		t.Error("Expected 'include_scaling' attribute to be removed from schema")
	}

	// Verify tags is optional
	tagsAttr := attrs["tags"]
	if tagsAttr.IsOptional() == false {
		t.Error("Expected 'tags' attribute to be optional")
	}
}

func TestEnvironmentResource_Configure(t *testing.T) {
	r := &environmentResource{}

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

func TestEnvironmentResource_Configure_WrongType(t *testing.T) {
	r := &environmentResource{}

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

func TestEnvironmentResourceModel_Structure(t *testing.T) {
	// Test that the model can be created with expected fields
	model := environmentResourceModel{
		Name:        types.StringValue("production-k8s"),
		Type:        types.StringValue("K8S"),
		Description: types.StringValue("Production cluster"),
		Tags:        types.MapNull(types.StringType),
	}

	if model.Name.ValueString() != "production-k8s" {
		t.Error("Expected Name to be set correctly")
	}

	if model.Type.ValueString() != "K8S" {
		t.Error("Expected Type to be set correctly")
	}

	if model.Description.ValueString() != "Production cluster" {
		t.Error("Expected Description to be set correctly")
	}

	if !model.Tags.IsNull() {
		t.Error("Expected Tags to be null")
	}
}

func TestEnvironmentResourceModel_WithTags(t *testing.T) {
	tagValues := map[string]string{
		"env":  "production",
		"team": "platform",
	}
	tagsMap, diags := types.MapValueFrom(context.TODO(), types.StringType, tagValues)
	if diags.HasError() {
		t.Fatalf("Failed to create tags map: %v", diags)
	}

	model := environmentResourceModel{
		Name:        types.StringValue("production-k8s"),
		Type:        types.StringValue("K8S"),
		Description: types.StringNull(),
		Tags:        tagsMap,
	}

	if model.Tags.IsNull() {
		t.Error("Expected Tags to not be null")
	}

	var resultTags map[string]string
	diags = model.Tags.ElementsAs(context.TODO(), &resultTags, false)
	if diags.HasError() {
		t.Fatalf("Failed to extract tags: %v", diags)
	}

	if resultTags["env"] != "production" {
		t.Errorf("Expected tags.env = 'production', got %q", resultTags["env"])
	}
	if resultTags["team"] != "platform" {
		t.Errorf("Expected tags.team = 'platform', got %q", resultTags["team"])
	}
}

func TestEnvironmentResourceModel_WithNullValues(t *testing.T) {
	// Test that the model handles null values correctly
	model := environmentResourceModel{
		Name:        types.StringValue("test-env"),
		Type:        types.StringValue("docker"),
		Description: types.StringNull(),
		Tags:        types.MapNull(types.StringType),
	}

	if model.Name.ValueString() != "test-env" {
		t.Error("Expected Name to be set correctly")
	}

	if !model.Description.IsNull() {
		t.Error("Expected Description to be null")
	}

	if !model.Tags.IsNull() {
		t.Error("Expected Tags to be null")
	}
}

func TestNewEnvironmentResource(t *testing.T) {
	r := NewEnvironmentResource()

	if r == nil {
		t.Fatal("Expected non-nil resource")
	}

	_, ok := r.(*environmentResource)
	if !ok {
		t.Error("Expected resource to be of type *environmentResource")
	}
}

func TestEnvironmentResource_Implements(t *testing.T) {
	// Verify the resource implements required interfaces
	var _ resource.Resource = &environmentResource{}
	var _ resource.ResourceWithImportState = &environmentResource{}
}

// Note: Full CRUD operation tests require acceptance testing (issue #71)
// These tests verify the resource structure and basic configuration,
// while acceptance tests will verify the full lifecycle against a real API.
//
// The CRUD methods (Create, Read, Update, Delete) have 0% coverage in unit tests
// because they require:
// - A real or mock Kosli API client
// - Terraform framework context (plans, state)
// - Complex setup with request/response mocking
//
// These will be thoroughly tested in acceptance tests where:
// - Real resources are created/updated/deleted in a test Kosli organization
// - The full Terraform lifecycle is exercised
// - API integration is validated end-to-end

// environmentRaw builds a raw environment value (no tags) for plan/state in unit tests.
func environmentRaw(t *testing.T, ctx context.Context, s schema.Schema, name, description string) tftypes.Value {
	t.Helper()
	desc := tftypes.NewValue(tftypes.String, nil)
	if description != "" {
		desc = tftypes.NewValue(tftypes.String, description)
	}
	return tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
		"name":        tftypes.NewValue(tftypes.String, name),
		"type":        tftypes.NewValue(tftypes.String, "K8S"),
		"description": desc,
		"tags":        tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{}),
	})
}

// runEnvironmentUpdate drives environmentResource.Update from a prior state
// (priorName, priorDesc) to a plan (planName, planDesc) against server. As the
// framework does, the response state is seeded with the prior state.
func runEnvironmentUpdate(t *testing.T, server *httptest.Server, priorName, priorDesc, planName, planDesc string) (*resource.UpdateResponse, environmentResourceModel) {
	t.Helper()
	ctx := context.TODO()

	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	r := &environmentResource{client: c}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: s, Raw: environmentRaw(t, ctx, s, planName, planDesc)},
		State: tfsdk.State{Schema: s, Raw: environmentRaw(t, ctx, s, priorName, priorDesc)},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: environmentRaw(t, ctx, s, priorName, priorDesc)}}

	r.Update(ctx, req, resp)

	var state environmentResourceModel
	if d := resp.State.Get(ctx, &state); d.HasError() {
		t.Fatalf("failed to read response state: %v", d)
	}
	return resp, state
}

// environmentUpdateHandler serves an environment rename-and-update sequence.
// renameStatus and patchStatus control the rename and PATCH responses; GET and
// PATCH only answer for currentName so a call by a stale name fails loudly.
func environmentUpdateHandler(t *testing.T, renameStatus, patchStatus int, currentName, description string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/environments/test-org/old-env/rename":
			w.WriteHeader(renameStatus)
			w.Write([]byte(`"OK"`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/environments/test-org/"+currentName:
			w.WriteHeader(patchStatus)
			w.Write([]byte(`"OK"`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/environments/test-org/"+currentName:
			json.NewEncoder(w).Encode(map[string]any{
				"org": "test-org", "name": currentName, "type": "K8S", "description": description,
				"last_modified_at": 1.0, "last_reported_at": nil, "state": nil, "tags": map[string]string{}, "policies": []any{},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "not found"}`))
		}
	}
}

// TestEnvironmentResource_Update_RenamesInPlace verifies a name change calls
// the rename endpoint with the prior-state name before the PATCH, which then
// targets the new name, and that state ends up with the new name.
func TestEnvironmentResource_Update_RenamesInPlace(t *testing.T) {
	var calls []recordedCall
	server := recordingServer(t, &calls, environmentUpdateHandler(t, http.StatusOK, http.StatusOK, "new-env", "after"))

	resp, state := runEnvironmentUpdate(t, server, "old-env", "before", "new-env", "after")

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics.Errors())
	}
	assertCallSequence(t, calls, []string{
		"PUT /api/v2/environments/test-org/old-env/rename",
		"PATCH /api/v2/environments/test-org/new-env",
		"GET /api/v2/environments/test-org/new-env",
	})

	var renameBody map[string]any
	if err := json.Unmarshal([]byte(calls[0].Body), &renameBody); err != nil {
		t.Fatalf("failed to decode rename body: %v", err)
	}
	if renameBody["new_name"] != "new-env" {
		t.Errorf("expected rename body new_name=new-env, got %v", renameBody)
	}

	if state.Name.ValueString() != "new-env" {
		t.Errorf("expected state name new-env, got %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != "after" {
		t.Errorf("expected state description after, got %q", state.Description.ValueString())
	}
}

// TestEnvironmentResource_Update_NoRenameWhenNameUnchanged verifies an update
// that keeps the name never touches the rename endpoint.
func TestEnvironmentResource_Update_NoRenameWhenNameUnchanged(t *testing.T) {
	var calls []recordedCall
	server := recordingServer(t, &calls, environmentUpdateHandler(t, http.StatusOK, http.StatusOK, "old-env", "after"))

	resp, state := runEnvironmentUpdate(t, server, "old-env", "before", "old-env", "after")

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics.Errors())
	}
	assertCallSequence(t, calls, []string{
		"PATCH /api/v2/environments/test-org/old-env",
		"GET /api/v2/environments/test-org/old-env",
	})
	if state.Name.ValueString() != "old-env" {
		t.Errorf("expected state name old-env, got %q", state.Name.ValueString())
	}
}

// TestEnvironmentResource_Update_RenameFails_KeepsPriorState verifies a
// rejected rename stops the update before any other call and leaves the prior
// state, including the old name, untouched.
func TestEnvironmentResource_Update_RenameFails_KeepsPriorState(t *testing.T) {
	var calls []recordedCall
	server := recordingServer(t, &calls, environmentUpdateHandler(t, http.StatusBadRequest, http.StatusOK, "old-env", "before"))

	resp, state := runEnvironmentUpdate(t, server, "old-env", "before", "new-env", "after")

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Error Renaming Environment" {
		t.Errorf("expected summary 'Error Renaming Environment', got %q", got)
	}
	assertCallSequence(t, calls, []string{"PUT /api/v2/environments/test-org/old-env/rename"})
	if state.Name.ValueString() != "old-env" {
		t.Errorf("expected state to keep old-env, got %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != "before" {
		t.Errorf("expected state to keep prior description, got %q", state.Description.ValueString())
	}
}

// TestEnvironmentResource_Update_UpdateFailsAfterRename_KeepsNewName verifies
// that when the rename succeeds but the following PATCH fails, state records
// the new name (the rename happened) while keeping the prior values otherwise.
func TestEnvironmentResource_Update_UpdateFailsAfterRename_KeepsNewName(t *testing.T) {
	var calls []recordedCall
	server := recordingServer(t, &calls, environmentUpdateHandler(t, http.StatusOK, http.StatusBadRequest, "new-env", "before"))

	resp, state := runEnvironmentUpdate(t, server, "old-env", "before", "new-env", "after")

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Error Updating Environment" {
		t.Errorf("expected summary 'Error Updating Environment', got %q", got)
	}
	assertCallSequence(t, calls, []string{
		"PUT /api/v2/environments/test-org/old-env/rename",
		"PATCH /api/v2/environments/test-org/new-env",
	})
	if state.Name.ValueString() != "new-env" {
		t.Errorf("expected state to record the applied rename new-env, got %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != "before" {
		t.Errorf("expected state to keep prior description, got %q", state.Description.ValueString())
	}
}
