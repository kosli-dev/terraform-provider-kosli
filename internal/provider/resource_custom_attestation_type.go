package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &customAttestationTypeResource{}
var _ resource.ResourceWithImportState = &customAttestationTypeResource{}

// NewCustomAttestationTypeResource creates a new custom attestation type resource.
func NewCustomAttestationTypeResource() resource.Resource {
	return &customAttestationTypeResource{}
}

// customAttestationTypeResource defines the resource implementation.
type customAttestationTypeResource struct {
	client *client.Client
}

// customAttestationTypeResourceModel describes the resource data model.
type customAttestationTypeResourceModel struct {
	Name        types.String         `tfsdk:"name"`
	Description types.String         `tfsdk:"description"`
	Schema      jsontypes.Normalized `tfsdk:"schema"`
	JqRules     types.List           `tfsdk:"jq_rules"`
}

// Metadata returns the resource type name.
func (r *customAttestationTypeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_attestation_type"
}

// Schema defines the schema for the resource.
func (r *customAttestationTypeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom attestation type in Kosli. Custom attestation types define how Kosli validates and evaluates evidence from proprietary tools, custom metrics, or specialized compliance requirements.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the custom attestation type. Must start with a letter or number and can only contain letters, numbers, periods, hyphens, underscores, and tildes. Changing this will force recreation of the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the custom attestation type. Explains what this attestation type validates.",
				Optional:            true,
			},
			"schema": schema.StringAttribute{
				MarkdownDescription: "JSON Schema definition that defines the structure of attestation data. Can be provided inline using heredoc syntax or loaded from a file using `file()`. Semantic equality is used for comparison, so formatting differences are ignored.",
				Required:            true,
				CustomType:          jsontypes.NormalizedType{},
			},
			"jq_rules": schema.ListAttribute{
				MarkdownDescription: "List of jq evaluation rules. Each rule is a jq expression that must evaluate to true for the attestation to be considered compliant. Example: `[\".coverage >= 80\"]`",
				Required:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *customAttestationTypeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *customAttestationTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data customAttestationTypeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract jq_rules from the list
	var jqRules []string
	resp.Diagnostics.Append(data.JqRules.ElementsAs(ctx, &jqRules, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create API request
	// jsontypes.Normalized handles semantic equality, so we send schema as-is
	createReq := &client.CreateCustomAttestationTypeRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Schema:      data.Schema.ValueString(),
		JqRules:     jqRules,
	}

	// Call API to create the custom attestation type
	if err := r.client.CreateCustomAttestationType(ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Custom Attestation Type",
			fmt.Sprintf("Could not create custom attestation type %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Per ADR 002: POST returns "OK", so we must GET to populate state
	attestationType, err := r.client.GetCustomAttestationType(ctx, data.Name.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Custom Attestation Type After Creation",
			fmt.Sprintf("Could not read custom attestation type %q after creation: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state
	data.Description = types.StringValue(attestationType.Description)
	data.Schema = jsontypes.NewNormalizedValue(attestationType.Schema)

	// Convert jq_rules back to list
	jqRulesList, diags := types.ListValueFrom(ctx, types.StringType, attestationType.JqRules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.JqRules = jqRulesList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *customAttestationTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data customAttestationTypeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state from API
	attestationType, err := r.client.GetCustomAttestationType(ctx, data.Name.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Custom Attestation Type",
			fmt.Sprintf("Could not read custom attestation type %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state
	data.Description = types.StringValue(attestationType.Description)
	data.Schema = jsontypes.NewNormalizedValue(attestationType.Schema)

	// Convert jq_rules back to list
	jqRulesList, diags := types.ListValueFrom(ctx, types.StringType, attestationType.JqRules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.JqRules = jqRulesList

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
// Per the API behavior, updates create a new version of the attestation type.
func (r *customAttestationTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data customAttestationTypeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract jq_rules from the list
	var jqRules []string
	resp.Diagnostics.Append(data.JqRules.ElementsAs(ctx, &jqRules, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create API request (updates create a new version)
	// jsontypes.Normalized handles semantic equality, so we send schema as-is
	createReq := &client.CreateCustomAttestationTypeRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Schema:      data.Schema.ValueString(),
		JqRules:     jqRules,
	}

	// Call API to create new version
	if err := r.client.CreateCustomAttestationType(ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Custom Attestation Type",
			fmt.Sprintf("Could not update custom attestation type %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// GET to populate state with new version
	attestationType, err := r.client.GetCustomAttestationType(ctx, data.Name.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Custom Attestation Type After Update",
			fmt.Sprintf("Could not read custom attestation type %q after update: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state
	data.Description = types.StringValue(attestationType.Description)
	data.Schema = jsontypes.NewNormalizedValue(attestationType.Schema)

	// Convert jq_rules back to list
	jqRulesList, diags := types.ListValueFrom(ctx, types.StringType, attestationType.JqRules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.JqRules = jqRulesList

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
// Per the API behavior, this archives the attestation type (soft delete).
func (r *customAttestationTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data customAttestationTypeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Archive the custom attestation type
	if err := r.client.ArchiveCustomAttestationType(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Custom Attestation Type",
			fmt.Sprintf("Could not archive custom attestation type %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// State is automatically removed by the framework
}

// ImportState imports an existing resource into Terraform state.
func (r *customAttestationTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by name
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
