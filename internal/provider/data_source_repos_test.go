package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestReposDataSource_Metadata(t *testing.T) {
	d := &reposDataSource{}

	resp := &datasource.MetadataResponse{}
	d.Metadata(context.TODO(), datasource.MetadataRequest{ProviderTypeName: "kosli"}, resp)

	if resp.TypeName != "kosli_repos" {
		t.Errorf("Expected TypeName %q, got %q", "kosli_repos", resp.TypeName)
	}
}

func TestReposDataSource_Schema(t *testing.T) {
	d := &reposDataSource{}

	resp := &datasource.SchemaResponse{}
	d.Schema(context.TODO(), datasource.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	for _, attr := range []string{"name", "search", "vcs_provider", "vcs_instance", "namespace"} {
		a, exists := attrs[attr]
		if !exists {
			t.Errorf("Expected filter attribute %q to exist in schema", attr)
			continue
		}
		if !a.IsOptional() {
			t.Errorf("Expected filter attribute %q to be optional", attr)
		}
	}

	tagFilter, ok := resp.Schema.Blocks["tag_filter"].(schema.ListNestedBlock)
	if !ok {
		t.Fatalf("Expected 'tag_filter' to be a list nested block, got %T", resp.Schema.Blocks["tag_filter"])
	}
	key, ok := tagFilter.NestedObject.Attributes["key"].(schema.StringAttribute)
	if !ok || !key.IsRequired() {
		t.Error("Expected tag_filter 'key' to be a required string")
	}
	if len(key.Validators) == 0 {
		t.Error("Expected tag_filter 'key' to reject empty keys and colons")
	}
	values, ok := tagFilter.NestedObject.Attributes["values"].(schema.ListAttribute)
	if !ok || !values.IsOptional() {
		t.Error("Expected tag_filter 'values' to be an optional list")
	}
	if len(values.Validators) == 0 {
		t.Error("Expected tag_filter 'values' to reject null and empty entries")
	}

	repos, ok := attrs["repos"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("Expected 'repos' to be a list nested attribute, got %T", attrs["repos"])
	}
	if !repos.IsComputed() {
		t.Error("Expected 'repos' attribute to be computed")
	}
	for _, attr := range []string{"id", "name", "url", "vcs_provider", "vcs_instance", "tags"} {
		if _, exists := repos.NestedObject.Attributes[attr]; !exists {
			t.Errorf("Expected repos attribute %q to exist in schema", attr)
		}
	}
}

func TestReposDataSource_Configure(t *testing.T) {
	d := &reposDataSource{}

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

func TestNewReposDataSource(t *testing.T) {
	if _, ok := NewReposDataSource().(*reposDataSource); !ok {
		t.Error("Expected data source to be of type *reposDataSource")
	}
}

// testTagFilter is one tag_filter block. Nil values are null.
type testTagFilter struct {
	key    string
	values []string
}

// reposConfig builds a raw kosli_repos config. Unset filters are null.
func reposConfig(ctx context.Context, s schema.Schema, filters map[string]string, tagFilters []testTagFilter) tftypes.Value {
	attrs := map[string]tftypes.Value{}
	for _, name := range []string{"name", "search", "vcs_provider", "vcs_instance", "namespace"} {
		if v, ok := filters[name]; ok {
			attrs[name] = tftypes.NewValue(tftypes.String, v)
		} else {
			attrs[name] = tftypes.NewValue(tftypes.String, nil)
		}
	}
	objType := s.Type().TerraformType(ctx).(tftypes.Object)
	blockType := objType.AttributeTypes["tag_filter"].(tftypes.List)
	filterType := blockType.ElementType.(tftypes.Object)
	valuesType := filterType.AttributeTypes["values"]
	blocks := make([]tftypes.Value, 0, len(tagFilters))
	for _, f := range tagFilters {
		values := tftypes.NewValue(valuesType, nil)
		if f.values != nil {
			elems := make([]tftypes.Value, 0, len(f.values))
			for _, v := range f.values {
				elems = append(elems, tftypes.NewValue(tftypes.String, v))
			}
			values = tftypes.NewValue(valuesType, elems)
		}
		blocks = append(blocks, tftypes.NewValue(filterType, map[string]tftypes.Value{
			"key":    tftypes.NewValue(tftypes.String, f.key),
			"values": values,
		}))
	}
	attrs["tag_filter"] = tftypes.NewValue(blockType, blocks)
	attrs["repos"] = tftypes.NewValue(objType.AttributeTypes["repos"], nil)
	return tftypes.NewValue(objType, attrs)
}

func runReposRead(t *testing.T, handler http.HandlerFunc, filters map[string]string, tagFilters []testTagFilter) (*datasource.ReadResponse, reposDataSourceModel) {
	t.Helper()
	ctx := context.TODO()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	d := &reposDataSource{client: c}
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema

	raw := reposConfig(ctx, s, filters, tagFilters)
	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: s, Raw: raw}}
	d.Read(ctx, req, resp)

	var state reposDataSourceModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("failed to read state: %v", diags)
		}
	}
	return resp, state
}

