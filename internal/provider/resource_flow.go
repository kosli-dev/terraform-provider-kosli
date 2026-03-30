package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &flowResource{}
var _ resource.ResourceWithImportState = &flowResource{}

// NewFlowResource creates a new flow resource.
func NewFlowResource() resource.Resource {
	return &flowResource{}
}

// flowResource defines the resource implementation.
type flowResource struct {
	client *client.Client
}

// flowResourceModel describes the resource data model.
type flowResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Visibility  types.String `tfsdk:"visibility"`
	Template    types.String `tfsdk:"template"`
}

// Metadata returns the resource type name.
func (r *flowResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flow"
}

// Schema defines the schema for the resource.
func (r *flowResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Kosli flow. A Kosli Flow represents a business or software process that requires change tracking. It allows you to monitor changes across all steps within a process or focus specifically on a subset of critical steps.\n\n" +
			"~> **Note:** The `template` attribute accepts a YAML string defining the flow template structure. " +
			"You can load it from a file using the `file()` function: `template = file(\"template.yml\")`. " +
			"Minor YAML formatting differences between what you provide and what the API returns may result in a no-op change being shown in plans.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the flow. Must be unique within the organization. Changing this will force recreation of the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the flow. Explains the purpose and context of this pipeline.",
				Optional:            true,
			},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "Visibility of the flow. Valid values: `public`, `private`. Defaults to `private`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("private"),
				Validators: []validator.String{
					stringvalidator.OneOf("public", "private"),
				},
			},
			"template": schema.StringAttribute{
				MarkdownDescription: "YAML template defining the flow structure (trails, artifacts, attestations). " +
					"Can be provided as an inline heredoc or loaded from a file using `file()`. " +
					"If omitted, the flow is created without a template.",
				Optional: true,
				Computed: true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *flowResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *flowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data flowResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request
	createReq := &client.CreateFlowRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Visibility:  data.Visibility.ValueString(),
		Template:    data.Template.ValueString(),
	}

	// Call API to create the flow
	if err := r.client.CreateFlow(ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Flow",
			fmt.Sprintf("Could not create flow %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Per ADR 002: PUT returns "OK", so we must GET to populate state
	flow, err := r.client.GetFlow(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Flow After Creation",
			fmt.Sprintf("Could not read flow %q after creation: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state
	mapFlowToModel(flow, &data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *flowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data flowResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state from API
	flow, err := r.client.GetFlow(ctx, data.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Flow was deleted outside Terraform; remove from state so Terraform
			// can plan a recreation on the next apply.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Flow",
			fmt.Sprintf("Could not read flow %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state
	mapFlowToModel(flow, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *flowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data flowResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request (PUT is idempotent)
	updateReq := &client.CreateFlowRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Visibility:  data.Visibility.ValueString(),
		Template:    data.Template.ValueString(),
	}

	// Call API to update the flow
	if err := r.client.CreateFlow(ctx, updateReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Flow",
			fmt.Sprintf("Could not update flow %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// GET to populate state
	flow, err := r.client.GetFlow(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Flow After Update",
			fmt.Sprintf("Could not read flow %q after update: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state
	mapFlowToModel(flow, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
// Per the API behavior, this archives the flow (soft delete).
func (r *flowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data flowResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Archive the flow
	if err := r.client.ArchiveFlow(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Flow",
			fmt.Sprintf("Could not archive flow %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// State is automatically removed by the framework
}

// ImportState imports an existing resource into Terraform state.
func (r *flowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by name
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// mapFlowToModel maps a Flow API response to the Terraform resource model.
func mapFlowToModel(flow *client.Flow, data *flowResourceModel) {
	data.Name = types.StringValue(flow.Name)

	// Handle empty description as null to avoid inconsistency when not provided in config
	if flow.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(flow.Description)
	}

	// Visibility always has a value from the API
	if flow.Visibility == "" {
		data.Visibility = types.StringValue("private")
	} else {
		data.Visibility = types.StringValue(flow.Visibility)
	}

	// Handle empty template as null
	if flow.Template == "" {
		data.Template = types.StringNull()
	} else {
		data.Template = types.StringValue(flow.Template)
	}
}
