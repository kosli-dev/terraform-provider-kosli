package client

import (
	"context"
	"fmt"
)

// Notification types Kosli raises itself and lets an organization configure.
const (
	// NotificationTypeAPIKeyExpiry is raised when a service account API key is about to expire.
	NotificationTypeAPIKeyExpiry = "api_key_expiry" //nolint:gosec // G101: notification type identifier, not a credential
)

// Notification target types accepted by the notification config API.
const (
	NotificationTargetTypeEmail   = "EMAIL"
	NotificationTargetTypeSlack   = "SLACK"
	NotificationTargetTypeWebhook = "WEBHOOK"
)

// NotificationPayloadVersionV1 is the WEBHOOK target payload format version 1.0.
const NotificationPayloadVersionV1 = "1.0"

// NotificationTarget is one destination for a notification type.
//
// Which fields apply depends on Type: EMAIL uses Emails, SLACK uses Webhook,
// and WEBHOOK uses Webhook and PayloadVersion.
type NotificationTarget struct {
	Type           string   `json:"type"`
	Webhook        string   `json:"webhook,omitempty"`
	PayloadVersion string   `json:"payload_version,omitempty"`
	Emails         []string `json:"emails,omitempty"`
}

// NotificationConfigRequest is the payload for creating or updating a notification config.
type NotificationConfigRequest struct {
	// Targets replace the recipients Kosli would otherwise derive. The API
	// requires at least one target; delete the config to restore the derived recipients.
	Targets []NotificationTarget `json:"targets"`
}

// NotificationFailure describes the most recent failed delivery of a notification type.
type NotificationFailure struct {
	FailedAt float64 `json:"failed_at"`
	Error    string  `json:"error"`
}

// NotificationConfigResponse is a notification config as returned by the API.
type NotificationConfigResponse struct {
	NotificationType string `json:"notification_type"`
	// Scope is the entity the config applies to, or nil for the whole organization.
	Scope *string `json:"scope"`
	// Targets is empty when the organization has not configured any, in which
	// case Kosli derives the recipients.
	Targets   []NotificationTarget `json:"targets"`
	Detection map[string]any       `json:"detection"`
	// LastNotifiedAt is nil when no notification of this type has been sent.
	LastNotifiedAt *float64 `json:"last_notified_at"`
	// LastFailure is nil unless the most recent delivery failed.
	LastFailure *NotificationFailure `json:"last_failure"`
}

// GetNotificationConfig retrieves the organization's config for one notification type.
//
// The API answers 200 for every known notification type, configured or not:
// an unconfigured type comes back with an empty Targets list.
func (c *Client) GetNotificationConfig(ctx context.Context, notificationType string) (*NotificationConfigResponse, error) {
	path := fmt.Sprintf("/notification-config/%s/%s", c.Organization(), notificationType)

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result NotificationConfigResponse
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SetNotificationConfig creates or replaces the organization's config for one
// notification type and returns the stored config. Requires admin permissions.
func (c *Client) SetNotificationConfig(ctx context.Context, notificationType string, req *NotificationConfigRequest) (*NotificationConfigResponse, error) {
	path := fmt.Sprintf("/notification-config/%s/%s", c.Organization(), notificationType)

	resp, err := c.Put(ctx, path, req)
	if err != nil {
		return nil, err
	}

	var result NotificationConfigResponse
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteNotificationConfig removes the organization's config for one
// notification type, so Kosli derives the recipients again. Requires admin permissions.
func (c *Client) DeleteNotificationConfig(ctx context.Context, notificationType string) error {
	path := fmt.Sprintf("/notification-config/%s/%s", c.Organization(), notificationType)

	resp, err := c.Delete(ctx, path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
