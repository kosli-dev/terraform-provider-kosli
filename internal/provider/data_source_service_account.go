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
var _ datasource.DataSource = &serviceAccountDataSource{}

// NewServiceAccountDataSource creates a new service account data source.
func NewServiceAccountDataSource() datasource.DataSource {
	return &serviceAccountDataSource{}
}

// serviceAccountDataSource defines the data source implementation.
type serviceAccountDataSource struct {
	client *client.Client
}

// serviceAccountDataSourceModel describes the data source data model.
type serviceAccountDataSourceModel struct {
	Name           types.String  `tfsdk:"name"`
	Description    types.String  `tfsdk:"description"`
	Privilege      types.String  `tfsdk:"privilege"`
	DisplayName    types.String  `tfsdk:"display_name"`
	CreatingUserID types.String  `tfsdk:"creating_user_id"`
	CreatedAt      types.Float64 `tfsdk:"created_at"`
	ForWebhook     types.Bool    `tfsdk:"for_webhook"`
}

// Metadata returns the data source type name.
func (d *serviceAccountDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

// Schema defines the schema for the data source.
func (d *serviceAccountDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches details of an existing Kosli service account. Use this data source to reference service accounts and access metadata such as the privilege, creator, and creation timestamp.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the service account to query.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the service account.",
			},
			"privilege": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The privilege (role) of the service account within the organization (`admin`, `member`, `snapshotter`, or `reader`).",
			},
			"display_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The display name of the service account.",
			},
			"creating_user_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of the user who created the service account.",
			},
			"created_at": schema.Float64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix timestamp of when the service account was created.",
			},
			"for_webhook": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the service account was created for webhook usage.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *serviceAccountDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *serviceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serviceAccountDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	account, err := d.client.GetServiceAccount(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Service Account",
			fmt.Sprintf("Could not read service account %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	data.Name = types.StringValue(account.Name)
	if account.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(account.Description)
	}
	data.Privilege = types.StringValue(account.Privilege)
	data.DisplayName = types.StringValue(account.DisplayName)
	data.CreatingUserID = types.StringValue(account.CreatingUserID)
	data.CreatedAt = types.Float64Value(account.CreatedAt)
	data.ForWebhook = types.BoolValue(account.ForWebhook)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
