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

func TestLogicalEnvironmentResource_Metadata(t *testing.T) {
	r := &logicalEnvironmentResource{}

	// Test metadata
	req := resource.MetadataRequest{
		ProviderTypeName: "kosli",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.TODO(), req, resp)

	expectedTypeName := "kosli_logical_environment"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %q, got %q", expectedTypeName, resp.TypeName)
	}
}

func TestLogicalEnvironmentResource_Schema(t *testing.T) {
	r := &logicalEnvironmentResource{}

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.TODO(), req, resp)

	// Verify schema has description
	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := resp.Schema.Attributes
	requiredAttrs := []string{"name", "type", "description", "included_environments", "tags"}
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

	// Verify type is computed (not configurable by user)
	typeAttr := attrs["type"]
	if typeAttr.IsComputed() == false {
		t.Error("Expected 'type' attribute to be computed")
	}
	if typeAttr.IsRequired() == true {
		t.Error("Expected 'type' attribute to not be required (it's computed)")
	}

	// Verify description is optional
	descAttr := attrs["description"]
	if descAttr.IsOptional() == false {
		t.Error("Expected 'description' attribute to be optional")
	}

	// Verify included_environments is required
	includedEnvsAttr := attrs["included_environments"]
	if includedEnvsAttr.IsRequired() == false {
		t.Error("Expected 'included_environments' attribute to be required")
	}

	// Verify tags is optional and computed
	tagsAttr := attrs["tags"]
	if tagsAttr.IsOptional() == false {
		t.Error("Expected 'tags' attribute to be optional")
	}
	if tagsAttr.IsComputed() == false {
		t.Error("Expected 'tags' attribute to be computed")
	}
}

