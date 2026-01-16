package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &customAttestationTypeDataSource{}

// NewCustomAttestationTypeDataSource creates a new custom attestation type data source.
func NewCustomAttestationTypeDataSource() datasource.DataSource {
	return &customAttestationTypeDataSource{}
}

// customAttestationTypeDataSource defines the data source implementation.
type customAttestationTypeDataSource struct {
	client *client.Client
}

// customAttestationTypeDataSourceModel describes the data source data model.
type customAttestationTypeDataSourceModel struct {
	Name        types.String         `tfsdk:"name"`
	Description types.String         `tfsdk:"description"`
	Schema      jsontypes.Normalized `tfsdk:"schema"`
	JqRules     types.List           `tfsdk:"jq_rules"`
	Archived    types.Bool           `tfsdk:"archived"`
}

// Metadata returns the data source type name.
func (d *customAttestationTypeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_attestation_type"
}

// Schema defines the schema for the data source.
func (d *customAttestationTypeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches details of an existing custom attestation type from Kosli. Custom attestation types define how Kosli validates and evaluates evidence from proprietary tools, custom metrics, or specialized compliance requirements.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the custom attestation type. Must start with a letter or number and contain only letters, numbers, periods, hyphens, underscores, and tildes.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "A description of what this attestation type validates.",
			},
			"schema": schema.StringAttribute{
				Computed:            true,
				CustomType:          jsontypes.NormalizedType{},
				MarkdownDescription: "JSON Schema that defines the structure of attestation data.",
			},
			"jq_rules": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of jq expressions that define evaluation rules. All rules must evaluate to `true` for compliance.",
			},
			"archived": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this attestation type has been archived.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *customAttestationTypeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *customAttestationTypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data customAttestationTypeDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get attestation type from API
	attestationType, err := d.client.GetCustomAttestationType(ctx, data.Name.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Custom Attestation Type",
			fmt.Sprintf("Could not read custom attestation type %s: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map response to model
	data.Name = types.StringValue(attestationType.Name)
	data.Description = types.StringValue(attestationType.Description)
	data.Schema = jsontypes.NewNormalizedValue(attestationType.Schema)
	data.Archived = types.BoolValue(attestationType.Archived)

	// Convert jq_rules (API client already transformed from evaluator format)
	jqRules := make([]types.String, 0, len(attestationType.JqRules))
	for _, rule := range attestationType.JqRules {
		jqRules = append(jqRules, types.StringValue(rule))
	}

	jqRulesList, diags := types.ListValueFrom(ctx, types.StringType, jqRules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.JqRules = jqRulesList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
