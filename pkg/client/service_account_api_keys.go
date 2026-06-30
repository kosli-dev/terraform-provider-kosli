package client

import (
	"context"
	"fmt"
	"net/http"
)

// ServiceAccountAPIKey represents an API key for a service account.
//
// The raw Key value is only populated by the create endpoint and is never
// returned again (it is SHA-256 hashed server-side). The list endpoint returns
// every field except Key.
type ServiceAccountAPIKey struct {
	ID          string  `json:"id"`
	Key         string  `json:"key"`
	Description string  `json:"description"`
	CreatedAt   float64 `json:"created_at"`
	ExpiresAt   int64   `json:"expires_at"`
	LastUsedAt  float64 `json:"last_used_at"`
}

// CreateAPIKeyRequest represents the request body for creating an API key via
// POST /api/v2/service-accounts/{org}/{name}/api-keys.
//
// ExpiresAt is a Unix timestamp (seconds); when omitted (zero) the key never
// expires. It must not be in the past.
type CreateAPIKeyRequest struct {
	Description string `json:"description"`
	ExpiresAt   int64  `json:"expires_at,omitempty"`
}

// ListServiceAccountAPIKeys retrieves all active API keys for a service account.
func (c *Client) ListServiceAccountAPIKeys(ctx context.Context, serviceAccountName string) ([]ServiceAccountAPIKey, error) {
	// Build path: GET /api/v2/service-accounts/{org}/{name}/api-keys
	path := fmt.Sprintf("/service-accounts/%s/%s/api-keys", c.Organization(), serviceAccountName)

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result []ServiceAccountAPIKey
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetServiceAccountAPIKeyByID retrieves a single API key by its id by scanning
// the list of active keys (the API has no direct GET-by-id endpoint). Returns a
// 404 APIError when no key with the given id exists.
func (c *Client) GetServiceAccountAPIKeyByID(ctx context.Context, serviceAccountName, keyID string) (*ServiceAccountAPIKey, error) {
	keys, err := c.ListServiceAccountAPIKeys(ctx, serviceAccountName)
	if err != nil {
		return nil, err
	}

	for i := range keys {
		if keys[i].ID == keyID {
			return &keys[i], nil
		}
	}

	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("api key %q not found for service account %q", keyID, serviceAccountName)}
}

// CreateServiceAccountAPIKey creates a new API key for a service account.
// The response includes the raw key value, which is only returned once.
func (c *Client) CreateServiceAccountAPIKey(ctx context.Context, serviceAccountName string, req *CreateAPIKeyRequest) (*ServiceAccountAPIKey, error) {
	// Build path: POST /api/v2/service-accounts/{org}/{name}/api-keys
	path := fmt.Sprintf("/service-accounts/%s/%s/api-keys", c.Organization(), serviceAccountName)

	resp, err := c.Post(ctx, path, req)
	if err != nil {
		return nil, err
	}

	var result ServiceAccountAPIKey
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// RevokeServiceAccountAPIKey revokes (deletes) an API key by its id.
// The endpoint returns a bare JSON string on success, which is ignored.
func (c *Client) RevokeServiceAccountAPIKey(ctx context.Context, serviceAccountName, keyID string) error {
	// Build path: DELETE /api/v2/service-accounts/{org}/{name}/api-keys/{key_id}
	path := fmt.Sprintf("/service-accounts/%s/%s/api-keys/%s", c.Organization(), serviceAccountName, keyID)

	resp, err := c.Delete(ctx, path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
