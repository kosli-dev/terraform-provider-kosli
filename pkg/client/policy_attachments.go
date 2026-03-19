package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// AttachedPolicy represents a policy attached to an environment.
type AttachedPolicy struct {
	Name string `json:"name"`
}

// AttachPolicy attaches a policy to an environment.
// POST /api/v2/environments/{org}/{env}/policies
func (c *Client) AttachPolicy(ctx context.Context, environmentName, policyName string) error {
	path := fmt.Sprintf("/environments/%s/%s/policies", c.Organization(), environmentName)
	body := map[string]any{"policy_names": []string{policyName}}

	resp, err := c.Post(ctx, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// DetachPolicy detaches a policy from an environment.
// DELETE /api/v2/environments/{org}/{env}/policies (with JSON body)
func (c *Client) DetachPolicy(ctx context.Context, environmentName, policyName string) error {
	path := fmt.Sprintf("/environments/%s/%s/policies", c.Organization(), environmentName)
	body := map[string]any{"policy_names": []string{policyName}}

	// Use doRequest directly (same package) because Delete() doesn't support a body.
	resp, err := c.doRequest(ctx, http.MethodDelete, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetEnvironmentPolicies returns the list of policies attached to an environment.
// Reads the policies from the standard GET /api/v2/environments/{org}/{env} response.
// The API may return policies as strings ("policy-name") or objects ({"name": "policy-name"}).
func (c *Client) GetEnvironmentPolicies(ctx context.Context, environmentName string) ([]AttachedPolicy, error) {
	env, err := c.GetEnvironment(ctx, environmentName)
	if err != nil {
		return nil, err
	}

	policies := make([]AttachedPolicy, 0, len(env.Policies))
	for _, p := range env.Policies {
		// Case 1: policy is a plain string — the API returns just the policy name.
		if name, ok := p.(string); ok {
			if name != "" {
				policies = append(policies, AttachedPolicy{Name: name})
			}
			continue
		}

		// Case 2: policy is an object — marshal back to JSON then unmarshal into AttachedPolicy.
		data, err := json.Marshal(p)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal policy entry: %w", err)
		}
		var ap AttachedPolicy
		if err := json.Unmarshal(data, &ap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal policy entry: %w", err)
		}
		if ap.Name != "" {
			policies = append(policies, ap)
		}
	}

	return policies, nil
}
