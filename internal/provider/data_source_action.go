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
var _ datasource.DataSource = &actionDataSource{}

// NewActionDataSource creates a new action data source.
func NewActionDataSource() datasource.DataSource {
	return &actionDataSource{}
}

// actionDataSource defines the data source implementation.
type actionDataSource struct {
	client *client.Client
}

// actionDataSourceModel describes the data source data model.
type actionDataSourceModel struct {
	Name           types.String  `tfsdk:"name"`
	Environments   types.List    `tfsdk:"environments"`
	Triggers       types.List    `tfsdk:"triggers"`
	Number         types.Int64   `tfsdk:"number"`
	CreatedBy      types.String  `tfsdk:"created_by"`
	LastModifiedAt types.Float64 `tfsdk:"last_modified_at"`
}

// Metadata returns the data source type name.
func (d *actionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action"
}

// Schema defines the schema for the data source.
func (d *actionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches details of an existing Kosli action.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the action to query.",
			},
			"environments": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "List of environment names this action monitors.",
			},
			"triggers": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "List of trigger event types that activate this action.",
			},
			"number": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Server-assigned numeric identifier for the action.",
			},
			"created_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User who created the action.",
			},
			"last_modified_at": schema.Float64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix timestamp (with fractional seconds) of when the action was last modified.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *actionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *actionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data actionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	action, err := d.client.GetActionByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Action",
			fmt.Sprintf("Could not read action %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	data.Name = types.StringValue(action.Name)
	data.Number = types.Int64Value(int64(action.Number))
	data.CreatedBy = types.StringValue(action.CreatedBy)
	data.LastModifiedAt = types.Float64Value(action.LastModifiedAt)

	environments := action.Environments
	if environments == nil {
		environments = []string{}
	}
	envList, diags := types.ListValueFrom(ctx, types.StringType, environments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Environments = envList

	triggers := action.Triggers
	if triggers == nil {
		triggers = []string{}
	}
	trigList, diags := types.ListValueFrom(ctx, types.StringType, triggers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Triggers = trigList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
