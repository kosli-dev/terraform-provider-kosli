package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// logicalTypeWarning warns when a logical environment is declared through
// kosli_environment. The set of physical types is left to the API to validate,
// so a type the API gains works without a provider release.
type logicalTypeWarning struct{}

func (logicalTypeWarning) Description(context.Context) string {
	return "warns that logical environments are managed with kosli_logical_environment"
}

func (v logicalTypeWarning) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (logicalTypeWarning) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() != "logical" {
		return
	}
	resp.Diagnostics.AddAttributeWarning(
		req.Path,
		"Logical Environment Declared As kosli_environment",
		"kosli_environment manages physical environments and cannot set included_environments. "+
			"Use the kosli_logical_environment resource for an environment of type \"logical\".",
	)
}