func TestLogicalEnvironmentResource_Configure(t *testing.T) {
	r := &logicalEnvironmentResource{}

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

func TestLogicalEnvironmentResource_Configure_WrongType(t *testing.T) {
	r := &logicalEnvironmentResource{}

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

func TestLogicalEnvironmentResourceModel_Structure(t *testing.T) {
	// Test that the model can be created with expected fields
	includedEnvs, diags := types.ListValueFrom(context.TODO(), types.StringType, []string{"prod-k8s", "prod-ecs"})
	if diags.HasError() {
		t.Fatalf("Unexpected diagnostics creating list: %v", diags)
	}

	model := logicalEnvironmentResourceModel{
		Name:                 types.StringValue("production-aggregate"),
		Type:                 types.StringValue("logical"),
		Description:          types.StringValue("All production environments"),
		IncludedEnvironments: includedEnvs,
	}

	if model.Name.ValueString() != "production-aggregate" {
		t.Error("Expected Name to be set correctly")
	}

	if model.Type.ValueString() != "logical" {
		t.Error("Expected Type to be set correctly")
	}

	if model.Description.ValueString() != "All production environments" {
		t.Error("Expected Description to be set correctly")
	}

	if model.IncludedEnvironments.IsNull() {
		t.Error("Expected IncludedEnvironments to be set")
	}

	// Verify list contents
	var envs []string
	diags = model.IncludedEnvironments.ElementsAs(context.TODO(), &envs, false)
	if diags.HasError() {
		t.Fatalf("Unexpected diagnostics converting IncludedEnvironments: %v", diags)
	}
	if len(envs) != 2 {
		t.Errorf("Expected 2 included environments, got %d", len(envs))
	}
	if envs[0] != "prod-k8s" || envs[1] != "prod-ecs" {
		t.Errorf("Expected ['prod-k8s', 'prod-ecs'], got %v", envs)
	}
}

func TestLogicalEnvironmentResourceModel_WithNullValues(t *testing.T) {
	// Test that the model handles null values correctly
	includedEnvs, _ := types.ListValueFrom(context.TODO(), types.StringType, []string{"test-env"})

	model := logicalEnvironmentResourceModel{
		Name:                 types.StringValue("test-logical"),
		Type:                 types.StringValue("logical"),
		Description:          types.StringNull(),
		IncludedEnvironments: includedEnvs,
	}

	if model.Name.ValueString() != "test-logical" {
		t.Error("Expected Name to be set correctly")
	}

	if !model.Description.IsNull() {
		t.Error("Expected Description to be null")
	}

	if model.IncludedEnvironments.IsNull() {
		t.Error("Expected IncludedEnvironments to not be null")
	}
}

func TestLogicalEnvironmentResourceModel_EmptyList(t *testing.T) {
	// Test handling of empty included_environments list
	emptyList, _ := types.ListValueFrom(context.TODO(), types.StringType, []string{})

	model := logicalEnvironmentResourceModel{
		Name:                 types.StringValue("test-env"),
		Type:                 types.StringValue("logical"),
		Description:          types.StringValue("Test"),
		IncludedEnvironments: emptyList,
	}

	if model.IncludedEnvironments.IsNull() {
		t.Error("Expected IncludedEnvironments to not be null even when empty")
	}

	var envs []string
	model.IncludedEnvironments.ElementsAs(context.TODO(), &envs, false)
	if len(envs) != 0 {
		t.Errorf("Expected 0 included environments, got %d", len(envs))
	}
}

func TestNewLogicalEnvironmentResource(t *testing.T) {
	r := NewLogicalEnvironmentResource()

	if r == nil {
		t.Fatal("Expected non-nil resource")
	}

	_, ok := r.(*logicalEnvironmentResource)
	if !ok {
		t.Error("Expected resource to be of type *logicalEnvironmentResource")
	}
}

func TestLogicalEnvironmentResource_Implements(t *testing.T) {
	// Verify the resource implements required interfaces
	var _ resource.Resource = &logicalEnvironmentResource{}
	var _ resource.ResourceWithImportState = &logicalEnvironmentResource{}
}

func TestLogicalEnvironmentResourceModel_WithTags(t *testing.T) {
	includedEnvs, _ := types.ListValueFrom(context.TODO(), types.StringType, []string{"prod-k8s"})
	tagsMap, diags := types.MapValueFrom(context.TODO(), types.StringType, map[string]string{
		"managed-by":  "terraform",
		"environment": "production",
	})
	if diags.HasError() {
		t.Fatalf("Unexpected diagnostics creating tags map: %v", diags)
	}

	model := logicalEnvironmentResourceModel{
		Name:                 types.StringValue("tagged-logical"),
		Type:                 types.StringValue("logical"),
		Description:          types.StringValue("Tagged logical environment"),
		IncludedEnvironments: includedEnvs,
		Tags:                 tagsMap,
	}

	if model.Tags.IsNull() {
		t.Error("Expected Tags to be set")
	}
	if model.Tags.IsUnknown() {
		t.Error("Expected Tags to be known")
	}

	var tags map[string]string
	diags = model.Tags.ElementsAs(context.TODO(), &tags, false)
	if diags.HasError() {
		t.Fatalf("Unexpected diagnostics converting Tags: %v", diags)
	}
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}
	if tags["managed-by"] != "terraform" {
		t.Errorf("Expected managed-by=terraform, got %q", tags["managed-by"])
	}
	if tags["environment"] != "production" {
		t.Errorf("Expected environment=production, got %q", tags["environment"])
	}
}

func TestLogicalEnvironmentResourceModel_EmptyTags(t *testing.T) {
	includedEnvs, _ := types.ListValueFrom(context.TODO(), types.StringType, []string{})
	emptyTags, diags := types.MapValueFrom(context.TODO(), types.StringType, map[string]string{})
	if diags.HasError() {
		t.Fatalf("Unexpected diagnostics creating empty tags map: %v", diags)
	}

	model := logicalEnvironmentResourceModel{
		Name:                 types.StringValue("no-tags-logical"),
		Type:                 types.StringValue("logical"),
		IncludedEnvironments: includedEnvs,
		Tags:                 emptyTags,
	}

	if model.Tags.IsNull() {
		t.Error("Expected Tags to be non-null (empty map, not null)")
	}

	var tags map[string]string
	model.Tags.ElementsAs(context.TODO(), &tags, false)
	if len(tags) != 0 {
		t.Errorf("Expected 0 tags, got %d", len(tags))
	}
}

// Note: Full CRUD operation tests require acceptance testing
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
// - Validation of logical environment constraints is tested

// logicalEnvironmentRaw builds a raw logical environment value (no tags, two
// included environments) for plan/state in unit tests.
func logicalEnvironmentRaw(t *testing.T, ctx context.Context, s schema.Schema, name, description string) tftypes.Value {
	t.Helper()
	desc := tftypes.NewValue(tftypes.String, nil)
	if description != "" {
		desc = tftypes.NewValue(tftypes.String, description)
	}
	listType := tftypes.List{ElementType: tftypes.String}
	return tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
		"name":        tftypes.NewValue(tftypes.String, name),
		"type":        tftypes.NewValue(tftypes.String, "logical"),
		"description": desc,
		"included_environments": tftypes.NewValue(listType, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "env-a"),
			tftypes.NewValue(tftypes.String, "env-b"),
		}),
		"tags": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{}),
	})
}

