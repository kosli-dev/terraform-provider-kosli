package client

import (
	"context"
	"fmt"
)

// ServiceAccount represents a Kosli service account as returned by the API.
type ServiceAccount struct {
	Name           string  `json:"name"`
	DisplayName    string  `json:"display_name"`
	Description    string  `json:"description"`
	Privilege      string  `json:"privilege"`
	CreatingUserID string  `json:"creating_user_id"`
	CreatedAt      float64 `json:"created_at"`
	// ForWebhook is only returned by the single-get, create, and update
	// endpoints (not by the list endpoint).
	ForWebhook bool `json:"for_webhook"`
}

// CreateServiceAccountRequest represents the request body for creating a
// service account via POST /api/v2/service-accounts/{org}.
type CreateServiceAccountRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Privilege   string `json:"privilege"`
}

// UpdateServiceAccountRequest represents the request body for updating a
// service account via PATCH /api/v2/service-accounts/{org}/{name}.
//
// The PATCH endpoint treats a JSON null description as "clear the description"
// and an omitted field as "leave unchanged". Description is a pointer without
// omitempty so a nil value marshals to JSON null (clearing the field), while a
// pointer to a string sets it. Privilege is always sent.
type UpdateServiceAccountRequest struct {
	Description *string `json:"description"`
	Privilege   string  `json:"privilege"`
}

// ListServiceAccounts retrieves all service accounts for the organization.
func (c *Client) ListServiceAccounts(ctx context.Context) ([]ServiceAccount, error) {
	// Build path: GET /api/v2/service-accounts/{org}
	path := fmt.Sprintf("/service-accounts/%s", c.Organization())

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result []ServiceAccount
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetServiceAccount retrieves a specific service account by name.
func (c *Client) GetServiceAccount(ctx context.Context, name string) (*ServiceAccount, error) {
	// Build path: GET /api/v2/service-accounts/{org}/{name}
	path := fmt.Sprintf("/service-accounts/%s/%s", c.Organization(), name)

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result ServiceAccount
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateServiceAccount creates a service account and returns the created object.
// The POST endpoint responds with the full service account (HTTP 201).
func (c *Client) CreateServiceAccount(ctx context.Context, req *CreateServiceAccountRequest) (*ServiceAccount, error) {
	// Build path: POST /api/v2/service-accounts/{org}
	path := fmt.Sprintf("/service-accounts/%s", c.Organization())

	resp, err := c.Post(ctx, path, req)
	if err != nil {
		return nil, err
	}

	var result ServiceAccount
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateServiceAccount updates a service account's description and/or privilege
// using PATCH and returns the updated object.
func (c *Client) UpdateServiceAccount(ctx context.Context, name string, req *UpdateServiceAccountRequest) (*ServiceAccount, error) {
	// Build path: PATCH /api/v2/service-accounts/{org}/{name}
	path := fmt.Sprintf("/service-accounts/%s/%s", c.Organization(), name)

	resp, err := c.Patch(ctx, path, req)
	if err != nil {
		return nil, err
	}

	var result ServiceAccount
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteServiceAccount removes a service account from the organization.
// The endpoint returns a bare JSON string on success, which is ignored.
func (c *Client) DeleteServiceAccount(ctx context.Context, name string) error {
	// Build path: DELETE /api/v2/service-accounts/{org}/{name}
	path := fmt.Sprintf("/service-accounts/%s/%s", c.Organization(), name)

	resp, err := c.Delete(ctx, path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
