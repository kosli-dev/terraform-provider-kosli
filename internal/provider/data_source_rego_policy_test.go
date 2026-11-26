package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func newRegoPolicyTestDataSource(t *testing.T, api *fakeRegoAPI) *regoPolicyDataSource {
	t.Helper()
	return &regoPolicyDataSource{client: newRegoPolicyTestResource(t, api).client}
}

func regoPolicyDataSourceModelFor(identifier string) regoPolicyDataSourceModel {
	return regoPolicyDataSourceModel{
		Identifier: types.StringValue(identifier),
		Version:    types.Int64Null(),
		Archived:   types.BoolNull(),
	}
}

func TestRegoPolicyDataSource_Metadata(t *testing.T) {
	d := &regoPolicyDataSource{}
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "kosli"}, resp)

	if resp.TypeName != "kosli_rego_policy" {
		t.Errorf("Expected TypeName %q, got %q", "kosli_rego_policy", resp.TypeName)
	}
}

func TestRegoPolicyDataSource_Schema(t *testing.T) {
	d := &regoPolicyDataSource{}
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	if attr, ok := attrs["identifier"]; !ok || !attr.IsRequired() {
		t.Error("Expected 'identifier' to exist and be required")
	}
	if attr, ok := attrs["version"]; !ok || !attr.IsOptional() || !attr.IsComputed() {
		t.Error("Expected 'version' to exist and be optional and computed")
	}
	if attr, ok := attrs["archived"]; !ok || !attr.IsOptional() {
		t.Error("Expected 'archived' to exist and be optional")
	}
	for _, name := range []string{"name", "description", "input_schema", "status", "comment", "files", "digest", "entrypoint", "params_schema", "created_at", "created_by"} {
		if attr, ok := attrs[name]; !ok || !attr.IsComputed() || attr.IsOptional() {
			t.Errorf("Expected %q to exist and be computed only", name)
		}
	}
}

func TestRegoPolicyDataSource_Implements(t *testing.T) {
	var _ datasource.DataSource = &regoPolicyDataSource{}
}

