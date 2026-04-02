package client

import (
	"context"
	"fmt"
)

// TagResourcePayload is the request body for PATCH /api/v2/tags/{org}/{resourceType}/{resourceID}
type TagResourcePayload struct {
	SetTags    map[string]string `json:"set_tags"`
	RemoveTags []string          `json:"remove_tags"`
}

// TagResource updates tags on a Kosli resource.
// It uses PATCH /api/v2/tags/{org}/{resourceType}/{resourceID}.
// SetTags adds or updates key-value tag pairs; RemoveTags removes tags by key.
func (c *Client) TagResource(ctx context.Context, resourceType, resourceID string, payload *TagResourcePayload) error {
	path := fmt.Sprintf("/tags/%s/%s/%s", c.Organization(), resourceType, resourceID)

	resp, err := c.Patch(ctx, path, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
