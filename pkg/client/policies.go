package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
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

// MarshalMultipart implements MultipartMarshaler. It encodes the request as
// multipart/form-data with:
//   - "payload": JSON with name, description, type, comment
//   - "policy_file": YAML content as a file upload (only when Content is non-empty)
func (req *CreatePolicyRequest) MarshalMultipart() (io.Reader, string, error) {
	if req == nil {
		return nil, "", fmt.Errorf("nil request")
	}

	payload := map[string]any{
		"name":        req.Name,
		"description": req.Description,
		"type":        "env", // NOTE: only env currently supported
		"comment":     req.Comment,
	}

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
	if req.Content != "" {
		part, err := writer.CreateFormFile("policy_file", "policy.yaml")
		if err != nil {
			return nil, "", fmt.Errorf("failed to create policy_file field: %w", err)
		}
		if _, err := part.Write([]byte(req.Content)); err != nil {
			return nil, "", fmt.Errorf("failed to write policy_file content: %w", err)
		}
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &buf, contentType, nil
}

// CreatePolicy creates or updates a policy.
// The API returns 201 for new policies and 200 for updates.
// Per ADR 002, this method is a thin wrapper; call GetPolicy to read state after.
func (c *Client) CreatePolicy(ctx context.Context, req *CreatePolicyRequest) error {
	// Build path: PUT /api/v2/policies/{org}
	path := fmt.Sprintf("/policies/%s", c.Organization())

	// doRequest dispatches to MarshalMultipart for the multipart body
	resp, err := c.Put(ctx, path, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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
