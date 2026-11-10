package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Validation statuses of a rego policy version.
const (
	// RegoPolicyVersionStatusPending is a version Kosli has not validated yet.
	RegoPolicyVersionStatusPending = "pending"
	// RegoPolicyVersionStatusValid is a version an evaluation can run.
	RegoPolicyVersionStatusValid = "valid"
	// RegoPolicyVersionStatusRejected is a version whose policy is at fault.
	RegoPolicyVersionStatusRejected = "rejected"
	// RegoPolicyVersionStatusFailed is a version Kosli could not validate.
	RegoPolicyVersionStatusFailed = "failed"
)

// RegoPolicyNewestVersion identifies a policy's newest version, whatever its status.
type RegoPolicyNewestVersion struct {
	Version int64  `json:"version"`
	Status  string `json:"status"`
}

// RegoPolicyCurrentVersion is a policy's highest-numbered valid version, the
// one an evaluation runs.
type RegoPolicyCurrentVersion struct {
	Version     int64  `json:"version"`
	Digest      string `json:"digest"`
	InputSchema string `json:"input_schema"`
	// ParamsSchema is the JSON Schema the version declares for its params,
	// empty when it declares none.
	ParamsSchema json.RawMessage `json:"params_schema,omitempty"`
}

// RegoPolicy represents a Kosli rego policy as returned by the API.
type RegoPolicy struct {
	Identifier    string                  `json:"identifier"`
	Name          string                  `json:"name"`
	Description   string                  `json:"description"`
	NewestVersion RegoPolicyNewestVersion `json:"newest_version"`
	// CurrentVersion is nil until a version has been validated.
	CurrentVersion *RegoPolicyCurrentVersion `json:"current_version"`
	CreatedAt      float64                   `json:"created_at"`
	CreatedBy      string                    `json:"created_by"`
	UpdatedAt      float64                   `json:"updated_at"`
}

// RegoPolicyValidationError says why a version is rejected or failed.
type RegoPolicyValidationError struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// RegoPolicyVersion represents one version of a rego policy. Entrypoint,
// EntrypointPath and ParamsSchema are set only when the version is valid, and
// Error only when it is rejected or failed.
type RegoPolicyVersion struct {
	Identifier     string                     `json:"identifier"`
	Version        int64                      `json:"version"`
	Status         string                     `json:"status"`
	Digest         string                     `json:"digest"`
	InputSchema    string                     `json:"input_schema"`
	Comment        string                     `json:"comment"`
	CreatedAt      float64                    `json:"created_at"`
	CreatedBy      string                     `json:"created_by"`
	ValidatedAt    float64                    `json:"validated_at"`
	OPAVersion     string                     `json:"opa_version"`
	Entrypoint     string                     `json:"entrypoint"`
	EntrypointPath string                     `json:"entrypoint_path"`
	ParamsSchema   json.RawMessage            `json:"params_schema,omitempty"`
	Error          *RegoPolicyValidationError `json:"error"`
}

// RegoPolicyFile describes one file of a version's bundle.
type RegoPolicyFile struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// RegoPolicyFiles lists the files of a version's bundle.
type RegoPolicyFiles struct {
	Identifier     string           `json:"identifier"`
	Version        int64            `json:"version"`
	Digest         string           `json:"digest"`
	EntrypointPath string           `json:"entrypoint_path"`
	Files          []RegoPolicyFile `json:"files"`
}

// PublishRegoPolicyRequest is the payload for publishing a version of a rego
// policy via PUT /api/v2/rego-policies/{org}/{identifier}.
type PublishRegoPolicyRequest struct {
	// Name is required when the policy does not exist yet. Left empty, the
	// API keeps the stored name.
	Name string `json:"name,omitempty"`
	// Description is always sent: nil reaches the API as null, which clears
	// the stored description.
	Description *string `json:"description"`
	InputSchema string  `json:"input_schema"`
	Comment     string  `json:"comment,omitempty"`
	// Files maps each file's relative path in the bundle to its Rego source.
	Files map[string]string `json:"files"`
}

// PublishRegoPolicy stores the next version of a policy, creating the policy
// when the identifier is new. The version comes back as pending and is
// validated asynchronously: poll GetRegoPolicyVersion for the outcome. A
// publish whose files and input schema match a pending or valid newest
// version returns that version and stores nothing.
func (c *Client) PublishRegoPolicy(ctx context.Context, identifier string, req *PublishRegoPolicyRequest) (*RegoPolicyVersion, error) {
	path := apiPath("rego-policies", c.Organization(), identifier)

	resp, err := c.Put(ctx, path, req)
	if err != nil {
		return nil, err
	}

	var result RegoPolicyVersion
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetRegoPolicy retrieves a rego policy by identifier.
func (c *Client) GetRegoPolicy(ctx context.Context, identifier string) (*RegoPolicy, error) {
	path := apiPath("rego-policies", c.Organization(), identifier)

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result RegoPolicy
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetRegoPolicyVersion retrieves one version of a rego policy, with its
// validation status. The bundle is not returned.
func (c *Client) GetRegoPolicyVersion(ctx context.Context, identifier string, version int64) (*RegoPolicyVersion, error) {
	path := apiPath("rego-policies", c.Organization(), identifier, "versions", strconv.FormatInt(version, 10))

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result RegoPolicyVersion
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListRegoPolicyFiles lists every file in a version's bundle, whatever the
// version's status.
func (c *Client) ListRegoPolicyFiles(ctx context.Context, identifier string, version int64) (*RegoPolicyFiles, error) {
	path := apiPath("rego-policies", c.Organization(), identifier, "versions", strconv.FormatInt(version, 10), "files")

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result RegoPolicyFiles
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetRegoPolicyFile retrieves the source of one file in a version's bundle.
// filePath is the file's path in the bundle, as ListRegoPolicyFiles gives it.
func (c *Client) GetRegoPolicyFile(ctx context.Context, identifier string, version int64, filePath string) (string, error) {
	segments := []string{"rego-policies", c.Organization(), identifier, "versions", strconv.FormatInt(version, 10), "files"}
	path := apiPath(append(segments, strings.Split(filePath, "/")...)...)

	resp, err := c.Get(ctx, path)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := readBody(resp.Body, MaxResponseBodySize)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// RegoPolicyDigest computes the digest Kosli gives a bundle: "sha256:" and
// the hex SHA-256 of a manifest holding one "<path>\0<sha256 hex>\n" line per
// file, in bytewise order of the paths.
func RegoPolicyDigest(files map[string]string) string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	// Go compares strings bytewise, which is the order the API uses.
	sort.Strings(paths)

	manifest := sha256.New()
	for _, path := range paths {
		source := sha256.Sum256([]byte(files[path]))
		manifest.Write([]byte(path + "\x00" + hex.EncodeToString(source[:]) + "\n"))
	}
	return "sha256:" + hex.EncodeToString(manifest.Sum(nil))
}
