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
var _ datasource.DataSource = &serviceAccountAPIKeysDataSource{}

// NewServiceAccountAPIKeysDataSource creates a new service account API keys data source.
func NewServiceAccountAPIKeysDataSource() datasource.DataSource {
	return &serviceAccountAPIKeysDataSource{}
}

// serviceAccountAPIKeysDataSource defines the data source implementation.
type serviceAccountAPIKeysDataSource struct {
	client *client.Client
}

// serviceAccountAPIKeysDataSourceModel describes the data source data model.
type serviceAccountAPIKeysDataSourceModel struct {
	ServiceAccountName types.String                       `tfsdk:"service_account_name"`
	Keys               []serviceAccountAPIKeyElementModel `tfsdk:"keys"`
}

// serviceAccountAPIKeyElementModel is a single API key entry (metadata only;
// the raw key value is never returned by the list endpoint).
type serviceAccountAPIKeyElementModel struct {
	ID          types.String  `tfsdk:"id"`
	Description types.String  `tfsdk:"description"`
	CreatedAt   types.Float64 `tfsdk:"created_at"`
	ExpiresAt   types.Int64   `tfsdk:"expires_at"`
	LastUsedAt  types.Float64 `tfsdk:"last_used_at"`
}

// Metadata returns the data source type name.
func (d *serviceAccountAPIKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_api_keys"
}

// Schema defines the schema for the data source.
func (d *serviceAccountAPIKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the active API keys for a Kosli service account. Only key metadata is returned — the raw key value is never retrievable after creation. Useful for auditing keys and building rotation logic based on `expires_at` and `last_used_at`.",

		Attributes: map[string]schema.Attribute{
			"service_account_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the service account whose API keys should be listed.",
			},
			"keys": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The list of active API keys for the service account.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Server-assigned identifier of the API key.",
						},
						"description": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Description of the API key.",
						},
						"created_at": schema.Float64Attribute{
							Computed:            true,
							MarkdownDescription: "Unix timestamp of when the API key was created.",
						},
						"expires_at": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Unix timestamp (seconds) at which the key expires. `0` if the key never expires.",
						},
						"last_used_at": schema.Float64Attribute{
							Computed:            true,
							MarkdownDescription: "Unix timestamp of when the API key was last used. `0` if never used.",
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *serviceAccountAPIKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *serviceAccountAPIKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serviceAccountAPIKeysDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keys, err := d.client.ListServiceAccountAPIKeys(ctx, data.ServiceAccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Service Account API Keys",
			fmt.Sprintf("Could not list API keys for service account %q: %s", data.ServiceAccountName.ValueString(), err.Error()),
		)
		return
	}

	data.Keys = make([]serviceAccountAPIKeyElementModel, 0, len(keys))
	for _, k := range keys {
		data.Keys = append(data.Keys, serviceAccountAPIKeyElementModel{
			ID:          types.StringValue(k.ID),
			Description: types.StringValue(k.Description),
			CreatedAt:   types.Float64Value(k.CreatedAt),
			ExpiresAt:   types.Int64Value(int64(k.ExpiresAt)),
			LastUsedAt:  types.Float64Value(k.LastUsedAt),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