func TestRegoPolicyDataSource_Read_CurrentVersion(t *testing.T) {
	api := &fakeRegoAPI{description: "Requires provenance"}
	api.seed("image-provenance", "Image provenance", regoValidFiles, client.RegoPolicyVersionStatusValid)
	d := newRegoPolicyTestDataSource(t, api)
	data := regoPolicyDataSourceModelFor("image-provenance")

	if diags := d.read(context.Background(), &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.Name.ValueString() != "Image provenance" || data.Description.ValueString() != "Requires provenance" {
		t.Errorf("unexpected metadata: %v / %v", data.Name, data.Description)
	}
	if data.Version.ValueInt64() != 1 || data.Status.ValueString() != client.RegoPolicyVersionStatusValid {
		t.Errorf("unexpected version/status: %v / %v", data.Version, data.Status)
	}
	if data.InputSchema.ValueString() != "trail" || data.Entrypoint.ValueString() != "data.policy" {
		t.Errorf("unexpected input_schema/entrypoint: %v / %v", data.InputSchema, data.Entrypoint)
	}
	if want := client.RegoPolicyDigest(regoValidFiles); data.Digest.ValueString() != want {
		t.Errorf("digest: got %q, want %q", data.Digest.ValueString(), want)
	}
	if data.ParamsSchema.ValueString() != `{"type":"object"}` {
		t.Errorf("params_schema: got %q", data.ParamsSchema.ValueString())
	}
	if data.Archived.IsNull() || data.Archived.ValueBool() {
		t.Errorf("expected archived to be false, got %v", data.Archived)
	}
	if data.CreatedBy.ValueString() != "Ada" || data.CreatedAt.IsNull() {
		t.Errorf("created_by/created_at: got %v / %v", data.CreatedBy, data.CreatedAt)
	}

	got := regoFilesFromMap(t, data.Files)
	if len(got) != 2 || got["lib/util.rego"] != regoValidFiles["lib/util.rego"] {
		t.Errorf("unexpected files: %v", got)
	}
	// The whole bundle comes in one request, not one request per file.
	if api.bundleReads != 1 || api.fileReads != 0 {
		t.Errorf("expected 1 bundle read and no file reads, got %d and %d", api.bundleReads, api.fileReads)
	}
}

// Without a version the data source reads the current version, which is the
// newest valid one, and not a newer rejected version.
func TestRegoPolicyDataSource_Read_SkipsNewerRejectedVersion(t *testing.T) {
	api := &fakeRegoAPI{}
	api.seed("image-provenance", "Image provenance", regoValidFiles, client.RegoPolicyVersionStatusValid)
	api.seed("image-provenance", "Image provenance", map[string]string{"policy.rego": "REJECT"}, client.RegoPolicyVersionStatusRejected)
	d := newRegoPolicyTestDataSource(t, api)
	data := regoPolicyDataSourceModelFor("image-provenance")

	if diags := d.read(context.Background(), &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if data.Version.ValueInt64() != 1 || len(regoFilesFromMap(t, data.Files)) != 2 {
		t.Errorf("expected the current version 1, got version=%v files=%v", data.Version, data.Files)
	}
}

func TestRegoPolicyDataSource_Read_RequestedVersion(t *testing.T) {
	rejected := map[string]string{"policy.rego": "REJECT"}
	api := &fakeRegoAPI{}
	api.seed("image-provenance", "Image provenance", regoValidFiles, client.RegoPolicyVersionStatusValid)
	api.seed("image-provenance", "Image provenance", rejected, client.RegoPolicyVersionStatusRejected)
	d := newRegoPolicyTestDataSource(t, api)
	data := regoPolicyDataSourceModelFor("image-provenance")
	data.Version = types.Int64Value(2)

	if diags := d.read(context.Background(), &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if data.Version.ValueInt64() != 2 || data.Status.ValueString() != client.RegoPolicyVersionStatusRejected {
		t.Errorf("unexpected version/status: %v / %v", data.Version, data.Status)
	}
	if !data.Entrypoint.IsNull() || !data.ParamsSchema.IsNull() {
		t.Errorf("expected no entrypoint or params schema for a rejected version, got %v / %v", data.Entrypoint, data.ParamsSchema)
	}
	if got := regoFilesFromMap(t, data.Files); got["policy.rego"] != "REJECT" {
		t.Errorf("expected the files of version 2, got %v", got)
	}
	if want := client.RegoPolicyDigest(rejected); data.Digest.ValueString() != want {
		t.Errorf("digest: got %q, want %q", data.Digest.ValueString(), want)
	}
}

func TestRegoPolicyDataSource_Read_VersionNotFound(t *testing.T) {
	api := &fakeRegoAPI{}
	api.seed("image-provenance", "Image provenance", regoValidFiles, client.RegoPolicyVersionStatusValid)
	d := newRegoPolicyTestDataSource(t, api)
	data := regoPolicyDataSourceModelFor("image-provenance")
	data.Version = types.Int64Value(9)

	diags := d.read(context.Background(), &data)
	if !diags.HasError() || diags.Errors()[0].Summary() != "Rego Policy Version Not Found" {
		t.Fatalf("expected a version-not-found error, got: %v", diags)
	}
}

func TestRegoPolicyDataSource_Read_NoValidVersion(t *testing.T) {
	api := &fakeRegoAPI{}
	api.seed("image-provenance", "Image provenance", map[string]string{"policy.rego": "REJECT"}, client.RegoPolicyVersionStatusRejected)
	d := newRegoPolicyTestDataSource(t, api)
	data := regoPolicyDataSourceModelFor("image-provenance")

	diags := d.read(context.Background(), &data)
	if !diags.HasError() || diags.Errors()[0].Summary() != "Rego Policy Has No Valid Version" {
		t.Fatalf("expected a no-valid-version error, got: %v", diags)
	}
	if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, "version = 1") {
		t.Errorf("expected the detail to name the newest version, got: %s", detail)
	}
}

func TestRegoPolicyDataSource_Read_NotFound(t *testing.T) {
	d := newRegoPolicyTestDataSource(t, &fakeRegoAPI{})
	data := regoPolicyDataSourceModelFor("missing")

	diags := d.read(context.Background(), &data)
	if !diags.HasError() || diags.Errors()[0].Summary() != "Rego Policy Not Found" {
		t.Fatalf("expected a not-found error, got: %v", diags)
	}
}

func TestRegoPolicyDataSource_Read_Archived(t *testing.T) {
	newAPI := func(archived bool) *fakeRegoAPI {
		api := &fakeRegoAPI{}
		api.seed("image-provenance", "Image provenance", regoValidFiles, client.RegoPolicyVersionStatusValid)
		api.archived = archived
		return api
	}

	t.Run("an archived policy is refused by default", func(t *testing.T) {
		d := newRegoPolicyTestDataSource(t, newAPI(true))
		data := regoPolicyDataSourceModelFor("image-provenance")
		diags := d.read(context.Background(), &data)
		if !diags.HasError() || diags.Errors()[0].Summary() != "Rego Policy Archived" {
			t.Fatalf("expected an archived error, got: %v", diags)
		}
	})

	t.Run("archived = true reads an archived policy", func(t *testing.T) {
		d := newRegoPolicyTestDataSource(t, newAPI(true))
		data := regoPolicyDataSourceModelFor("image-provenance")
		data.Archived = types.BoolValue(true)
		if diags := d.read(context.Background(), &data); diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !data.Archived.ValueBool() || data.Version.ValueInt64() != 1 {
			t.Errorf("unexpected state: archived=%v version=%v", data.Archived, data.Version)
		}
	})

	t.Run("archived = true refuses an active policy", func(t *testing.T) {
		d := newRegoPolicyTestDataSource(t, newAPI(false))
		data := regoPolicyDataSourceModelFor("image-provenance")
		data.Archived = types.BoolValue(true)
		diags := d.read(context.Background(), &data)
		if !diags.HasError() || diags.Errors()[0].Summary() != "Rego Policy Not Archived" {
			t.Fatalf("expected a not-archived error, got: %v", diags)
		}
	})
}

func TestRegoPolicyDataSource_Read_ForbiddenAddsRolloutHint(t *testing.T) {
	api := &fakeRegoAPI{readStatus: 403}
	d := newRegoPolicyTestDataSource(t, api)
	data := regoPolicyDataSourceModelFor("image-provenance")

	diags := d.read(context.Background(), &data)
	if !diags.HasError() {
		t.Fatal("expected an error for a forbidden read")
	}
	if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, "rolled out per organization") {
		t.Errorf("expected the rollout hint, got: %s", detail)
	}
}

// The files must be the ones the digest describes. A bundle that does not
// hash to the version's digest is refused, so files and digest cannot
// disagree in state.
func TestRegoPolicyDataSource_Read_BundleDoesNotMatchDigest(t *testing.T) {
	api := &fakeRegoAPI{bundleSuffix: "# altered\n"}
	api.seed("image-provenance", "Image provenance", regoValidFiles, client.RegoPolicyVersionStatusValid)
	d := newRegoPolicyTestDataSource(t, api)
	data := regoPolicyDataSourceModelFor("image-provenance")

	diags := d.read(context.Background(), &data)
	if !diags.HasError() || diags.Errors()[0].Summary() != "Rego Policy Bundle Mismatch" {
		t.Fatalf("expected a bundle mismatch error, got: %v", diags)
	}
	if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, client.RegoPolicyDigest(regoValidFiles)) {
		t.Errorf("expected the detail to name the digest Kosli reports, got: %s", detail)
	}
}

func TestRegoPolicyDataSource_Read_BundleEndpointMissing(t *testing.T) {
	api := &fakeRegoAPI{bundleStatus: 404}
	api.seed("image-provenance", "Image provenance", regoValidFiles, client.RegoPolicyVersionStatusValid)
	d := newRegoPolicyTestDataSource(t, api)
	data := regoPolicyDataSourceModelFor("image-provenance")

	diags := d.read(context.Background(), &data)
	if !diags.HasError() || diags.Errors()[0].Summary() != "Rego Policy Bundle Not Found" {
		t.Fatalf("expected a bundle-not-found error, got: %v", diags)
	}
	if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, "does not serve") {
		t.Errorf("expected the detail to point at the server, got: %s", detail)
	}
}
