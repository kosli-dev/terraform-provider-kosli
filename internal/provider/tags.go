package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// titleCase returns s with its first byte uppercased. It is safe on empty strings.
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// applyTags computes the tag diff between oldTags and newTags and calls the API
// PATCH tags endpoint if there are any changes. resourceType is the Kosli API
// resource type string (e.g. "environment", "flow") and is also used in error messages.
func applyTags(ctx context.Context, c *client.Client, name, resourceType string, oldTags, newTags types.Map, diags *diag.Diagnostics) {
	// Extract old tag map
	oldMap := map[string]string{}
	if !oldTags.IsNull() && !oldTags.IsUnknown() {
		d := oldTags.ElementsAs(ctx, &oldMap, false)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
	}

	// Extract new tag map
	newMap := map[string]string{}
	if !newTags.IsNull() && !newTags.IsUnknown() {
		d := newTags.ElementsAs(ctx, &newMap, false)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
	}

	payload := &client.TagResourcePayload{
		SetTags:    map[string]string{},
		RemoveTags: []string{},
	}

	// Tags to add or update (in new but not in old, or value changed)
	for k, v := range newMap {
		if oldV, exists := oldMap[k]; !exists || oldV != v {
			payload.SetTags[k] = v
		}
	}

	// Tags to remove (in old but not in new)
	for k := range oldMap {
		if _, exists := newMap[k]; !exists {
			payload.RemoveTags = append(payload.RemoveTags, k)
		}
	}

	// Only call API if there are changes
	if len(payload.SetTags) == 0 && len(payload.RemoveTags) == 0 {
		return
	}

	if err := c.TagResource(ctx, resourceType, name, payload); err != nil {
		diags.AddError(
			fmt.Sprintf("Error Updating %s Tags", titleCase(resourceType)),
			fmt.Sprintf("Could not update tags for %s %q: %s", resourceType, name, err.Error()),
		)
	}
}
