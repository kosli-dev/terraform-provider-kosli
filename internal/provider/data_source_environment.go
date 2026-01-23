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
var _ datasource.DataSource = &environmentDataSource{}

// NewEnvironmentDataSource creates a new environment data source.
func NewEnvironmentDataSource() datasource.DataSource {
	return &environmentDataSource{}
}

// environmentDataSource defines the data source implementation.
type environmentDataSource struct {
	client *client.Client
}

// environmentDataSourceModel describes the data source data model.
type environmentDataSourceModel struct {
	Name           types.String  `tfsdk:"name"`
	Type           types.String  `tfsdk:"type"`
	Description    types.String  `tfsdk:"description"`
	IncludeScaling types.Bool    `tfsdk:"include_scaling"`
	LastModifiedAt types.Float64 `tfsdk:"last_modified_at"`
	LastReportedAt types.Float64 `tfsdk:"last_reported_at"`
}

// Metadata returns the data source type name.
func (d *environmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

// Schema defines the schema for the data source.
func (d *environmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches details of an existing Kosli environment. Use this data source to reference environments and access metadata like last modified and last reported timestamps.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the environment to query.",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The environment type (e.g., K8S, ECS, S3, docker, server, lambda).",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the environment.",
			},
			"include_scaling": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the environment includes scaling events in snapshots.",
			},
			"last_modified_at": schema.Float64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix timestamp (with fractional seconds) of when the environment was last modified.",
			},
			"last_reported_at": schema.Float64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix timestamp (with fractional seconds) of when the environment was last reported. May be null if never reported.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *environmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *environmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data environmentDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get environment from API
	env, err := d.client.GetEnvironment(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Environment",
			fmt.Sprintf("Could not read environment %s: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to data source model
	data.Name = types.StringValue(env.Name)
	data.Type = types.StringValue(env.Type)

	// Handle empty description as null for consistency
	if env.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(env.Description)
	}

	data.IncludeScaling = types.BoolValue(env.IncludeScaling)
	data.LastModifiedAt = types.Float64Value(env.LastModifiedAt)

	// Handle nullable LastReportedAt
	if env.LastReportedAt == nil {
		data.LastReportedAt = types.Float64Null()
	} else {
		data.LastReportedAt = types.Float64Value(*env.LastReportedAt)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
