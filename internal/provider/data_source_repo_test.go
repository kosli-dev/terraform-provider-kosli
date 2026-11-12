package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestRepoDataSource_Metadata(t *testing.T) {
	d := &repoDataSource{}

	resp := &datasource.MetadataResponse{}
	d.Metadata(context.TODO(), datasource.MetadataRequest{ProviderTypeName: "kosli"}, resp)

	if resp.TypeName != "kosli_repo" {
		t.Errorf("Expected TypeName %q, got %q", "kosli_repo", resp.TypeName)
	}
}

func TestRepoDataSource_Schema(t *testing.T) {
	d := &repoDataSource{}

	resp := &datasource.SchemaResponse{}
	d.Schema(context.TODO(), datasource.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	for _, attr := range []string{"id", "name", "vcs_provider"} {
		a, exists := attrs[attr]
		if !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
			continue
		}
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("Expected attribute %q to be optional and computed", attr)
		}
	}
	for _, attr := range []string{"url", "vcs_instance", "tags"} {
		a, exists := attrs[attr]
		if !exists {
			t.Errorf("Expected attribute %q to exist in schema", attr)
			continue
		}
		if a.IsOptional() || !a.IsComputed() {
			t.Errorf("Expected attribute %q to be computed only", attr)
		}
	}
}

func TestRepoDataSource_ConfigValidators(t *testing.T) {
	d := &repoDataSource{}
	// Exactly one of id and name; vcs_provider conflicts with id.
	if got := len(d.ConfigValidators(context.TODO())); got != 2 {
		t.Fatalf("expected 2 config validators, got %d", got)
	}
}

func TestRepoDataSource_Configure(t *testing.T) {
	d := &repoDataSource{}

	resp := &datasource.ConfigureResponse{}
	d.Configure(context.TODO(), datasource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Error("Expected no errors when provider data is nil")
	}

	resp = &datasource.ConfigureResponse{}
	d.Configure(context.TODO(), datasource.ConfigureRequest{ProviderData: "wrong type"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Expected error when provider data is wrong type")
	}
}

func TestNewRepoDataSource(t *testing.T) {
	if _, ok := NewRepoDataSource().(*repoDataSource); !ok {
		t.Error("Expected data source to be of type *repoDataSource")
	}
}

// repoConfig builds a raw kosli_repo config. Empty inputs are null.
func repoConfig(ctx context.Context, s schema.Schema, id, name, vcsProvider string) tftypes.Value {
	str := func(v string) tftypes.Value {
		if v == "" {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, v)
	}
	return tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
		"id":           str(id),
		"name":         str(name),
		"vcs_provider": str(vcsProvider),
		"url":          str(""),
		"vcs_instance": str(""),
		"tags":         tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
	})
}

func runRepoRead(t *testing.T, handler http.HandlerFunc, id, name, vcsProvider string) (*datasource.ReadResponse, repoDataSourceModel) {
	t.Helper()
	ctx := context.TODO()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	d := &repoDataSource{client: c}
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema

	raw := repoConfig(ctx, s, id, name, vcsProvider)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: s, Raw: raw}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}, resp)

	var state repoDataSourceModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("failed to read state: %v", diags)
		}
	}
	return resp, state
}

// reposListHandler answers the list endpoint with repos, checking the query.
func reposListHandler(t *testing.T, wantQuery map[string]string, repos ...map[string]any) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/repos/test-org" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		for key, want := range wantQuery {
			if got := r.URL.Query().Get(key); got != want {
				t.Errorf("expected %s=%q, got %q", key, want, got)
			}
		}
		if repos == nil {
			repos = []map[string]any{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"page": 1, "per_page": 50, "total_pages": 1, "total_count": len(repos), "repos": repos})
	}
}

