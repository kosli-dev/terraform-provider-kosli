package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

const testRepoID = "repo-123"

func TestRepoTagsResource_Metadata(t *testing.T) {
	r := &repoTagsResource{}

	resp := &resource.MetadataResponse{}
	r.Metadata(context.TODO(), resource.MetadataRequest{ProviderTypeName: "kosli"}, resp)

	if resp.TypeName != "kosli_repo_tags" {
		t.Errorf("Expected TypeName %q, got %q", "kosli_repo_tags", resp.TypeName)
	}
}

func TestRepoTagsResource_Schema(t *testing.T) {
	r := &repoTagsResource{}

	resp := &resource.SchemaResponse{}
	r.Schema(context.TODO(), resource.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	repoID, ok := resp.Schema.Attributes["repo_id"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("Expected 'repo_id' to be a string attribute, got %T", resp.Schema.Attributes["repo_id"])
	}
	if !repoID.IsRequired() {
		t.Error("Expected 'repo_id' to be required")
	}
	if len(repoID.PlanModifiers) == 0 {
		t.Error("Expected 'repo_id' to force replacement")
	}

	tags, ok := resp.Schema.Attributes["tags"]
	if !ok {
		t.Fatal("Expected 'tags' attribute to exist")
	}
	if !tags.IsRequired() {
		t.Error("Expected 'tags' to be required")
	}
}

func TestRepoTagsResource_Configure(t *testing.T) {
	r := &repoTagsResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.TODO(), resource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}

	resp = &resource.ConfigureResponse{}
	r.Configure(context.TODO(), resource.ConfigureRequest{ProviderData: "wrong type"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
}

func TestNewRepoTagsResource(t *testing.T) {
	r := NewRepoTagsResource()
	if _, ok := r.(*repoTagsResource); !ok {
		t.Error("Expected resource to be of type *repoTagsResource")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("Expected resource to support import")
	}
}

// fakeRepoAPI serves GET repo and PATCH repo tags for one repo. A nil tags map
// means the repo does not exist.
type fakeRepoAPI struct {
	mu    sync.Mutex
	tags  map[string]string
	calls []recordedCall
}

func (f *fakeRepoAPI) currentTags() map[string]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tags
}

func (f *fakeRepoAPI) server(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		var body struct {
			SetTags    map[string]string `json:"set_tags"`
			RemoveTags []string          `json:"remove_tags"`
		}
		raw := ""
		if r.Method == http.MethodPatch {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode PATCH body: %v", err)
			}
			b, _ := json.Marshal(body)
			raw = string(b)
		}
		f.calls = append(f.calls, recordedCall{Method: r.Method, Path: r.URL.Path, Body: raw})

		w.Header().Set("Content-Type", "application/json")
		if f.tags == nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Repo not found"}`))
			return
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/repos/test-org/"+testRepoID:
			if got := r.URL.Query().Get("id"); got != testRepoID {
				t.Errorf("expected id=%s query, got %q", testRepoID, got)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"id": testRepoID, "name": "acme/api", "url": "https://github.com/acme/api", "provider": "github", "tags": f.tags,
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/tags/test-org/repo/"+testRepoID:
			for _, k := range body.RemoveTags {
				delete(f.tags, k)
			}
			for k, v := range body.SetTags {
				f.tags[k] = v
			}
			w.Write([]byte(`"OK"`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func newRepoTagsTestResource(t *testing.T, server *httptest.Server) (*repoTagsResource, schema.Schema) {
	t.Helper()
	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
	r := &repoTagsResource{client: c}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.TODO(), resource.SchemaRequest{}, schemaResp)
	return r, schemaResp.Schema
}

// repoTagsRaw builds a raw kosli_repo_tags value. A nil tags map is null.
func repoTagsRaw(ctx context.Context, s schema.Schema, repoID string, tags map[string]string) tftypes.Value {
	mapType := tftypes.Map{ElementType: tftypes.String}
	tagsValue := tftypes.NewValue(mapType, nil)
	if tags != nil {
		elems := map[string]tftypes.Value{}
		for k, v := range tags {
			elems[k] = tftypes.NewValue(tftypes.String, v)
		}
		tagsValue = tftypes.NewValue(mapType, elems)
	}
	repoIDValue := tftypes.NewValue(tftypes.String, nil)
	if repoID != "" {
		repoIDValue = tftypes.NewValue(tftypes.String, repoID)
	}
	return tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
		"repo_id": repoIDValue,
		"tags":    tagsValue,
	})
}

func stateTags(t *testing.T, state tfsdk.State) map[string]string {
	t.Helper()
	var data repoTagsResourceModel
	if d := state.Get(context.TODO(), &data); d.HasError() {
		t.Fatalf("failed to read state: %v", d)
	}
	tags := map[string]string{}
	if d := data.Tags.ElementsAs(context.TODO(), &tags, false); d.HasError() {
		t.Fatalf("failed to read state tags: %v", d)
	}
	return tags
}

func patchBody(t *testing.T, call recordedCall) client.TagResourcePayload {
	t.Helper()
	var body client.TagResourcePayload
	if err := json.Unmarshal([]byte(call.Body), &body); err != nil {
		t.Fatalf("failed to decode PATCH body %q: %v", call.Body, err)
	}
	sort.Strings(body.RemoveTags)
	return body
}

const (
	getRepoCall   = "GET /api/v2/repos/test-org/" + testRepoID
	patchTagsCall = "PATCH /api/v2/tags/test-org/repo/" + testRepoID
)

func TestRepoTagsResource_Create_ReplacesTagsSetOutsideTerraform(t *testing.T) {
	api := &fakeRepoAPI{tags: map[string]string{"team": "old", "manual": "x"}}
	r, s := newRepoTagsTestResource(t, api.server(t))
	ctx := context.TODO()

	want := map[string]string{"team": "platform", "tier": "1"}
	plan := repoTagsRaw(ctx, s, testRepoID, want)
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: repoTagsRaw(ctx, s, "", nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: plan}}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	assertCallSequence(t, api.calls, []string{getRepoCall, patchTagsCall, getRepoCall})

	body := patchBody(t, api.calls[1])
	if !reflect.DeepEqual(body.SetTags, map[string]string{"team": "platform", "tier": "1"}) {
		t.Errorf("unexpected set_tags: %v", body.SetTags)
	}
	if !reflect.DeepEqual(body.RemoveTags, []string{"manual"}) {
		t.Errorf("unexpected remove_tags: %v", body.RemoveTags)
	}
	if got := stateTags(t, resp.State); !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected state tags: %v", got)
	}
}

