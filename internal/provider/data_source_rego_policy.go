package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &regoPolicyDataSource{}

// NewRegoPolicyDataSource creates a new rego policy data source.
func NewRegoPolicyDataSource() datasource.DataSource {
	return &regoPolicyDataSource{}
}

// regoPolicyDataSource defines the data source implementation.
type regoPolicyDataSource struct {
	client *client.Client
}

// regoPolicyDataSourceModel describes the data source data model.
type regoPolicyDataSourceModel struct {
	Identifier   types.String `tfsdk:"identifier"`
	Version      types.Int64  `tfsdk:"version"`
	Archived     types.Bool   `tfsdk:"archived"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	InputSchema  types.String `tfsdk:"input_schema"`
	Status       types.String `tfsdk:"status"`
	Comment      types.String `tfsdk:"comment"`
	Files        types.Map    `tfsdk:"files"`
	Digest       types.String `tfsdk:"digest"`
	Entrypoint   types.String `tfsdk:"entrypoint"`
	ParamsSchema types.String `tfsdk:"params_schema"`
	CreatedAt    types.String `tfsdk:"created_at"`
	CreatedBy    types.String `tfsdk:"created_by"`
}

// Metadata returns the data source type name.
func (d *regoPolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rego_policy"
}

// Schema defines the schema for the data source.
func (d *regoPolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches an existing Kosli rego policy and the files of one of its versions. " +
			"Without `version` it reads the policy's current version: the highest-numbered valid version, which is the one an evaluation runs.\n\n" +
			"~> **Note:** Rego policies are part of the Controls **beta** feature, which must be enabled for your organization; API requests return `403 Forbidden` otherwise.",

		Attributes: map[string]schema.Attribute{
			"identifier": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the policy to query. Kosli compares identifiers without regard to case.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(controlIdentifierRegexp, "must start with a letter or number and contain only letters, numbers, periods, hyphens, underscores, and tildes"),
				},
			},
			"version": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "The version to read. Any version can be read, whatever its status. " +
					"When unset, the current version is read and its number is returned here.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"archived": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Whether the policy is expected to be archived. Deleting a `kosli_rego_policy` archives it, so by default only an active policy is read and an archived one is an error. " +
					"Set to `true` to read an archived policy; an active one is then an error.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Human-readable display name of the policy.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the policy. Null when it has none.",
			},
			"input_schema": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Shape of the input the version is written against, such as `trail` or `trails`.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Validation status of the version: `pending`, `valid`, `rejected` or `failed`. Always `valid` when `version` is unset.",
			},
			"comment": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Why the version was published.",
			},
			"files": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The version's Rego bundle, as a map of each file's relative path to its source.",
			},
			"digest": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Digest of the version's files, as `sha256:` and a hex digest. A decision made with the version cites this value.",
			},
			"entrypoint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Policy package Kosli found as the entrypoint. Null unless the version is `valid`.",
			},
			"params_schema": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "JSON Schema, as a JSON string, that the version declares for its params. Null when it declares none or is not `valid`.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 UTC timestamp of when the policy was created.",
			},
			"created_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Display name of the user who created the policy.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *regoPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *regoPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx = dataSourceOperation(ctx, d)
	var data regoPolicyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(d.read(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// read fills the model from the policy and from the requested version, or the
// current version when none is requested.
func (d *regoPolicyDataSource) read(ctx context.Context, data *regoPolicyDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	identifier := data.Identifier.ValueString()

	policy, err := d.client.GetRegoPolicy(ctx, identifier)
	if err != nil {
		if client.IsNotFound(err) {
			diags.AddError("Rego Policy Not Found", fmt.Sprintf("Rego policy %q does not exist in the organization.", identifier))
			return diags
		}
		detail := fmt.Sprintf("Could not read rego policy %q: %s", identifier, err.Error())
		if client.IsForbidden(err) {
			detail += regoPolicyBetaHint
		}
		diags.AddError("Error Reading Rego Policy", detail)
		return diags
	}

	// Deleting a kosli_rego_policy archives it, so the archived argument
	// asserts the expected state, as it does on the kosli_control data source.
	if policy.Archived != data.Archived.ValueBool() {
		if policy.Archived {
			diags.AddError("Rego Policy Archived", fmt.Sprintf(
				"Rego policy %q exists but is archived. Set `archived = true` to read archived policies.", identifier))
		} else {
			diags.AddError("Rego Policy Not Archived", fmt.Sprintf(
				"Rego policy %q is not archived, but `archived = true` was set. Remove the argument to read active policies.", identifier))
		}
		return diags
	}

	versionNumber := data.Version.ValueInt64()
	if data.Version.IsNull() || data.Version.IsUnknown() {
		if policy.CurrentVersion == nil {
			diags.AddError("Rego Policy Has No Valid Version", fmt.Sprintf(
				"Rego policy %q has no valid version yet; its newest version %d is %s. Set `version = %d` to read that version.",
				identifier, policy.NewestVersion.Version, policy.NewestVersion.Status, policy.NewestVersion.Version))
			return diags
		}
		versionNumber = policy.CurrentVersion.Version
	}

	version, err := d.client.GetRegoPolicyVersion(ctx, identifier, versionNumber)
	if err != nil {
		if client.IsNotFound(err) {
			diags.AddError("Rego Policy Version Not Found", fmt.Sprintf(
				"Version %d of rego policy %q does not exist.", versionNumber, identifier))
			return diags
		}
		diags.AddError("Error Reading Rego Policy Version", fmt.Sprintf(
			"Could not read version %d of rego policy %q: %s", versionNumber, identifier, err.Error()))
		return diags
	}

	files, err := d.client.GetRegoPolicyBundle(ctx, identifier, versionNumber)
	if err != nil {
		diags.AddError("Error Reading Rego Policy Files", fmt.Sprintf(
			"Could not read the files of version %d of rego policy %q: %s", versionNumber, identifier, err.Error()))
		return diags
	}
	filesValue, d2 := types.MapValueFrom(ctx, types.StringType, files)
	diags.Append(d2...)
	if diags.HasError() {
		return diags
	}

	data.Archived = types.BoolValue(policy.Archived)
	data.Name = types.StringValue(policy.Name)
	data.Description = types.StringNull()
	if policy.Description != "" {
		data.Description = types.StringValue(policy.Description)
	}
	data.CreatedAt = timestampToState(policy.CreatedAt)
	data.CreatedBy = types.StringValue(policy.CreatedBy)

	data.Version = types.Int64Value(version.Version)
	data.Status = types.StringValue(version.Status)
	data.InputSchema = types.StringValue(version.InputSchema)
	data.Comment = types.StringValue(version.Comment)
	data.Digest = types.StringValue(version.Digest)
	data.Files = filesValue
	data.Entrypoint = types.StringNull()
	if version.Entrypoint != "" {
		data.Entrypoint = types.StringValue(version.Entrypoint)
	}
	data.ParamsSchema = regoParamsSchemaValue(version.ParamsSchema)

	return diags
}
