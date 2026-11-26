package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newRegoTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	c, err := NewClient("test-token", "test-org", WithBaseURL(server.URL), WithRetryPolicy(0, time.Millisecond, time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
	return c
}

func TestPublishRegoPolicy_SendsBundleAndReturnsPendingVersion(t *testing.T) {
	description := "Requires provenance"
	c := newRegoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/rego-policies/test-org/image-provenance" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["name"] != "Image provenance" {
			t.Errorf("unexpected name: %v", body["name"])
		}
		if body["description"] != description {
			t.Errorf("unexpected description: %v", body["description"])
		}
		if body["input_schema"] != "trail" {
			t.Errorf("unexpected input_schema: %v", body["input_schema"])
		}
		if body["comment"] != "first version" {
			t.Errorf("unexpected comment: %v", body["comment"])
		}
		files, _ := body["files"].(map[string]any)
		if len(files) != 1 || files["policy.rego"] != "package policy\n" {
			t.Errorf("unexpected files: %v", body["files"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"identifier": "image-provenance", "version": 1, "status": "pending",
			"digest": "sha256:abc", "input_schema": "trail", "comment": "first version",
			"created_at": 1.5, "created_by": "Ada",
		})
	})

	version, err := c.PublishRegoPolicy(context.Background(), "image-provenance", &PublishRegoPolicyRequest{
		Name:        "Image provenance",
		Description: &description,
		InputSchema: "trail",
		Comment:     "first version",
		Files:       map[string]string{"policy.rego": "package policy\n"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version.Version != 1 || version.Status != RegoPolicyVersionStatusPending || version.Digest != "sha256:abc" {
		t.Errorf("unexpected version: %+v", version)
	}
}

// A nil description must reach the API as an explicit null, which clears the
// stored description; an omitted key would keep it.
func TestPublishRegoPolicy_NilDescriptionIsSentAsNull(t *testing.T) {
	c := newRegoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]json.RawMessage
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if got, ok := body["description"]; !ok || string(got) != "null" {
			t.Errorf("expected description null, got %q (present=%v)", got, ok)
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"identifier":"p","version":2,"status":"pending","digest":"sha256:x","input_schema":"trail"}`))
	})

	_, err := c.PublishRegoPolicy(context.Background(), "p", &PublishRegoPolicyRequest{
		Name: "P", InputSchema: "trail", Files: map[string]string{"a.rego": "package a"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetRegoPolicy_Success(t *testing.T) {
	c := newRegoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/rego-policies/test-org/image-provenance" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`{
			"identifier": "image-provenance", "name": "Image provenance", "description": "d",
			"newest_version": {"version": 3, "status": "rejected"},
			"current_version": {"version": 2, "digest": "sha256:two", "input_schema": "trails",
				"params_schema": {"type": "object"}},
			"created_at": 10.5, "created_by": "Ada", "updated_at": 20.5
		}`))
	})

	policy, err := c.GetRegoPolicy(context.Background(), "image-provenance")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Name != "Image provenance" || policy.Description != "d" || policy.CreatedBy != "Ada" {
		t.Errorf("unexpected policy: %+v", policy)
	}
	if policy.NewestVersion.Version != 3 || policy.NewestVersion.Status != RegoPolicyVersionStatusRejected {
		t.Errorf("unexpected newest version: %+v", policy.NewestVersion)
	}
	if policy.CurrentVersion == nil || policy.CurrentVersion.Version != 2 || policy.CurrentVersion.InputSchema != "trails" {
		t.Fatalf("unexpected current version: %+v", policy.CurrentVersion)
	}
	if string(policy.CurrentVersion.ParamsSchema) != `{"type": "object"}` {
		t.Errorf("unexpected params schema: %s", policy.CurrentVersion.ParamsSchema)
	}
}

func TestGetRegoPolicy_NoValidVersionYet(t *testing.T) {
	c := newRegoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"identifier":"p","name":"P","newest_version":{"version":1,"status":"pending"},"created_at":1,"created_by":"Ada","updated_at":1}`))
	})

	policy, err := c.GetRegoPolicy(context.Background(), "p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.CurrentVersion != nil {
		t.Errorf("expected no current version, got %+v", policy.CurrentVersion)
	}
}

func TestGetRegoPolicy_NotFound(t *testing.T) {
	c := newRegoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Rego policy 'p' does not exist"}`))
	})

	if _, err := c.GetRegoPolicy(context.Background(), "p"); !IsNotFound(err) {
		t.Errorf("expected a not-found error, got %v", err)
	}
}

