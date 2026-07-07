package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Control represents a Kosli control as returned by the API.
//
// Archived and PoliciesReferencing are only returned by the single-get,
// create, update, and archive endpoints (not by the list endpoint).
type Control struct {
	Identifier          string            `json:"identifier"`
	Name                string            `json:"name"`
	Description         string            `json:"description"`
	Links               map[string]string `json:"links"`
	Version             int64             `json:"version"`
	CreatedAt           float64           `json:"created_at"`
	CreatedBy           string            `json:"created_by"`
	Tags                map[string]string `json:"tags"`
	Archived            bool              `json:"archived"`
	PoliciesReferencing []string          `json:"policies_referencing"`
}

// CreateControlRequest represents the request body for creating a control via
// POST /api/v2/controls/{org}.
//
// Description has no omitempty: an unset description is sent as "", which the
// create endpoint treats the same as absent.
type CreateControlRequest struct {
	Identifier  string            `json:"identifier"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Links       map[string]string `json:"links,omitempty"`
}

// UpdateControlRequest represents the request body for updating a control via
// PUT /api/v2/controls/{org}/{identifier}.
//
// PUT replaces the mutable fields wholesale, so every field is always sent
// with its resolved value ("" / empty map to clear) rather than using pointer
// null semantics. Links must be a non-nil map: a nil map marshals to JSON
// null rather than {}, and only an empty object reliably clears the field.
type UpdateControlRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Links       map[string]string `json:"links"`
}

// ControlsPage is the paginated envelope returned by the list endpoint.
type ControlsPage struct {
	Page       int       `json:"page"`
	PerPage    int       `json:"per_page"`
	TotalPages int       `json:"total_pages"`
	TotalCount int       `json:"total_count"`
	Controls   []Control `json:"controls"`
}

// ListControlsOptions are the optional query parameters for ListControls.
// Zero values are omitted from the request so the API defaults apply.
type ListControlsOptions struct {
	Page     int    // page number (API default 1)
	PerPage  int    // controls per page (API default 15, max 100)
	Search   string // case-insensitive substring match on name or identifier
	Archived bool   // include archived controls when true
}

// ListControls retrieves one page of controls for the organization.
func (c *Client) ListControls(ctx context.Context, opts *ListControlsOptions) (*ControlsPage, error) {
	// Build path: GET /api/v2/controls/{org}
	path := fmt.Sprintf("/controls/%s", c.Organization())

	if opts != nil {
		query := url.Values{}
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			query.Set("per_page", strconv.Itoa(opts.PerPage))
		}
		if opts.Search != "" {
			query.Set("search", opts.Search)
		}
		if opts.Archived {
			query.Set("archived", "true")
		}
		if encoded := query.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result ControlsPage
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListAllControls retrieves all controls for the organization, following
// pagination until every page has been fetched. Page and PerPage in opts are
// managed internally (each page is requested at the API maximum page size);
// Search and Archived are honored.
func (c *Client) ListAllControls(ctx context.Context, opts *ListControlsOptions) ([]Control, error) {
	pageOpts := ListControlsOptions{}
	if opts != nil {
		pageOpts = *opts
	}
	pageOpts.PerPage = 100 // API maximum, minimizes round trips
	pageOpts.Page = 1

	var all []Control
	for {
		result, err := c.ListControls(ctx, &pageOpts)
		if err != nil {
			return nil, err
		}
		all = append(all, result.Controls...)
		if pageOpts.Page >= result.TotalPages || len(result.Controls) == 0 {
			return all, nil
		}
		pageOpts.Page++
	}
}

// GetControl retrieves a specific control by identifier.
func (c *Client) GetControl(ctx context.Context, identifier string) (*Control, error) {
	// Build path: GET /api/v2/controls/{org}/{identifier}
	path := fmt.Sprintf("/controls/%s/%s", c.Organization(), identifier)

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result Control
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateControl creates a control and returns the created object.
// The POST endpoint responds with the full control (HTTP 201) and returns
// 409 Conflict if a control with the same identifier already exists.
func (c *Client) CreateControl(ctx context.Context, req *CreateControlRequest) (*Control, error) {
	// Build path: POST /api/v2/controls/{org}
	path := fmt.Sprintf("/controls/%s", c.Organization())

	resp, err := c.Post(ctx, path, req)
	if err != nil {
		return nil, err
	}

	var result Control
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateControl updates a control's mutable fields (name, description, links)
// using PUT and returns the updated object. Each update bumps the control's
// version.
func (c *Client) UpdateControl(ctx context.Context, identifier string, req *UpdateControlRequest) (*Control, error) {
	// Build path: PUT /api/v2/controls/{org}/{identifier}
	path := fmt.Sprintf("/controls/%s/%s", c.Organization(), identifier)

	resp, err := c.Put(ctx, path, req)
	if err != nil {
		return nil, err
	}

	var result Control
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ArchiveControl archives (soft-deletes) a control and returns the archived
// object. Controls cannot be hard-deleted via the API.
func (c *Client) ArchiveControl(ctx context.Context, identifier string) (*Control, error) {
	// Build path: POST /api/v2/controls/{org}/{identifier}/archive
	path := fmt.Sprintf("/controls/%s/%s/archive", c.Organization(), identifier)

	resp, err := c.Post(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var result Control
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
