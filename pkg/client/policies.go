package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
)

// Policy represents a Kosli policy as returned by the API.
type Policy struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	CreatedAt   float64         `json:"created_at"`
	Versions    []PolicyVersion `json:"versions"`
}

// PolicyVersion represents a single immutable version of a policy.
type PolicyVersion struct {
	Version   int     `json:"version"`
	Content   string  `json:"policy_yaml"`
	CreatedAt float64 `json:"timestamp"`
	CreatedBy string  `json:"created_by"`
}

// CreatePolicyRequest is the user-facing request to create or update a policy.
type CreatePolicyRequest struct {
	Name        string
	Description string
	Comment     string
	Content     string // YAML policy content
}

// CreatePolicy creates or updates a policy.
// The API returns 201 for new policies and 200 for updates.
// Per ADR 002, this method is a thin wrapper; call GetPolicy to read state after.
func (c *Client) CreatePolicy(ctx context.Context, req *CreatePolicyRequest) error {
	// Build payload JSON
	payload := map[string]any{
		"name":        req.Name,
		"description": req.Description,
		"type":        "env", // NOTE: only env currently supported
		"comment":     req.Comment,
	}

	// Build multipart body
	body, contentType, err := createPolicyMultipartRequest(payload, req.Content)
	if err != nil {
		return fmt.Errorf("failed to create multipart request: %w", err)
	}

	// Build path: PUT /api/v2/policies/{org}
	path := fmt.Sprintf("/policies/%s", c.Organization())

	// Create custom HTTP request (multipart, not JSON)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, c.apiURL+path, body)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// doRequest only supports JSON bodies; set auth/UA headers manually for this multipart request.
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseErrorResponse(resp)
	}

	return nil
}

// GetPolicy retrieves a specific policy by name.
func (c *Client) GetPolicy(ctx context.Context, name string) (*Policy, error) {
	path := fmt.Sprintf("/policies/%s/%s", c.Organization(), name)

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	log.Printf("[DEBUG] GetPolicy: received response for policy %q", name)

	var result Policy
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// ListPolicies retrieves all policies for the organization.
func (c *Client) ListPolicies(ctx context.Context) ([]Policy, error) {
	path := fmt.Sprintf("/policies/%s", c.Organization())

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result []Policy
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// createPolicyMultipartRequest builds a multipart/form-data body for policy create/update.
// Fields:
//   - "payload": JSON with name, description, type, comment
//   - "policy_file": YAML content as a file upload
func createPolicyMultipartRequest(payload map[string]any, content string) (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add payload field
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal payload: %w", err)
	}
	if err := writer.WriteField("payload", string(payloadJSON)); err != nil {
		return nil, "", fmt.Errorf("failed to write payload field: %w", err)
	}

	// Add policy_file field if content is provided
	if content != "" {
		part, err := writer.CreateFormFile("policy_file", "policy.yaml")
		if err != nil {
			return nil, "", fmt.Errorf("failed to create policy_file field: %w", err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			return nil, "", fmt.Errorf("failed to write policy_file content: %w", err)
		}
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &buf, contentType, nil
}
