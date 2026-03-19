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
var _ datasource.DataSource = &policyDataSource{}

// NewPolicyDataSource creates a new policy data source.
func NewPolicyDataSource() datasource.DataSource {
	return &policyDataSource{}
}

// policyDataSource defines the data source implementation.
type policyDataSource struct {
	client *client.Client
}

// policyDataSourceModel describes the data source data model.
type policyDataSourceModel struct {
	Name          types.String  `tfsdk:"name"`
	Description   types.String  `tfsdk:"description"`
	Content       types.String  `tfsdk:"content"`
	LatestVersion types.Int64   `tfsdk:"latest_version"`
	CreatedAt     types.Float64 `tfsdk:"created_at"`
}

// Metadata returns the data source type name.
func (d *policyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

// Schema defines the schema for the data source.
func (d *policyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches details of an existing Kosli policy.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the policy to query.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description of the policy.",
			},
			"content": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "YAML content of the latest policy version. Null if the policy has no versions.",
			},
			"latest_version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The version number of the latest policy version. Null if the policy has no versions.",
			},
			"created_at": schema.Float64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix timestamp of when the policy was first created.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *policyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *policyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data policyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := d.client.GetPolicy(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Policy",
			fmt.Sprintf("Could not read policy %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	data.Name = types.StringValue(policy.Name)

	if policy.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(policy.Description)
	}

	data.CreatedAt = types.Float64Value(policy.CreatedAt)

	if latest, ok := latestPolicyVersion(policy.Versions); ok {
		data.LatestVersion = types.Int64Value(int64(latest.Version))
		data.Content = types.StringValue(latest.Content)
	} else {
		data.LatestVersion = types.Int64Null()
		data.Content = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