func TestRepoDataSource_Read_ByName(t *testing.T) {
	handler := reposListHandler(t, map[string]string{"name": "acme/api", "provider": ""},
		map[string]any{"id": "abc", "name": "acme/api", "url": "https://github.com/acme/api", "provider": "github", "vcs_instance": "github.com", "tags": map[string]string{"team": "platform"}},
	)

	resp, state := runRepoRead(t, handler, "", "acme/api", "")
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if state.ID.ValueString() != "abc" || state.Name.ValueString() != "acme/api" {
		t.Errorf("unexpected id/name: %v %v", state.ID, state.Name)
	}
	if state.VCSProvider.ValueString() != "github" || state.VCSInstance.ValueString() != "github.com" {
		t.Errorf("unexpected vcs fields: %v %v", state.VCSProvider, state.VCSInstance)
	}
	if state.URL.ValueString() != "https://github.com/acme/api" {
		t.Errorf("unexpected url: %v", state.URL)
	}
	if v, ok := state.Tags.Elements()["team"]; !ok || v.String() != `"platform"` {
		t.Errorf("expected tag team=platform, got %v", state.Tags)
	}
}

func TestRepoDataSource_Read_ByNameAndProvider_KeepsConfiguredProvider(t *testing.T) {
	// The API may store a finer provider than the one filtered on.
	handler := reposListHandler(t, map[string]string{"name": "acme/api", "provider": "bitbucket"},
		map[string]any{"id": "abc", "name": "acme/api", "url": "https://bitbucket.org/acme/api", "provider": "bitbucket_cloud"},
	)

	resp, state := runRepoRead(t, handler, "", "acme/api", "bitbucket")
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if state.VCSProvider.ValueString() != "bitbucket" {
		t.Errorf("expected configured vcs_provider to be kept, got %v", state.VCSProvider)
	}
	if state.Tags.IsNull() || len(state.Tags.Elements()) != 0 {
		t.Errorf("expected empty, non-null tags, got %v", state.Tags)
	}
}

func TestRepoDataSource_Read_ByName_NotFound(t *testing.T) {
	resp, _ := runRepoRead(t, reposListHandler(t, nil), "", "acme/missing", "")

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Repo Not Found" {
		t.Errorf("unexpected summary %q", got)
	}
}

func TestRepoDataSource_Read_ByName_Ambiguous(t *testing.T) {
	handler := reposListHandler(t, nil,
		map[string]any{"id": "a", "name": "acme/api", "url": "https://gitlab.com/acme/api", "provider": "gitlab"},
		map[string]any{"id": "b", "name": "acme/api", "url": "https://github.com/acme/api", "provider": "github"},
	)

	resp, _ := runRepoRead(t, handler, "", "acme/api", "")
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	diag := resp.Diagnostics.Errors()[0]
	if diag.Summary() != "Ambiguous Repo Name" {
		t.Errorf("unexpected summary %q", diag.Summary())
	}
	for _, want := range []string{"github, gitlab", "vcs_provider"} {
		if !strings.Contains(diag.Detail(), want) {
			t.Errorf("expected detail to contain %q, got %q", want, diag.Detail())
		}
	}
}

func TestRepoDataSource_Read_ByID(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/repos/test-org/abc" || r.URL.Query().Get("id") != "abc" {
			t.Errorf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": "abc", "name": "acme/api", "url": "https://github.com/acme/api", "provider": null, "tags": {}}`))
	}

	resp, state := runRepoRead(t, handler, "abc", "", "")
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if state.Name.ValueString() != "acme/api" {
		t.Errorf("expected name from the API, got %v", state.Name)
	}
	if !state.VCSProvider.IsNull() {
		t.Errorf("expected unknown vcs_provider to be null, got %v", state.VCSProvider)
	}
}

func TestRepoDataSource_Read_ByID_NotFound(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Repo 'abc' not found"}`))
	}

	resp, _ := runRepoRead(t, handler, "abc", "", "")
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Repo Not Found" {
		t.Errorf("unexpected summary %q", got)
	}
}
