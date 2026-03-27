package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// Flow represents a Kosli flow as returned by the API.
type Flow struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Visibility  string            `json:"visibility"`
	Template    string            `json:"template"`
	Tags        map[string]string `json:"tags"`
}

// CreateFlowRequest is the user-facing request format for creating or updating a flow.
type CreateFlowRequest struct {
	Name        string
	Description string
	Visibility  string
	Template    string // YAML content; empty means no template
}

// createFlowMultipartRequest builds a multipart/form-data body for flow creation with a template.
// Fields:
//   - payload: JSON with name, description, visibility
//   - template_file: YAML template content
func createFlowMultipartRequest(payload map[string]any, template string) (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add payload field (JSON metadata)
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal payload: %w", err)
	}
	if err := writer.WriteField("payload", string(payloadJSON)); err != nil {
		return nil, "", fmt.Errorf("failed to write payload field: %w", err)
	}

	// Add template_file field
	part, err := writer.CreateFormFile("template_file", "template.yml")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create template_file field: %w", err)
	}
	if _, err := part.Write([]byte(template)); err != nil {
		return nil, "", fmt.Errorf("failed to write template_file content: %w", err)
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &buf, contentType, nil
}

// CreateFlow creates or updates a flow.
// Per ADR 002, this method is a thin wrapper that returns what the API returns.
// When template is provided, uses multipart PUT to /flows/{org}/template_file.
// When template is empty, uses JSON PUT to /flows/{org}.
func (c *Client) CreateFlow(ctx context.Context, req *CreateFlowRequest) error {
	payload := map[string]any{
		"name":        req.Name,
		"description": req.Description,
		"visibility":  req.Visibility,
	}

	if req.Template != "" {
		// Use multipart endpoint when template is provided
		path := fmt.Sprintf("/flows/%s/template_file", c.Organization())

		body, contentType, err := createFlowMultipartRequest(payload, req.Template)
		if err != nil {
			return fmt.Errorf("failed to create multipart request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, c.apiURL+path, body)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		httpReq.Header.Set("Content-Type", contentType)
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
		httpReq.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return fmt.Errorf("failed to execute request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return parseErrorResponse(resp)
		}

		return nil
	}

	// Use JSON endpoint when no template
	path := fmt.Sprintf("/flows/%s", c.Organization())

	resp, err := c.Put(ctx, path, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetFlow retrieves a specific flow by name.
func (c *Client) GetFlow(ctx context.Context, name string) (*Flow, error) {
	path := fmt.Sprintf("/flows/%s/%s", c.Organization(), name)

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result Flow
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListFlows retrieves all flows for the organization.
func (c *Client) ListFlows(ctx context.Context) ([]Flow, error) {
	path := fmt.Sprintf("/flows/%s", c.Organization())

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result []Flow
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ArchiveFlow archives a flow (soft delete).
func (c *Client) ArchiveFlow(ctx context.Context, name string) error {
	path := fmt.Sprintf("/flows/%s/%s/archive", c.Organization(), name)

	resp, err := c.Put(ctx, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
