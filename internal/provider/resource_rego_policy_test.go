package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// fakeRegoVersion is one stored version in fakeRegoAPI.
type fakeRegoVersion struct {
	number      int64
	files       map[string]string
	inputSchema string
	comment     string
	// outcome is the status the version reaches once pendingPolls version
	// reads have been answered with "pending".
	outcome      string
	pendingPolls int
}

func (v *fakeRegoVersion) status() string {
	if v.pendingPolls > 0 {
		return client.RegoPolicyVersionStatusPending
	}
	return v.outcome
}

// fakeRegoAPI serves the rego policy endpoints for one policy. It mirrors the
// API's rules: a publish matching a pending or valid newest version stores
// nothing, and the current version is the highest-numbered valid one.
type fakeRegoAPI struct {
	t  *testing.T
	mu sync.Mutex

	exists      bool
	identifier  string
	name        string
	description string
	versions    []*fakeRegoVersion

	// publishStatus, when set, answers every publish with that status code.
	publishStatus int
	// pendingPolls is how many version reads a new version stays pending for.
	pendingPolls int

	publishBodies []map[string]json.RawMessage
	fileReads     int
}

// outcomeFor decides how a bundle validates from a marker in its source.
func outcomeFor(files map[string]string) string {
	for _, source := range files {
		switch {
		case strings.Contains(source, "REJECT"):
			return client.RegoPolicyVersionStatusRejected
		case strings.Contains(source, "FAIL"):
			return client.RegoPolicyVersionStatusFailed
		}
	}
	return client.RegoPolicyVersionStatusValid
}

func (f *fakeRegoAPI) current() *fakeRegoVersion {
	for i := len(f.versions) - 1; i >= 0; i-- {
		if f.versions[i].status() == client.RegoPolicyVersionStatusValid {
			return f.versions[i]
		}
	}
	return nil
}

func (f *fakeRegoAPI) versionJSON(v *fakeRegoVersion) map[string]any {
	body := map[string]any{
		"identifier": f.identifier, "version": v.number, "status": v.status(),
		"digest": client.RegoPolicyDigest(v.files), "input_schema": v.inputSchema,
		"comment": v.comment, "created_at": 100.0, "created_by": "Ada",
	}
	switch v.status() {
	case client.RegoPolicyVersionStatusValid:
		body["entrypoint"] = "data.policy"
		body["entrypoint_path"] = "policy.rego"
		body["params_schema"] = map[string]any{"type": "object"}
	case client.RegoPolicyVersionStatusRejected:
		body["error"] = map[string]any{"kind": "rego_parse_error", "message": "policy.rego:1: unexpected token"}
	case client.RegoPolicyVersionStatusFailed:
		body["error"] = map[string]any{"kind": "validator_unavailable", "message": "the validator did not answer"}
	}
	return body
}

