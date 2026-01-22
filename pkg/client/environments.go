package client

import (
	"context"
	"fmt"
)

// Environment represents a Kosli environment as returned by the API
type Environment struct {
	Org               string            `json:"org"`
	Name              string            `json:"name"`
	Type              string            `json:"type"`
	Description       string            `json:"description"`
	LastModifiedAt    float64           `json:"last_modified_at"`
	LastReportedAt    *float64          `json:"last_reported_at"` // nullable
	State             any               `json:"state"`            // any JSON type
	IncludeScaling    bool              `json:"include_scaling"`
	RequireProvenance bool              `json:"require_provenance"`
	Tags              map[string]string `json:"tags"`
	Policies          []any             `json:"policies"`
	// Logical environments only:
	IncludedEnvironments []string `json:"included_environments,omitempty"`
}

// CreateEnvironmentRequest represents the user-facing request format for creating or updating an environment
type CreateEnvironmentRequest struct {
	Name                 string
	Type                 string
	Description          string
	IncludeScaling       bool
	IncludedEnvironments []string // for logical environments only
	Policies             []any    // policies to attach to the environment
}

// ListEnvironments retrieves all environments for the organization.
func (c *Client) ListEnvironments(ctx context.Context) ([]Environment, error) {
	// Build path: GET /api/v2/environments/{org}
	path := fmt.Sprintf("/environments/%s", c.Organization())

	// Call API
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	// Parse response
	var result []Environment
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetEnvironment retrieves a specific environment by name.
func (c *Client) GetEnvironment(ctx context.Context, name string) (*Environment, error) {
	// Build path: GET /api/v2/environments/{org}/{name}
	path := fmt.Sprintf("/environments/%s/%s", c.Organization(), name)

	// Call API
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	// Parse response
	var result Environment
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateEnvironment creates or updates an environment.
// Per ADR 002, this method is a thin wrapper that returns what the API returns.
// The API returns "OK" (200 OK), not the created object.
func (c *Client) CreateEnvironment(ctx context.Context, req *CreateEnvironmentRequest) error {
	// Build path
	path := fmt.Sprintf("/environments/%s", c.Organization())

	// Build request body with proper JSON structure
	body := map[string]any{
		"name":                  req.Name,
		"type":                  req.Type,
		"description":           req.Description,
		"include_scaling":       req.IncludeScaling,
		"included_environments": req.IncludedEnvironments,
		"policies":              req.Policies,
	}

	// Call API
	resp, err := c.Put(ctx, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ArchiveEnvironment archives an environment (soft delete).
func (c *Client) ArchiveEnvironment(ctx context.Context, name string) error {
	// Build path
	path := fmt.Sprintf("/environments/%s/%s/archive", c.Organization(), name)

	// Call API with no body
	resp, err := c.Put(ctx, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
