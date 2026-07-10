package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ list.ListResource = &controlListResource{}
var _ list.ListResourceWithConfigure = &controlListResource{}

// NewControlListResource creates a new control list resource, enabling
// `terraform query` (Terraform >= 1.14) to discover existing controls.
func NewControlListResource() list.ListResource {
	return &controlListResource{}
}

// controlListResource defines the list resource implementation.
type controlListResource struct {
	client *client.Client
}

// controlListModel describes the list configuration data model.
type controlListModel struct {
	Search   types.String `tfsdk:"search"`
	Archived types.Bool   `tfsdk:"archived"`
}

// Metadata returns the list resource type name, which must match the managed
// resource type name.
func (l *controlListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_control"
}

// ListResourceConfigSchema defines the schema for `list` block configuration.
func (l *controlListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "Lists Kosli controls in the organization for use with `terraform query` (Terraform >= 1.14), " +
			"so existing controls can be discovered and brought under Terraform management without hand-writing one `import` block per control.\n\n" +
			"~> **Note:** Controls is a **beta** feature and must be enabled for your organization; API requests return `403 Forbidden` otherwise.\n\n" +
			"-> The Kosli list endpoint does not return `policies_referencing`, so when resource data is included in query results " +
			"that attribute is always reported as an empty list. Archived controls are only returned when `archived = true` is set.",

		Attributes: map[string]listschema.Attribute{
			"search": listschema.StringAttribute{
				MarkdownDescription: "Case-insensitive substring to match against control names and identifiers.",
				Optional:            true,
			},
			"archived": listschema.BoolAttribute{
				MarkdownDescription: "Include archived controls in the results. Defaults to `false`.",
				Optional:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the list resource.
func (l *controlListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected List Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	l.client = c
}

// List streams one result per control matching the configured filters.
func (l *controlListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	var config controlListModel

	if diags := req.Config.Get(ctx, &config); diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	// Null attributes yield zero values, which the client omits from the
	// request query string.
	opts := &client.ListControlsOptions{
		Search:   config.Search.ValueString(),
		Archived: config.Archived.ValueBool(),
	}

	// The client transparently paginates over the whole collection.
	controls, err := l.client.ListControls(ctx, opts)
	if err != nil {
		detail := fmt.Sprintf("Could not list controls: %s", err.Error())
		if client.IsForbidden(err) {
			detail += controlBetaHint
		}
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Error Listing Controls", detail),
		})
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for i := range controls {
			if req.Limit > 0 && int64(i) >= req.Limit {
				return
			}

			control := &controls[i]

			result := req.NewListResult(ctx)
			result.DisplayName = control.Name

			result.Diagnostics.Append(result.Identity.Set(ctx, controlResourceIdentityModel{
				Identifier: types.StringValue(control.Identifier),
			})...)

			if req.IncludeResource {
				var data controlResourceModel
				// The list endpoint does not return policies_referencing;
				// mapControlToState normalizes it to an empty list.
				mapControlToState(ctx, control, &data, &result.Diagnostics)
				if !result.Diagnostics.HasError() {
					result.Diagnostics.Append(result.Resource.Set(ctx, &data)...)
				}
			}

			if !push(result) {
				return
			}
		}
	}
}
