package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &controlDataSource{}

// NewControlDataSource creates a new control data source.
func NewControlDataSource() datasource.DataSource {
	return &controlDataSource{}
}

// controlDataSource defines the data source implementation.
type controlDataSource struct {
	client *client.Client
}

// controlDataSourceModel describes the data source data model.
type controlDataSourceModel struct {
	Identifier          types.String `tfsdk:"identifier"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Links               types.Map    `tfsdk:"links"`
	Version             types.Int64  `tfsdk:"version"`
	Status              types.String `tfsdk:"status"`
	CreatedAt           types.String `tfsdk:"created_at"`
	CreatedBy           types.String `tfsdk:"created_by"`
	Tags                types.Map    `tfsdk:"tags"`
	Archived            types.Bool   `tfsdk:"archived"`
	PoliciesReferencing types.List   `tfsdk:"policies_referencing"`
}

// Metadata returns the data source type name.
func (d *controlDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_control"
}

// Schema defines the schema for the data source.
func (d *controlDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches details of an existing Kosli control. Use this data source to reference controls and access metadata such as the version, tags, and referencing policies.\n\n" +
			"~> **Note:** Controls is a **beta** feature and must be enabled for your organization; API requests return `403 Forbidden` otherwise.",

		Attributes: map[string]schema.Attribute{
			"identifier": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique identifier of the control to query (e.g. `SDLC-001`).",
				Validators: []validator.String{
					stringvalidator.RegexMatches(controlIdentifierRegexp, "must start with a letter or number and contain only letters, numbers, periods, hyphens, underscores, and tildes"),
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Human-readable display name of the control.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the control.",
			},
			"links": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Named links related to the control, as a map of link name to URL.",
			},
			"version": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Version of the control to read. Every update to a control creates a new version; set this to read the name, description, and links of a specific version. Defaults to the latest version.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Status of the requested version (e.g. `created`). Only populated when `version` is set; the latest-control endpoint does not report a status.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 UTC timestamp of when the control was created. When `version` is set, this is when that version was created.",
			},
			"created_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of the user who created the control. When `version` is set, this is who created that version.",
			},
			"tags": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Tags on the control, as a map of tag key to value.",
			},
			"archived": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the control is archived. Deleting a `kosli_control` resource archives the control rather than hard-deleting it, and by default reading an archived control fails as if it did not exist. Set to `true` to read an archived control. Defaults to `false`.",
			},
			"policies_referencing": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Names of the environment policies that reference this control.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *controlDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

// Read refreshes the Terraform state with the latest data.
func (d *controlDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data controlDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	control, err := d.client.GetControl(ctx, data.Identifier.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError(
				"Control Not Found",
				fmt.Sprintf("Control %q does not exist in the organization.", data.Identifier.ValueString()),
			)
			return
		}
		detail := fmt.Sprintf("Could not read control %q: %s", data.Identifier.ValueString(), err.Error())
		if client.IsForbidden(err) {
			detail += controlBetaHint
		}
		resp.Diagnostics.AddError("Error Reading Control", detail)
		return
	}

	// Deleting a kosli_control archives it, so the archived argument asserts
	// the expected state: unset (or false) reads only live controls, true
	// reads only archived ones.
	if control.Archived != data.Archived.ValueBool() {
		if control.Archived {
			resp.Diagnostics.AddError(
				"Control Archived",
				fmt.Sprintf("Control %q exists but is archived. Set `archived = true` to read archived controls.", data.Identifier.ValueString()),
			)
		} else {
			resp.Diagnostics.AddError(
				"Control Not Archived",
				fmt.Sprintf("Control %q is not archived, but `archived = true` was set. Remove the argument to read live controls.", data.Identifier.ValueString()),
			)
		}
		return
	}

	// Versioned fields come from the requested version when one is set, and
	// from the control (latest version) otherwise. Control-level fields
	// (tags, policies_referencing, archived) always come from the control.
	name := control.Name
	description := control.Description
	links := control.Links
	version := control.Version
	createdAt := control.CreatedAt
	createdBy := control.CreatedBy
	// The latest-control endpoint does not report a version status.
	data.Status = types.StringNull()
	if !data.Version.IsNull() {
		controlVersion, err := d.client.GetControlVersion(ctx, data.Identifier.ValueString(), data.Version.ValueInt64())
		if err != nil {
			if client.IsNotFound(err) {
				resp.Diagnostics.AddError(
					"Control Version Not Found",
					fmt.Sprintf("Version %d of control %q does not exist.", data.Version.ValueInt64(), data.Identifier.ValueString()),
				)
				return
			}
			detail := fmt.Sprintf("Could not read version %d of control %q: %s", data.Version.ValueInt64(), data.Identifier.ValueString(), err.Error())
			if client.IsForbidden(err) {
				detail += controlBetaHint
			}
			resp.Diagnostics.AddError("Error Reading Control Version", detail)
			return
		}
		name = controlVersion.Name
		description = controlVersion.Description
		links = controlVersion.Links
		version = controlVersion.Version
		createdAt = controlVersion.CreatedAt
		createdBy = controlVersion.CreatedBy
		if controlVersion.Status != "" {
			data.Status = types.StringValue(controlVersion.Status)
		}
	}

	data.Identifier = types.StringValue(control.Identifier)
	data.Name = types.StringValue(name)
	// Unlike the resource mapper, "" always maps to null here: a data source
	// has no configured value to round-trip, so the consistency concern that
	// forces the resource to preserve an explicit "" does not apply.
	if description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(description)
	}

	// Normalize nil collections to empty so values are always known.
	if links == nil {
		links = map[string]string{}
	}
	linksValue, diags := types.MapValueFrom(ctx, types.StringType, links)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Links = linksValue

	data.Version = types.Int64Value(version)
	data.CreatedAt = timestampToState(createdAt)
	data.CreatedBy = types.StringValue(createdBy)

	tags := control.Tags
	if tags == nil {
		tags = map[string]string{}
	}
	tagsValue, diags := types.MapValueFrom(ctx, types.StringType, tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Tags = tagsValue

	data.Archived = types.BoolValue(control.Archived)

	policies := control.PoliciesReferencing
	if policies == nil {
		policies = []string{}
	}
	policiesValue, diags := types.ListValueFrom(ctx, types.StringType, policies)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.PoliciesReferencing = policiesValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