func TestGetRegoPolicyVersion_Rejected(t *testing.T) {
	c := newRegoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/rego-policies/test-org/p/versions/4" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{
			"identifier": "p", "version": 4, "status": "rejected", "digest": "sha256:four",
			"input_schema": "trail", "comment": "c", "created_at": 1, "created_by": "Ada",
			"validated_at": 2, "opa_version": "1.2.0",
			"error": {"kind": "rego_parse_error", "message": "policy.rego:3: unexpected }"}
		}`))
	})

	version, err := c.GetRegoPolicyVersion(context.Background(), "p", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version.Status != RegoPolicyVersionStatusRejected || version.Error == nil {
		t.Fatalf("unexpected version: %+v", version)
	}
	if version.Error.Kind != "rego_parse_error" || version.Error.Message != "policy.rego:3: unexpected }" {
		t.Errorf("unexpected error detail: %+v", version.Error)
	}
}

func TestGetRegoPolicyVersion_Valid(t *testing.T) {
	c := newRegoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"identifier": "p", "version": 1, "status": "valid", "digest": "sha256:one",
			"input_schema": "trail", "comment": "", "created_at": 1, "created_by": "Ada",
			"entrypoint": "data.policy", "entrypoint_path": "policy.rego",
			"params_schema": {"type":"object"}
		}`))
	})

	version, err := c.GetRegoPolicyVersion(context.Background(), "p", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version.Entrypoint != "data.policy" || version.EntrypointPath != "policy.rego" {
		t.Errorf("unexpected entrypoint: %+v", version)
	}
	if string(version.ParamsSchema) != `{"type":"object"}` {
		t.Errorf("unexpected params schema: %s", version.ParamsSchema)
	}
}

func TestListRegoPolicyFiles_Success(t *testing.T) {
	c := newRegoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/rego-policies/test-org/p/versions/2/files" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"identifier":"p","version":2,"digest":"sha256:two","files":[
			{"path":"lib/util.rego","bytes":12,"sha256":"aa"},
			{"path":"policy.rego","bytes":34,"sha256":"bb"}]}`))
	})

	files, err := c.ListRegoPolicyFiles(context.Background(), "p", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files.Files) != 2 || files.Files[0].Path != "lib/util.rego" || files.Files[1].Bytes != 34 {
		t.Errorf("unexpected files: %+v", files.Files)
	}
}

// Each path segment is escaped on its own, so a slash in the file's path stays
// a path separator and other reserved characters do not change the URL.
func TestGetRegoPolicyFile_EscapesEachSegment(t *testing.T) {
	c := newRegoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.EscapedPath(); got != "/api/v2/rego-policies/test-org/p/versions/2/files/lib/my%20util%3F.rego" {
			t.Errorf("unexpected escaped path: %s", got)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("package lib\n\nallow := true\n"))
	})

	source, err := c.GetRegoPolicyFile(context.Background(), "p", 2, "lib/my util?.rego")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "package lib\n\nallow := true\n" {
		t.Errorf("unexpected source: %q", source)
	}
}

func TestGetRegoPolicyFile_NotFound(t *testing.T) {
	c := newRegoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "no file"}`))
	})

	if _, err := c.GetRegoPolicyFile(context.Background(), "p", 2, "missing.rego"); !IsNotFound(err) {
		t.Errorf("expected a not-found error, got %v", err)
	}
}

func TestRegoPolicyDigest(t *testing.T) {
	sum := func(s string) string {
		h := sha256.Sum256([]byte(s))
		return hex.EncodeToString(h[:])
	}

	files := map[string]string{
		"policy.rego":   "package policy\n",
		"lib/util.rego": "package lib\n",
		"Zed.rego":      "package zed\n",
	}
	// Paths in bytewise order: an uppercase letter sorts before a lowercase one.
	manifest := "Zed.rego\x00" + sum("package zed\n") + "\n" +
		"lib/util.rego\x00" + sum("package lib\n") + "\n" +
		"policy.rego\x00" + sum("package policy\n") + "\n"
	want := "sha256:" + sum(manifest)

	if got := RegoPolicyDigest(files); got != want {
		t.Errorf("digest: got %s, want %s", got, want)
	}
}

func TestRegoPolicyDigest_DependsOnContentAndPath(t *testing.T) {
	base := RegoPolicyDigest(map[string]string{"a.rego": "package a"})
	if base == RegoPolicyDigest(map[string]string{"a.rego": "package b"}) {
		t.Error("expected a different digest for different content")
	}
	if base == RegoPolicyDigest(map[string]string{"b.rego": "package a"}) {
		t.Error("expected a different digest for a different path")
	}
}
