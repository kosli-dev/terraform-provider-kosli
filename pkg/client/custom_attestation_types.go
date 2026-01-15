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
	Versions    []Version `json:"versions"` // API format (contains schema and evaluator)
	Archived    bool      `json:"archived"`
	Org         string    `json:"org"`
}

// Version represents a version of a custom attestation type.
type Version struct {
	Version    int        `json:"version"`
	Timestamp  float64    `json:"timestamp"`
	TypeSchema string     `json:"type_schema"`
	Evaluator  *Evaluator `json:"evaluator"`
	CreatedBy  string     `json:"created_by"`
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
}

// GetCustomAttestationTypeOptions contains optional parameters for GetCustomAttestationType.
type GetCustomAttestationTypeOptions struct {
	Version string // Optional version parameter
}

// toAPIFormat converts user-facing jq_rules to API's evaluator format.
func (req *CreateCustomAttestationTypeRequest) toAPIFormat() map[string]any {
	return map[string]any{
		"name":        req.Name,
		"description": req.Description,
		"evaluator": map[string]any{
			"content_type": "jq",
			"rules":        req.JqRules,
		},
	}
}

// fromAPIFormat converts API response to user-facing format.
// Extracts schema and jq_rules from the latest version in the versions array.
func (at *CustomAttestationType) fromAPIFormat() {
	// Get the latest version (first element in the array - newest is first)
	if len(at.Versions) > 0 {
		latestVersion := at.Versions[0]

		// Extract schema as-is from API
		// jsontypes.Normalized in the Terraform provider handles semantic equality,
		// so we don't need to normalize the format here
		at.Schema = latestVersion.TypeSchema

		// Extract jq_rules from evaluator
		if latestVersion.Evaluator != nil && latestVersion.Evaluator.ContentType == "jq" {
			at.JqRules = latestVersion.Evaluator.Rules
		}
	}
}

// createMultipartRequest builds multipart/form-data request for POST.
func createMultipartRequest(data map[string]any, schema string) (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add data_json field
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal data: %w", err)
	}
	if err := writer.WriteField("data_json", string(dataJSON)); err != nil {
		return nil, "", fmt.Errorf("failed to write data_json field: %w", err)
	}

	// Add type_schema field if provided
	if schema != "" {
		part, err := writer.CreateFormFile("type_schema", "schema.json")
		if err != nil {
			return nil, "", fmt.Errorf("failed to create type_schema field: %w", err)
		}
		if _, err := part.Write([]byte(schema)); err != nil {
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
	// Build API-format data
	data := req.toAPIFormat()

	// Create multipart form body
	body, contentType, err := createMultipartRequest(data, req.Schema)
	if err != nil {
		return fmt.Errorf("failed to create multipart request: %w", err)
	}

	// Build path
	path := fmt.Sprintf("/custom-attestation-types/%s", c.Organization())

	// Create custom HTTP request (not using client.Post because it sends JSON)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+path, body)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers manually
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	httpReq.Header.Set("User-Agent", c.userAgent)

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Handle errors
	if resp.StatusCode >= 400 {
		return parseErrorResponse(resp)
	}

	// Verify 201 status
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
	result.fromAPIFormat()

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
		result[i].fromAPIFormat()
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