// runLogicalEnvironmentUpdate drives logicalEnvironmentResource.Update from a
// prior state (priorName, priorDesc) to a plan (planName, planDesc) against
// server. As the framework does, the response state is seeded with the prior state.
func runLogicalEnvironmentUpdate(t *testing.T, server *httptest.Server, priorName, priorDesc, planName, planDesc string) (*resource.UpdateResponse, logicalEnvironmentResourceModel) {
	t.Helper()
	ctx := context.TODO()

	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	r := &logicalEnvironmentResource{client: c}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: s, Raw: logicalEnvironmentRaw(t, ctx, s, planName, planDesc)},
		State: tfsdk.State{Schema: s, Raw: logicalEnvironmentRaw(t, ctx, s, priorName, priorDesc)},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: logicalEnvironmentRaw(t, ctx, s, priorName, priorDesc)}}

	r.Update(ctx, req, resp)

	var state logicalEnvironmentResourceModel
	if d := resp.State.Get(ctx, &state); d.HasError() {
		t.Fatalf("failed to read response state: %v", d)
	}
	return resp, state
}

// logicalEnvironmentUpdateHandler serves a logical environment
// rename-and-update sequence. renameStatus and patchStatus control the rename
// and PATCH responses; GET and PATCH only answer for currentName so a call by
// a stale name fails loudly.
func logicalEnvironmentUpdateHandler(t *testing.T, renameStatus, patchStatus int, currentName, description string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/environments/test-org/old-logical/rename":
			w.WriteHeader(renameStatus)
			w.Write([]byte(`"OK"`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/environments/test-org/"+currentName:
			w.WriteHeader(patchStatus)
			w.Write([]byte(`"OK"`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/environments/test-org/"+currentName:
			json.NewEncoder(w).Encode(map[string]any{
				"org": "test-org", "name": currentName, "type": "logical", "description": description,
				"last_modified_at": 1.0, "last_reported_at": nil, "state": nil, "tags": map[string]string{},
				"policies": []any{}, "included_environments": []string{"env-a", "env-b"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "not found"}`))
		}
	}
}

// TestLogicalEnvironmentResource_Update_RenamesInPlace verifies a name change
// calls the rename endpoint with the prior-state name before the PATCH, which
// then targets the new name, and that state ends up with the new name.
func TestLogicalEnvironmentResource_Update_RenamesInPlace(t *testing.T) {
	var calls []recordedCall
	server := recordingServer(t, &calls, logicalEnvironmentUpdateHandler(t, http.StatusOK, http.StatusOK, "new-logical", "after"))

	resp, state := runLogicalEnvironmentUpdate(t, server, "old-logical", "before", "new-logical", "after")

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics.Errors())
	}
	assertCallSequence(t, calls, []string{
		"PUT /api/v2/environments/test-org/old-logical/rename",
		"PATCH /api/v2/environments/test-org/new-logical",
		"GET /api/v2/environments/test-org/new-logical",
	})

	var renameBody map[string]any
	if err := json.Unmarshal([]byte(calls[0].Body), &renameBody); err != nil {
		t.Fatalf("failed to decode rename body: %v", err)
	}
	if renameBody["new_name"] != "new-logical" {
		t.Errorf("expected rename body new_name=new-logical, got %v", renameBody)
	}

	if state.Name.ValueString() != "new-logical" {
		t.Errorf("expected state name new-logical, got %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != "after" {
		t.Errorf("expected state description after, got %q", state.Description.ValueString())
	}
}

// TestLogicalEnvironmentResource_Update_NoRenameWhenNameUnchanged verifies an
// update that keeps the name never touches the rename endpoint.
func TestLogicalEnvironmentResource_Update_NoRenameWhenNameUnchanged(t *testing.T) {
	var calls []recordedCall
	server := recordingServer(t, &calls, logicalEnvironmentUpdateHandler(t, http.StatusOK, http.StatusOK, "old-logical", "after"))

	resp, state := runLogicalEnvironmentUpdate(t, server, "old-logical", "before", "old-logical", "after")

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics.Errors())
	}
	assertCallSequence(t, calls, []string{
		"PATCH /api/v2/environments/test-org/old-logical",
		"GET /api/v2/environments/test-org/old-logical",
	})
	if state.Name.ValueString() != "old-logical" {
		t.Errorf("expected state name old-logical, got %q", state.Name.ValueString())
	}
}

// TestLogicalEnvironmentResource_Update_RenameFails_KeepsPriorState verifies a
// rejected rename stops the update before any other call and leaves the prior
// state, including the old name, untouched.
func TestLogicalEnvironmentResource_Update_RenameFails_KeepsPriorState(t *testing.T) {
	var calls []recordedCall
	server := recordingServer(t, &calls, logicalEnvironmentUpdateHandler(t, http.StatusBadRequest, http.StatusOK, "old-logical", "before"))

	resp, state := runLogicalEnvironmentUpdate(t, server, "old-logical", "before", "new-logical", "after")

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Error Renaming Logical Environment" {
		t.Errorf("expected summary 'Error Renaming Logical Environment', got %q", got)
	}
	assertCallSequence(t, calls, []string{"PUT /api/v2/environments/test-org/old-logical/rename"})
	if state.Name.ValueString() != "old-logical" {
		t.Errorf("expected state to keep old-logical, got %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != "before" {
		t.Errorf("expected state to keep prior description, got %q", state.Description.ValueString())
	}
}

// TestLogicalEnvironmentResource_Update_UpdateFailsAfterRename_KeepsNewName
// verifies that when the rename succeeds but the following PATCH fails, state
// records the new name (the rename happened) while keeping the prior values
// otherwise.
func TestLogicalEnvironmentResource_Update_UpdateFailsAfterRename_KeepsNewName(t *testing.T) {
	var calls []recordedCall
	server := recordingServer(t, &calls, logicalEnvironmentUpdateHandler(t, http.StatusOK, http.StatusBadRequest, "new-logical", "before"))

	resp, state := runLogicalEnvironmentUpdate(t, server, "old-logical", "before", "new-logical", "after")

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Error Updating Logical Environment" {
		t.Errorf("expected summary 'Error Updating Logical Environment', got %q", got)
	}
	assertCallSequence(t, calls, []string{
		"PUT /api/v2/environments/test-org/old-logical/rename",
		"PATCH /api/v2/environments/test-org/new-logical",
	})
	if state.Name.ValueString() != "new-logical" {
		t.Errorf("expected state to record the applied rename new-logical, got %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != "before" {
		t.Errorf("expected state to keep prior description, got %q", state.Description.ValueString())
	}
}