func (f *fakeRegoAPI) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		rest := strings.TrimPrefix(r.URL.Path, "/api/v2/rego-policies/test-org/")
		parts := strings.Split(rest, "/")
		notFound := func() {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "not found"}`))
		}

		if !strings.EqualFold(parts[0], f.identifierOr(parts[0])) {
			notFound()
			return
		}

		switch {
		case r.Method == http.MethodPut && len(parts) == 1:
			f.publish(w, r, parts[0])
		case r.Method != http.MethodGet || !f.exists:
			notFound()
		case len(parts) == 1:
			body := map[string]any{
				"identifier": f.identifier, "name": f.name,
				"newest_version": map[string]any{
					"version": f.versions[len(f.versions)-1].number,
					"status":  f.versions[len(f.versions)-1].status(),
				},
				"created_at": 50.0, "created_by": "Ada", "updated_at": 100.0,
			}
			if f.description != "" {
				body["description"] = f.description
			}
			if current := f.current(); current != nil {
				body["current_version"] = map[string]any{
					"version": current.number, "digest": client.RegoPolicyDigest(current.files),
					"input_schema": current.inputSchema, "params_schema": map[string]any{"type": "object"},
				}
			}
			json.NewEncoder(w).Encode(body)
		case len(parts) >= 3 && parts[1] == "versions":
			number, _ := strconv.ParseInt(parts[2], 10, 64)
			if number < 1 || number > int64(len(f.versions)) {
				notFound()
				return
			}
			version := f.versions[number-1]
			switch {
			case len(parts) == 3:
				body := f.versionJSON(version)
				if version.pendingPolls > 0 {
					version.pendingPolls--
				}
				json.NewEncoder(w).Encode(body)
			case len(parts) == 4 && parts[3] == "files":
				paths := make([]string, 0, len(version.files))
				for path := range version.files {
					paths = append(paths, path)
				}
				sort.Strings(paths)
				files := make([]map[string]any, 0, len(paths))
				for _, path := range paths {
					files = append(files, map[string]any{"path": path, "bytes": len(version.files[path]), "sha256": "x"})
				}
				json.NewEncoder(w).Encode(map[string]any{
					"identifier": f.identifier, "version": version.number,
					"digest": client.RegoPolicyDigest(version.files), "files": files,
				})
			case len(parts) > 4 && parts[3] == "files":
				source, ok := version.files[strings.Join(parts[4:], "/")]
				if !ok {
					notFound()
					return
				}
				f.fileReads++
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte(source))
			default:
				notFound()
			}
		default:
			notFound()
		}
	}
}

func (f *fakeRegoAPI) identifierOr(fallback string) string {
	if f.exists {
		return f.identifier
	}
	return fallback
}

func (f *fakeRegoAPI) publish(w http.ResponseWriter, r *http.Request, identifier string) {
	if f.publishStatus != 0 {
		w.WriteHeader(f.publishStatus)
		w.Write([]byte(`{"message": "Controls is not enabled for this organization"}`))
		return
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		f.t.Errorf("failed to decode publish body: %v", err)
	}
	f.publishBodies = append(f.publishBodies, raw)

	var body struct {
		Name        string            `json:"name"`
		Description *string           `json:"description"`
		InputSchema string            `json:"input_schema"`
		Comment     string            `json:"comment"`
		Files       map[string]string `json:"files"`
	}
	for key, dst := range map[string]any{
		"name": &body.Name, "description": &body.Description, "input_schema": &body.InputSchema,
		"comment": &body.Comment, "files": &body.Files,
	} {
		if value, ok := raw[key]; ok {
			if err := json.Unmarshal(value, dst); err != nil {
				f.t.Errorf("failed to decode %s: %v", key, err)
			}
		}
	}

	if !f.exists {
		f.exists = true
		f.identifier = identifier
	}
	if body.Name != "" {
		f.name = body.Name
	}
	if _, sent := raw["description"]; sent {
		f.description = ""
		if body.Description != nil {
			f.description = *body.Description
		}
	}

	var version *fakeRegoVersion
	if len(f.versions) > 0 {
		newest := f.versions[len(f.versions)-1]
		reusable := newest.status() == client.RegoPolicyVersionStatusPending || newest.status() == client.RegoPolicyVersionStatusValid
		if reusable && newest.inputSchema == body.InputSchema &&
			client.RegoPolicyDigest(newest.files) == client.RegoPolicyDigest(body.Files) {
			version = newest
		}
	}
	if version == nil {
		version = &fakeRegoVersion{
			number: int64(len(f.versions) + 1), files: body.Files, inputSchema: body.InputSchema,
			comment: body.Comment, outcome: outcomeFor(body.Files), pendingPolls: f.pendingPolls,
		}
		f.versions = append(f.versions, version)
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(f.versionJSON(version))
}

// seed stores a version directly, as a publish made outside Terraform would.
func (f *fakeRegoAPI) seed(identifier, name string, files map[string]string, outcome string) {
	f.exists = true
	f.identifier = identifier
	f.name = name
	f.versions = append(f.versions, &fakeRegoVersion{
		number: int64(len(f.versions) + 1), files: files, inputSchema: "trail", outcome: outcome,
	})
}

func newRegoPolicyTestResource(t *testing.T, api *fakeRegoAPI) *regoPolicyResource {
	t.Helper()
	api.t = t
	server := httptest.NewServer(api.handler())
	t.Cleanup(server.Close)

	c, err := client.NewClient("test-token", "test-org",
		client.WithBaseURL(server.URL),
		client.WithRetryPolicy(0, time.Millisecond, time.Millisecond),
	)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
	return &regoPolicyResource{client: c, pollInterval: time.Millisecond, pollTimeout: 2 * time.Second}
}

func regoFilesMap(t *testing.T, files map[string]string) types.Map {
	t.Helper()
	value, diags := types.MapValueFrom(context.Background(), types.StringType, files)
	if diags.HasError() {
		t.Fatalf("building files map: %v", diags)
	}
	return value
}

func regoFilesFromMap(t *testing.T, value types.Map) map[string]string {
	t.Helper()
	if value.IsNull() {
		return nil
	}
	var files map[string]string
	if diags := value.ElementsAs(context.Background(), &files, false); diags.HasError() {
		t.Fatalf("reading files map: %v", diags)
	}
	return files
}

func regoPolicyModel(t *testing.T, files map[string]string) regoPolicyResourceModel {
	t.Helper()
	return regoPolicyResourceModel{
		Identifier:  types.StringValue("image-provenance"),
		Name:        types.StringValue("Image provenance"),
		Description: types.StringNull(),
		InputSchema: types.StringValue("trail"),
		Comment:     types.StringNull(),
		Files:       regoFilesMap(t, files),
	}
}

var regoValidFiles = map[string]string{
	"policy.rego":   "package policy\n\nallow := true\n",
	"lib/util.rego": "package lib\n",
}

func TestRegoPolicyResource_Metadata(t *testing.T) {
	r := &regoPolicyResource{}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "kosli"}, resp)

	if resp.TypeName != "kosli_rego_policy" {
		t.Errorf("Expected TypeName %q, got %q", "kosli_rego_policy", resp.TypeName)
	}
}

func TestRegoPolicyResource_Schema(t *testing.T) {
	r := &regoPolicyResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Error("Expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	for _, name := range []string{"identifier", "name", "input_schema", "files"} {
		if attr, ok := attrs[name]; !ok || !attr.IsRequired() {
			t.Errorf("Expected %q to exist and be required", name)
		}
	}
	for _, name := range []string{"description", "comment"} {
		if attr, ok := attrs[name]; !ok || !attr.IsOptional() || attr.IsComputed() {
			t.Errorf("Expected %q to exist and be optional only", name)
		}
	}
	for _, name := range []string{"version", "digest", "entrypoint", "params_schema", "created_at", "created_by"} {
		if attr, ok := attrs[name]; !ok || !attr.IsComputed() || attr.IsOptional() {
			t.Errorf("Expected %q to exist and be computed only", name)
		}
	}
}

func TestRegoPolicyResource_Implements(t *testing.T) {
	var _ resource.Resource = &regoPolicyResource{}
	var _ resource.ResourceWithImportState = &regoPolicyResource{}
	var _ resource.ResourceWithModifyPlan = &regoPolicyResource{}
}

func TestRegoPolicyResource_Publish_Valid(t *testing.T) {
	api := &fakeRegoAPI{}
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyModel(t, regoValidFiles)
	data.Comment = types.StringValue("first version")

	diags := r.publish(context.Background(), &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.Version.ValueInt64() != 1 {
		t.Errorf("version: got %v, want 1", data.Version)
	}
	if want := client.RegoPolicyDigest(regoValidFiles); data.Digest.ValueString() != want {
		t.Errorf("digest: got %q, want %q", data.Digest.ValueString(), want)
	}
	if data.Entrypoint.ValueString() != "data.policy" {
		t.Errorf("entrypoint: got %q", data.Entrypoint.ValueString())
	}
	if data.ParamsSchema.ValueString() != `{"type":"object"}` {
		t.Errorf("params_schema: got %q", data.ParamsSchema.ValueString())
	}
	if data.CreatedBy.ValueString() != "Ada" || data.CreatedAt.IsNull() {
		t.Errorf("created_by/created_at: got %v / %v", data.CreatedBy, data.CreatedAt)
	}
	if data.Comment.ValueString() != "first version" || !data.Description.IsNull() {
		t.Errorf("comment/description changed: %v / %v", data.Comment, data.Description)
	}

	body := api.publishBodies[0]
	if string(body["description"]) != "null" {
		t.Errorf("expected an unset description to be sent as null, got %s", body["description"])
	}
	if string(body["comment"]) != `"first version"` {
		t.Errorf("unexpected comment in request: %s", body["comment"])
	}
}

func TestRegoPolicyResource_Publish_WaitsWhilePending(t *testing.T) {
	api := &fakeRegoAPI{pendingPolls: 3}
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyModel(t, regoValidFiles)

	diags := r.publish(context.Background(), &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if data.Version.ValueInt64() != 1 || data.Entrypoint.ValueString() != "data.policy" {
		t.Errorf("unexpected state after waiting: version=%v entrypoint=%v", data.Version, data.Entrypoint)
	}
}

func TestRegoPolicyResource_Publish_Rejected(t *testing.T) {
	api := &fakeRegoAPI{pendingPolls: 1}
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyModel(t, map[string]string{"policy.rego": "package policy REJECT"})

	diags := r.publish(context.Background(), &data)
	if !diags.HasError() {
		t.Fatal("expected an error for a rejected version")
	}
	detail := diags.Errors()[0].Detail()
	for _, want := range []string{"rego_parse_error", "policy.rego:1: unexpected token", "version 1"} {
		if !strings.Contains(detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, detail)
		}
	}
}

func TestRegoPolicyResource_Publish_Failed(t *testing.T) {
	api := &fakeRegoAPI{}
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyModel(t, map[string]string{"policy.rego": "package policy FAIL"})

	diags := r.publish(context.Background(), &data)
	if !diags.HasError() {
		t.Fatal("expected an error for a failed validation")
	}
	detail := diags.Errors()[0].Detail()
	for _, want := range []string{"validator_unavailable", "apply again"} {
		if !strings.Contains(detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, detail)
		}
	}
}

func TestRegoPolicyResource_Publish_TimesOutWhilePending(t *testing.T) {
	api := &fakeRegoAPI{pendingPolls: 1 << 30}
	r := newRegoPolicyTestResource(t, api)
	r.pollTimeout = 20 * time.Millisecond
	data := regoPolicyModel(t, regoValidFiles)

	diags := r.publish(context.Background(), &data)
	if !diags.HasError() {
		t.Fatal("expected an error when validation does not finish in time")
	}
	detail := diags.Errors()[0].Detail()
	for _, want := range []string{"still pending", "Apply again"} {
		if !strings.Contains(detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, detail)
		}
	}
}

func TestRegoPolicyResource_Publish_ForbiddenAddsBetaHint(t *testing.T) {
	api := &fakeRegoAPI{publishStatus: http.StatusForbidden}
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyModel(t, regoValidFiles)

	diags := r.publish(context.Background(), &data)
	if !diags.HasError() {
		t.Fatal("expected an error for a forbidden publish")
	}
	if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, "beta feature") {
		t.Errorf("expected the beta hint, got: %s", detail)
	}
}

// A change of name alone publishes the same bundle again, which the API
// answers with the existing version.
func TestRegoPolicyResource_Publish_MetadataOnlyKeepsVersion(t *testing.T) {
	api := &fakeRegoAPI{}
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyModel(t, regoValidFiles)
	if diags := r.publish(context.Background(), &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	data.Name = types.StringValue("Renamed")
	data.Description = types.StringValue("Now described")
	if diags := r.publish(context.Background(), &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.Version.ValueInt64() != 1 || len(api.versions) != 1 {
		t.Errorf("expected the version to stay at 1, got %v (%d stored)", data.Version, len(api.versions))
	}
	if api.name != "Renamed" || api.description != "Now described" {
		t.Errorf("metadata not updated: %q / %q", api.name, api.description)
	}
	if data.Name.ValueString() != "Renamed" || data.Description.ValueString() != "Now described" {
		t.Errorf("unexpected state: %v / %v", data.Name, data.Description)
	}
}

func TestRegoPolicyResource_Publish_NewFilesMakeNewVersion(t *testing.T) {
	api := &fakeRegoAPI{}
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyModel(t, regoValidFiles)
	if diags := r.publish(context.Background(), &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	changed := map[string]string{"policy.rego": "package policy\n\nallow := false\n"}
	data.Files = regoFilesMap(t, changed)
	if diags := r.publish(context.Background(), &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.Version.ValueInt64() != 2 {
		t.Errorf("version: got %v, want 2", data.Version)
	}
	if want := client.RegoPolicyDigest(changed); data.Digest.ValueString() != want {
		t.Errorf("digest: got %q, want %q", data.Digest.ValueString(), want)
	}
}

func TestRegoPolicyResource_Publish_KeepsConfiguredSpellings(t *testing.T) {
	api := &fakeRegoAPI{}
	api.seed("Image-Provenance", "Image provenance", regoValidFiles, client.RegoPolicyVersionStatusValid)
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyModel(t, regoValidFiles)
	data.Description = types.StringValue("")

	if diags := r.publish(context.Background(), &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.Identifier.ValueString() != "image-provenance" {
		t.Errorf("expected the configured identifier spelling, got %q", data.Identifier.ValueString())
	}
	if data.Description.IsNull() || data.Description.ValueString() != "" {
		t.Errorf("expected a configured empty description to stay empty, got %v", data.Description)
	}
}

func TestRegoPolicyResource_Refresh_UnchangedReadsNoFiles(t *testing.T) {
	api := &fakeRegoAPI{}
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyModel(t, regoValidFiles)
	if diags := r.publish(context.Background(), &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	exists, diags := r.refresh(context.Background(), &data)
	if diags.HasError() || !exists {
		t.Fatalf("unexpected result: exists=%v diags=%v", exists, diags)
	}
	if api.fileReads != 0 {
		t.Errorf("expected no file downloads when the digest matches, got %d", api.fileReads)
	}
	if data.Version.ValueInt64() != 1 || len(regoFilesFromMap(t, data.Files)) != 2 {
		t.Errorf("unexpected state: version=%v files=%v", data.Version, data.Files)
	}
}

func TestRegoPolicyResource_Refresh_DetectsFilesPublishedOutsideTerraform(t *testing.T) {
	api := &fakeRegoAPI{}
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyModel(t, regoValidFiles)
	if diags := r.publish(context.Background(), &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	outside := map[string]string{"policy.rego": "package policy\n\nallow := false\n"}
	api.seed("image-provenance", "Image provenance", outside, client.RegoPolicyVersionStatusValid)

	exists, diags := r.refresh(context.Background(), &data)
	if diags.HasError() || !exists {
		t.Fatalf("unexpected result: exists=%v diags=%v", exists, diags)
	}
	got := regoFilesFromMap(t, data.Files)
	if len(got) != 1 || got["policy.rego"] != outside["policy.rego"] {
		t.Errorf("expected the files of version 2 in state, got %v", got)
	}
	if data.Version.ValueInt64() != 2 || data.Digest.ValueString() != client.RegoPolicyDigest(outside) {
		t.Errorf("unexpected version/digest: %v / %v", data.Version, data.Digest)
	}
}

// State follows the current version, so a newer rejected version leaves the
// files as they were and the configuration keeps showing a diff.
func TestRegoPolicyResource_Refresh_IgnoresNewerRejectedVersion(t *testing.T) {
	api := &fakeRegoAPI{}
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyModel(t, regoValidFiles)
	if diags := r.publish(context.Background(), &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	api.seed("image-provenance", "Image provenance", map[string]string{"policy.rego": "REJECT"}, client.RegoPolicyVersionStatusRejected)

	exists, diags := r.refresh(context.Background(), &data)
	if diags.HasError() || !exists {
		t.Fatalf("unexpected result: exists=%v diags=%v", exists, diags)
	}
	if data.Version.ValueInt64() != 1 || api.fileReads != 0 {
		t.Errorf("expected state to stay on version 1 with no downloads, got version=%v reads=%d", data.Version, api.fileReads)
	}
}

func TestRegoPolicyResource_Refresh_Import(t *testing.T) {
	api := &fakeRegoAPI{description: "Requires provenance"}
	api.seed("image-provenance", "Image provenance", regoValidFiles, client.RegoPolicyVersionStatusValid)
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyResourceModel{Identifier: types.StringValue("image-provenance")}

	exists, diags := r.refresh(context.Background(), &data)
	if diags.HasError() || !exists {
		t.Fatalf("unexpected result: exists=%v diags=%v", exists, diags)
	}
	if data.Name.ValueString() != "Image provenance" || data.Description.ValueString() != "Requires provenance" {
		t.Errorf("unexpected metadata: %v / %v", data.Name, data.Description)
	}
	if data.InputSchema.ValueString() != "trail" {
		t.Errorf("unexpected input_schema: %v", data.InputSchema)
	}
	got := regoFilesFromMap(t, data.Files)
	if len(got) != 2 || got["lib/util.rego"] != regoValidFiles["lib/util.rego"] {
		t.Errorf("expected both files downloaded, got %v", got)
	}
	if !data.Comment.IsNull() {
		t.Errorf("expected comment to stay null on import, got %v", data.Comment)
	}
}

func TestRegoPolicyResource_Refresh_NoValidVersion(t *testing.T) {
	api := &fakeRegoAPI{}
	api.seed("image-provenance", "Image provenance", map[string]string{"policy.rego": "REJECT"}, client.RegoPolicyVersionStatusRejected)
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyResourceModel{Identifier: types.StringValue("image-provenance")}

	exists, diags := r.refresh(context.Background(), &data)
	if diags.HasError() || !exists {
		t.Fatalf("unexpected result: exists=%v diags=%v", exists, diags)
	}
	if !data.Files.IsNull() || !data.Version.IsNull() || !data.Digest.IsNull() || !data.Entrypoint.IsNull() {
		t.Errorf("expected version attributes to be null, got files=%v version=%v digest=%v entrypoint=%v",
			data.Files, data.Version, data.Digest, data.Entrypoint)
	}
	if data.Name.ValueString() != "Image provenance" {
		t.Errorf("unexpected name: %v", data.Name)
	}
}

func TestRegoPolicyResource_Refresh_NotFound(t *testing.T) {
	api := &fakeRegoAPI{}
	r := newRegoPolicyTestResource(t, api)
	data := regoPolicyModel(t, regoValidFiles)

	exists, diags := r.refresh(context.Background(), &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if exists {
		t.Error("expected the policy to be reported as gone")
	}
}

func TestKeepRegoPolicyVersionAttributes(t *testing.T) {
	files := regoFilesMap(t, regoValidFiles)
	state := regoPolicyResourceModel{
		InputSchema: types.StringValue("trail"), Files: files,
		Version: types.Int64Value(3), Digest: types.StringValue("sha256:abc"),
		Entrypoint: types.StringValue("data.policy"), ParamsSchema: types.StringNull(),
	}
	unknownPlan := func() regoPolicyResourceModel {
		return regoPolicyResourceModel{
			InputSchema: types.StringValue("trail"), Files: files,
			Version: types.Int64Unknown(), Digest: types.StringUnknown(),
			Entrypoint: types.StringUnknown(), ParamsSchema: types.StringUnknown(),
		}
	}

	t.Run("same bundle keeps the version attributes", func(t *testing.T) {
		plan := unknownPlan()
		keepRegoPolicyVersionAttributes(&plan, &state)
		if plan.Version.ValueInt64() != 3 || plan.Digest.ValueString() != "sha256:abc" ||
			plan.Entrypoint.ValueString() != "data.policy" || !plan.ParamsSchema.IsNull() {
			t.Errorf("expected state values in the plan, got %+v", plan)
		}
	})

	t.Run("changed files leave them unknown", func(t *testing.T) {
		plan := unknownPlan()
		plan.Files = regoFilesMap(t, map[string]string{"policy.rego": "package other"})
		keepRegoPolicyVersionAttributes(&plan, &state)
		if !plan.Version.IsUnknown() || !plan.Digest.IsUnknown() {
			t.Errorf("expected unknown version attributes, got %+v", plan)
		}
	})

	t.Run("changed input schema leaves them unknown", func(t *testing.T) {
		plan := unknownPlan()
		plan.InputSchema = types.StringValue("trails")
		keepRegoPolicyVersionAttributes(&plan, &state)
		if !plan.Version.IsUnknown() {
			t.Errorf("expected an unknown version, got %v", plan.Version)
		}
	})

	t.Run("a state with no valid version leaves them unknown", func(t *testing.T) {
		plan := unknownPlan()
		empty := state
		empty.Version = types.Int64Null()
		keepRegoPolicyVersionAttributes(&plan, &empty)
		if !plan.Version.IsUnknown() {
			t.Errorf("expected an unknown version, got %v", plan.Version)
		}
	})
}
