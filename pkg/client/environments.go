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

// UpdateEnvironmentRequest represents the user-facing request format for updating
// an existing environment via PATCH /api/v2/environments/{org}/{env_name}.
//
// Unlike CreateEnvironmentRequest, the PATCH endpoint does not accept Name or
// Type (those are immutable) and "omitted fields are left unchanged". Pointer
// fields are only sent when non-nil so callers can update individual fields
// without disturbing others. To clear the description, set Description to a
// non-nil pointer to an empty string ("") — the PATCH endpoint accepts that
// (see issue #122).
type UpdateEnvironmentRequest struct {
	Description          *string  // nil to omit; pointer to "" to clear
	IncludeScaling       *bool    // nil to omit (e.g. logical environments)
	IncludedEnvironments []string // for logical environments only; nil to omit
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

// UpdateEnvironment updates an existing environment using PATCH.
//
// Unlike CreateEnvironment (which uses PUT and silently ignores attempts to
// clear the description by sending an empty string), this endpoint accepts
// empty values and applies them — for example, an empty description will
// clear the field. See issue #122 for context.
func (c *Client) UpdateEnvironment(ctx context.Context, name string, req *UpdateEnvironmentRequest) error {
	// Build path: PATCH /api/v2/environments/{org}/{env_name}
	path := fmt.Sprintf("/environments/%s/%s", c.Organization(), name)

	// Build request body. Only include optional fields when the caller
	// provided them, so fields that don't apply to a given environment
	// kind (e.g. include_scaling on a logical environment) are omitted.
	body := map[string]any{}
	if req.Description != nil {
		body["description"] = *req.Description
	}
	if req.IncludeScaling != nil {
		body["include_scaling"] = *req.IncludeScaling
	}
	if req.IncludedEnvironments != nil {
		body["included_environments"] = req.IncludedEnvironments
	}

	// Call API
	resp, err := c.Patch(ctx, path, body)
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
