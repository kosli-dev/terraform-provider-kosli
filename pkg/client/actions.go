package client

import (
	"context"
	"fmt"
	"net/http"
)

// ActionTarget represents a notification target for a Kosli action.
type ActionTarget struct {
	Type           string `json:"type"`
	Webhook        string `json:"webhook,omitempty"`
	PayloadVersion string `json:"payload_version,omitempty"`
}

// ActionRequest represents the payload for creating or updating a Kosli action.
type ActionRequest struct {
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	Environments []string       `json:"environments"`
	Triggers     []string       `json:"triggers"`
	Targets      []ActionTarget `json:"targets"`
}

// ActionResponse represents a Kosli action as returned by the API.
type ActionResponse struct {
	Name                  string         `json:"name"`
	Type                  string         `json:"type"`
	Number                int            `json:"number"`
	Environments          []string       `json:"environments"`
	Triggers              []string       `json:"triggers"`
	Targets               []ActionTarget `json:"targets"`
	CreatedBy             string         `json:"created_by"`
	IsCreatedFromSlackApp bool           `json:"is_created_from_slack_app"`
	IsFailing             bool           `json:"is_failing"`
	CreatedAt             float64        `json:"created_at"`
	LastModifiedAt        float64        `json:"last_modified_at"`
}

// ListActions retrieves all actions for the organization.
func (c *Client) ListActions(ctx context.Context) ([]ActionResponse, error) {
	path := fmt.Sprintf("/organizations/%s/environments_notifications", c.Organization())

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result []ActionResponse
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetActionByNumber retrieves a specific action by its server-assigned number.
func (c *Client) GetActionByNumber(ctx context.Context, number int) (*ActionResponse, error) {
	path := fmt.Sprintf("/organizations/%s/environments_notifications/%d", c.Organization(), number)

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result ActionResponse
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetActionByName retrieves a specific action by name, finding it from the list.
// Used for import operations where only the name is known.
func (c *Client) GetActionByName(ctx context.Context, name string) (*ActionResponse, error) {
	actions, err := c.ListActions(ctx)
	if err != nil {
		return nil, err
	}

	for _, action := range actions {
		if action.Name == name {
			return &action, nil
		}
	}

	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("action named %q not found", name)}
}

// CreateOrUpdateAction creates or updates an action.
// The API returns "OK" on success — must GET to read state.
func (c *Client) CreateOrUpdateAction(ctx context.Context, req *ActionRequest) error {
	path := fmt.Sprintf("/organizations/%s/environments_notifications", c.Organization())

	resp, err := c.Put(ctx, path, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// UpdateAction updates an existing action in-place by its server-assigned number.
// Uses PUT /environments_notifications/:number — this updates without changing the number,
// unlike PUT /environments_notifications which creates a new action for non-Slack actions.
func (c *Client) UpdateAction(ctx context.Context, number int, req *ActionRequest) error {
	path := fmt.Sprintf("/organizations/%s/environments_notifications/%d", c.Organization(), number)

	resp, err := c.Put(ctx, path, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// DeleteAction deletes an action by its server-assigned number.
func (c *Client) DeleteAction(ctx context.Context, number int) error {
	path := fmt.Sprintf("/organizations/%s/environments_notifications/%d", c.Organization(), number)

	resp, err := c.Delete(ctx, path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
