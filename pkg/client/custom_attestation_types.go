package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// CustomAttestationType represents a custom attestation type in Kosli.
// Contains both API format (Versions) and user-facing format (Schema, JqRules).
type CustomAttestationType struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Schema      string    `json:"-"`        // User-facing (extracted from latest version)
	JqRules     []string  `json:"-"`        // User-facing (extracted from latest version)
	Summary     string    `json:"-"`        // User-facing JSON array (extracted from latest version)
	Versions    []Version `json:"versions"` // API format (contains schema, evaluator and summary)
	Archived    bool      `json:"archived"`
	Org         string    `json:"org"`
}

// Version represents a version of a custom attestation type.
type Version struct {
	Version    int             `json:"version"`
	Timestamp  float64         `json:"timestamp"`
	TypeSchema json.RawMessage `json:"type_schema"`
	Evaluator  *Evaluator      `json:"evaluator"`
	Summary    json.RawMessage `json:"summary"`
	CreatedBy  string          `json:"created_by"`
}

// Evaluator represents the API's evaluator structure.
type Evaluator struct {
	ContentType string   `json:"content_type"`
	Rules       []string `json:"rules,omitempty"`
}

// CreateCustomAttestationTypeRequest is the user-facing request format.
type CreateCustomAttestationTypeRequest struct {
	Name        string
	Description string
	Schema      string
	JqRules     []string
	Summary     string // JSON array of {name, expression} objects; empty means "no summary"
}

// GetCustomAttestationTypeOptions contains optional parameters for GetCustomAttestationType.
type GetCustomAttestationTypeOptions struct {
	Version string // Optional version parameter
}

// toAPIFormat converts user-facing jq_rules to API's evaluator format.
func (req *CreateCustomAttestationTypeRequest) toAPIFormat() map[string]any {
	data := map[string]any{
		"name":        req.Name,
		"description": req.Description,
	}

	// Only include evaluator if jq_rules are provided
	if len(req.JqRules) > 0 {
		data["evaluator"] = map[string]any{
			"content_type": "jq",
			"rules":        req.JqRules,
		}
	}

	// Only include summary if provided. Omitting the key clears any summary
	// carried by the previous version; the API stores null for that version.
	// The raw JSON is passed through verbatim (json.Marshal validates and
	// compacts it, and surfaces malformed input as a marshalling error).
	if req.Summary != "" {
		data["summary"] = json.RawMessage(req.Summary)
	}

	return data
}

// normalizeJSON re-marshals raw JSON to canonical compact form. Empty or JSON
// null input yields an empty string, so callers can treat "absent" uniformly.
func normalizeJSON(raw json.RawMessage, field string) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}

	var obj any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", fmt.Errorf("invalid JSON in %s: %w", field, err)
	}
	normalized, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("failed to normalize %s JSON: %w", field, err)
	}
	return string(normalized), nil
}

// fromAPIFormat converts API response to user-facing format.
// Extracts schema, jq_rules and summary from the latest version in the versions array.
func (at *CustomAttestationType) fromAPIFormat() error {
	if len(at.Versions) > 0 {
		latestVersion := at.Versions[0]

		schema, err := normalizeJSON(latestVersion.TypeSchema, "type_schema")
		if err != nil {
			return err
		}
		at.Schema = schema

		summary, err := normalizeJSON(latestVersion.Summary, "summary")
		if err != nil {
			return err
		}
		at.Summary = summary

		if latestVersion.Evaluator != nil && latestVersion.Evaluator.ContentType == "jq" {
			at.JqRules = latestVersion.Evaluator.Rules
		}
	}

	return nil
}

// MarshalMultipart implements MultipartMarshaler. It encodes the request as
// multipart/form-data with a data_json field and an optional type_schema file.
func (req *CreateCustomAttestationTypeRequest) MarshalMultipart() (io.Reader, string, error) {
	if req == nil {
		return nil, "", fmt.Errorf("nil request")
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add data_json field
	dataJSON, err := json.Marshal(req.toAPIFormat())
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal data: %w", err)
	}
	if err := writer.WriteField("data_json", string(dataJSON)); err != nil {
		return nil, "", fmt.Errorf("failed to write data_json field: %w", err)
	}

	// Add type_schema field if provided
	if req.Schema != "" {
		part, err := writer.CreateFormFile("type_schema", "schema.json")
		if err != nil {
			return nil, "", fmt.Errorf("failed to create type_schema field: %w", err)
		}
		if _, err := part.Write([]byte(req.Schema)); err != nil {
			return nil, "", fmt.Errorf("failed to write type_schema content: %w", err)
		}
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &buf, contentType, nil
}

// CreateCustomAttestationType creates a new custom attestation type.
// Per ADR 002, this method is a thin wrapper that returns what the API returns.
// The API returns "OK" (201 Created), not the created object.
func (c *Client) CreateCustomAttestationType(ctx context.Context, req *CreateCustomAttestationTypeRequest) error {
	// Build path
	path := fmt.Sprintf("/custom-attestation-types/%s", c.Organization())

	// doRequest dispatches to MarshalMultipart for the multipart body
	resp, err := c.Post(ctx, path, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// This POST's API contract is exactly 201, even for existing names
	// (a new version is created). Deliberately stricter than CreatePolicy
	// and CreateFlow, whose PUT upserts legitimately return 200 or 201.
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// API returns "OK" - that's all we do
	return nil
}

// GetCustomAttestationType retrieves a specific custom attestation type.
func (c *Client) GetCustomAttestationType(ctx context.Context, name string, opts *GetCustomAttestationTypeOptions) (*CustomAttestationType, error) {
	// Build path
	path := fmt.Sprintf("/custom-attestation-types/%s/%s", c.Organization(), name)

	// Add optional version query parameter
	if opts != nil && opts.Version != "" {
		params := url.Values{}
		params.Add("version", opts.Version)
		path = fmt.Sprintf("%s?%s", path, params.Encode())
	}

	// Call API
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	// Parse response
	var result CustomAttestationType
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	// Transform from API format to user format
	if err := result.fromAPIFormat(); err != nil {
		return nil, fmt.Errorf("failed to transform API response: %w", err)
	}

	return &result, nil
}

// ListCustomAttestationTypes retrieves all custom attestation types for the organization.
func (c *Client) ListCustomAttestationTypes(ctx context.Context) ([]CustomAttestationType, error) {
	// Build path
	path := fmt.Sprintf("/custom-attestation-types/%s", c.Organization())

	// Call API
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	// Parse response
	var result []CustomAttestationType
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	// Transform each item from API format to user format
	for i := range result {
		if err := result[i].fromAPIFormat(); err != nil {
			return nil, fmt.Errorf("failed to transform item %d: %w", i, err)
		}
	}

	return result, nil
}

// ArchiveCustomAttestationType archives a custom attestation type.
func (c *Client) ArchiveCustomAttestationType(ctx context.Context, name string) error {
	// Build path
	path := fmt.Sprintf("/custom-attestation-types/%s/%s/archive", c.Organization(), name)

	// Call API with no body
	resp, err := c.Put(ctx, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
