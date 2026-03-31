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
var _ datasource.DataSource = &flowDataSource{}

// NewFlowDataSource creates a new flow data source.
func NewFlowDataSource() datasource.DataSource {
	return &flowDataSource{}
}

// flowDataSource defines the data source implementation.
type flowDataSource struct {
	client *client.Client
}

// flowDataSourceModel describes the data source data model.
type flowDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Template    types.String `tfsdk:"template"`
}

// Metadata returns the data source type name.
func (d *flowDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flow"
}

// Schema defines the schema for the data source.
func (d *flowDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches details of an existing Kosli flow. A Kosli Flow represents a business or software process that requires change tracking.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the flow to query.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the flow.",
			},
			"template": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "YAML template defining the flow structure (trails, artifacts, attestations).",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *flowDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *flowDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data flowDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get flow from API
	flow, err := d.client.GetFlow(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Flow",
			fmt.Sprintf("Could not read flow %s: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to data source model
	data.Name = types.StringValue(flow.Name)

	if flow.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(flow.Description)
	}

	if flow.Template == "" {
		data.Template = types.StringNull()
	} else {
		data.Template = types.StringValue(flow.Template)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
