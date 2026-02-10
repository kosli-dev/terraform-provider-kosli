package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &logicalEnvironmentDataSource{}

// NewLogicalEnvironmentDataSource creates a new logical environment data source.
func NewLogicalEnvironmentDataSource() datasource.DataSource {
	return &logicalEnvironmentDataSource{}
}

// logicalEnvironmentDataSource defines the data source implementation.
type logicalEnvironmentDataSource struct {
	client *client.Client
}

// logicalEnvironmentDataSourceModel describes the data source data model.
type logicalEnvironmentDataSourceModel struct {
	Name                 types.String  `tfsdk:"name"`
	Type                 types.String  `tfsdk:"type"`
	Description          types.String  `tfsdk:"description"`
	IncludedEnvironments types.List    `tfsdk:"included_environments"`
	LastModifiedAt       types.Float64 `tfsdk:"last_modified_at"`
}

// Metadata returns the data source type name.
func (d *logicalEnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logical_environment"
}

// Schema defines the schema for the data source.
func (d *logicalEnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches details of an existing Kosli logical environment. Use this data source to reference logical environments and access their aggregated physical environments.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the logical environment to query.",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The environment type (always `logical` for logical environments).",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the logical environment.",
			},
			"included_environments": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "List of physical environment names aggregated by this logical environment.",
			},
			"last_modified_at": schema.Float64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix timestamp (with fractional seconds) of when the logical environment was last modified.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *logicalEnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *logicalEnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data logicalEnvironmentDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get environment from API
	env, err := d.client.GetEnvironment(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Logical Environment",
			fmt.Sprintf("Could not read logical environment %s: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Ensure the environment is a logical environment
	if env.Type != "logical" {
		resp.Diagnostics.AddError(
			"Invalid Environment Type",
			fmt.Sprintf(
				"Environment %s is of type %q, but this data source only supports logical environments.",
				data.Name.ValueString(),
				env.Type,
			),
		)
		return
	}

	// Map API response to data source model
	data.Name = types.StringValue(env.Name)
	data.Type = types.StringValue("logical")

	// Handle empty description as null for consistency
	if env.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(env.Description)
	}

	// Convert included_environments from API response to types.List
	// Normalize nil to empty slice to ensure consistent state (empty list vs null)
	includedEnvs := env.IncludedEnvironments
	if includedEnvs == nil {
		includedEnvs = []string{}
	}
	includedEnvsList, diags := types.ListValueFrom(ctx, types.StringType, includedEnvs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.IncludedEnvironments = includedEnvsList

	data.LastModifiedAt = types.Float64Value(env.LastModifiedAt)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