func TestRepoTagsResource_Create_SkipsPatchWhenTagsMatch(t *testing.T) {
	api := &fakeRepoAPI{tags: map[string]string{"team": "platform"}}
	r, s := newRepoTagsTestResource(t, api.server(t))
	ctx := context.TODO()

	plan := repoTagsRaw(ctx, s, testRepoID, map[string]string{"team": "platform"})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: repoTagsRaw(ctx, s, "", nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: plan}}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	assertCallSequence(t, api.calls, []string{getRepoCall, getRepoCall})
}

func TestRepoTagsResource_Create_RepoNotFound(t *testing.T) {
	api := &fakeRepoAPI{}
	r, s := newRepoTagsTestResource(t, api.server(t))
	ctx := context.TODO()

	plan := repoTagsRaw(ctx, s, testRepoID, map[string]string{"team": "platform"})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: repoTagsRaw(ctx, s, "", nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: plan}}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Repo Not Found" {
		t.Errorf("unexpected summary %q", got)
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "kosli_repos") {
		t.Errorf("expected detail to point at the kosli_repos data source, got %q", resp.Diagnostics.Errors()[0].Detail())
	}
	assertCallSequence(t, api.calls, []string{getRepoCall})
}

func TestRepoTagsResource_Read_StoresAllRepoTags(t *testing.T) {
	api := &fakeRepoAPI{tags: map[string]string{"team": "platform", "manual": "x"}}
	r, s := newRepoTagsTestResource(t, api.server(t))
	ctx := context.TODO()

	prior := repoTagsRaw(ctx, s, testRepoID, map[string]string{"team": "platform"})
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: s, Raw: prior}}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: prior}}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if got := stateTags(t, resp.State); !reflect.DeepEqual(got, map[string]string{"team": "platform", "manual": "x"}) {
		t.Errorf("expected every repo tag in state, got %v", got)
	}
}

func TestRepoTagsResource_Read_RepoGoneRemovesResource(t *testing.T) {
	api := &fakeRepoAPI{}
	r, s := newRepoTagsTestResource(t, api.server(t))
	ctx := context.TODO()

	prior := repoTagsRaw(ctx, s, testRepoID, map[string]string{"team": "platform"})
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: s, Raw: prior}}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: prior}}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected the resource to be removed from state")
	}
}

func TestRepoTagsResource_Update_DiffsAgainstLiveTags(t *testing.T) {
	// Prior state predates "manual", as with -refresh=false.
	api := &fakeRepoAPI{tags: map[string]string{"team": "platform", "manual": "x"}}
	r, s := newRepoTagsTestResource(t, api.server(t))
	ctx := context.TODO()

	prior := repoTagsRaw(ctx, s, testRepoID, map[string]string{"team": "platform"})
	want := map[string]string{"team": "payments"}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: prior}}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: s, Raw: repoTagsRaw(ctx, s, testRepoID, want)},
		State: tfsdk.State{Schema: s, Raw: prior},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	assertCallSequence(t, api.calls, []string{getRepoCall, patchTagsCall, getRepoCall})
	body := patchBody(t, api.calls[1])
	if !reflect.DeepEqual(body.SetTags, map[string]string{"team": "payments"}) || !reflect.DeepEqual(body.RemoveTags, []string{"manual"}) {
		t.Errorf("unexpected PATCH body: %+v", body)
	}
	if got := stateTags(t, resp.State); !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected state tags: %v", got)
	}
}

func TestRepoTagsResource_Delete_RemovesAllTags(t *testing.T) {
	api := &fakeRepoAPI{tags: map[string]string{"team": "platform", "manual": "x"}}
	r, s := newRepoTagsTestResource(t, api.server(t))
	ctx := context.TODO()

	prior := repoTagsRaw(ctx, s, testRepoID, map[string]string{"team": "platform"})
	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: s, Raw: prior}}
	r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: prior}}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	assertCallSequence(t, api.calls, []string{getRepoCall, patchTagsCall})
	if got := api.currentTags(); len(got) != 0 {
		t.Errorf("expected the repo to have no tags, got %v", got)
	}
}

func TestRepoTagsResource_Delete_RepoGoneSucceeds(t *testing.T) {
	api := &fakeRepoAPI{}
	r, s := newRepoTagsTestResource(t, api.server(t))
	ctx := context.TODO()

	prior := repoTagsRaw(ctx, s, testRepoID, map[string]string{"team": "platform"})
	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: s, Raw: prior}}
	r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: prior}}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
}

func TestRepoTagsResource_ImportState_SetsRepoID(t *testing.T) {
	r := &repoTagsResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.TODO(), resource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema
	ctx := context.TODO()

	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: repoTagsRaw(ctx, s, "", nil)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: testRepoID}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	var repoID string
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("repo_id"), &repoID)...)
	if repoID != testRepoID {
		t.Errorf("expected repo_id %q, got %q", testRepoID, repoID)
	}
}