func TestReposDataSource_Read_PassesFiltersAndMapsRepos(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		for key, want := range map[string]string{"provider": "github", "namespace": "acme/platform"} {
			if got := q.Get(key); got != want {
				t.Errorf("expected %s=%q, got %q", key, want, got)
			}
		}
		want := []string{"team:platform", "team:payments", "tier"}
		if got := q["tag"]; !reflect.DeepEqual(got, want) {
			t.Errorf("expected tag=%v, got %v", want, got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"page": 1, "per_page": 50, "total_pages": 1, "total_count": 2,
			"repos": []map[string]any{
				{"id": "id-b", "name": "acme/platform/api", "url": "https://github.com/acme/platform/api", "provider": "github", "vcs_instance": "github.com", "tags": map[string]string{"team": "platform"}},
				{"id": "id-a", "name": "acme/platform/api", "url": "https://github.example.com/acme/platform/api", "provider": nil, "tags": map[string]string{}},
			},
		})
	}

	resp, state := runReposRead(t, handler, map[string]string{"vcs_provider": "github", "namespace": "acme/platform"}, []testTagFilter{
		{key: "team", values: []string{"platform", "payments"}},
		{key: "tier"},
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}

	if len(state.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(state.Repos))
	}
	// Same name: ordered by ID so plans stay stable.
	first, second := state.Repos[0], state.Repos[1]
	if first.ID.ValueString() != "id-a" || second.ID.ValueString() != "id-b" {
		t.Errorf("expected repos ordered by ID for equal names, got %q then %q", first.ID.ValueString(), second.ID.ValueString())
	}
	if !first.VCSProvider.IsNull() || !first.VCSInstance.IsNull() {
		t.Errorf("expected unknown provider and vcs_instance to be null, got %v and %v", first.VCSProvider, first.VCSInstance)
	}
	if got := len(first.Tags.Elements()); got != 0 || first.Tags.IsNull() {
		t.Errorf("expected empty, non-null tags, got %v", first.Tags)
	}
	if second.VCSProvider.ValueString() != "github" || second.VCSInstance.ValueString() != "github.com" {
		t.Errorf("unexpected provider fields: %v %v", second.VCSProvider, second.VCSInstance)
	}
	if second.URL.ValueString() != "https://github.com/acme/platform/api" {
		t.Errorf("unexpected url: %v", second.URL)
	}
	if v, ok := second.Tags.Elements()["team"]; !ok || v.String() != `"platform"` {
		t.Errorf("expected tag team=platform, got %v", second.Tags)
	}
}

func TestReposDataSource_Read_NoMatchesIsEmptyList(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"page": 1, "per_page": 50, "total_pages": 0, "total_count": 0, "repos": []}`))
	}

	resp, state := runReposRead(t, handler, map[string]string{"name": "missing"}, nil)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if state.Repos == nil || len(state.Repos) != 0 {
		t.Errorf("expected an empty, non-null repos list, got %v", state.Repos)
	}
}

func TestReposDataSource_Read_APIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid provider"}`))
	}

	resp, _ := runReposRead(t, handler, nil, nil)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Error Listing Repos" {
		t.Errorf("unexpected summary %q", got)
	}
}

func TestReposDataSource_ConfigValidators_NameConflictsWithSearch(t *testing.T) {
	d := &reposDataSource{}
	if got := len(d.ConfigValidators(context.TODO())); got != 1 {
		t.Fatalf("expected 1 config validator, got %d", got)
	}
}

func TestTagFilterParams(t *testing.T) {
	ctx := context.TODO()
	list := func(values ...string) types.List {
		v, _ := types.ListValueFrom(ctx, types.StringType, values)
		return v
	}

	tests := []struct {
		name    string
		filters []tagFilterModel
		want    []string
	}{
		{name: "no blocks", filters: nil, want: nil},
		{name: "values omitted matches key", filters: []tagFilterModel{{Key: types.StringValue("tier"), Values: types.ListNull(types.StringType)}}, want: []string{"tier"}},
		{name: "empty values matches key", filters: []tagFilterModel{{Key: types.StringValue("tier"), Values: list()}}, want: []string{"tier"}},
		{name: "one param per value", filters: []tagFilterModel{{Key: types.StringValue("team"), Values: list("platform", "a:b")}}, want: []string{"team:platform", "team:a:b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := tagFilterParams(ctx, tt.filters, &diags)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
