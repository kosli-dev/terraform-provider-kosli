package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &environmentResource{}
var _ resource.ResourceWithImportState = &environmentResource{}

// NewEnvironmentResource creates a new environment resource.
func NewEnvironmentResource() resource.Resource {
	return &environmentResource{}
}

// environmentResource defines the resource implementation.
type environmentResource struct {
	client *client.Client
}

// environmentResourceModel describes the resource data model.
type environmentResourceModel struct {
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	Description    types.String `tfsdk:"description"`
	IncludeScaling types.Bool   `tfsdk:"include_scaling"`
}

// Metadata returns the resource type name.
func (r *environmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

// Schema defines the schema for the resource.
func (r *environmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Kosli environment. Environments represent deployment targets where artifacts are deployed. Supports physical environment types: K8S, ECS, S3, docker, server, and lambda.\n\n" +
			"~> **Note:** This resource manages the environment configuration only. Environment tags are managed through a separate Kosli API. " +
			"Environment policies will be available in a future release. " +
			"For querying environment metadata such as `last_modified_at`, `last_reported_at`, and `archived` status, use the `kosli_environment` data source.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the environment. Must be unique within the organization. Changing this will force recreation of the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of the environment. Valid values: `K8S`, `ECS`, `S3`, `docker`, `server`, `lambda`. Changing this will force recreation of the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the environment. Explains the purpose and characteristics of this deployment target.",
				Optional:            true,
			},
			"include_scaling": schema.BoolAttribute{
				MarkdownDescription: "Whether to include scaling information when reporting environment snapshots. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *environmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *environmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data environmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create API request
	createReq := &client.CreateEnvironmentRequest{
		Name:           data.Name.ValueString(),
		Type:           data.Type.ValueString(),
		Description:    data.Description.ValueString(),
		IncludeScaling: data.IncludeScaling.ValueBool(),
	}

	// Call API to create the environment
	if err := r.client.CreateEnvironment(ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Environment",
			fmt.Sprintf("Could not create environment %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Per ADR 002: PUT returns "OK", so we must GET to populate state
	env, err := r.client.GetEnvironment(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Environment After Creation",
			fmt.Sprintf("Could not read environment %q after creation: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state (configuration fields only, no timestamps)
	// Handle empty description as null to avoid inconsistency when not provided in config
	if env.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(env.Description)
	}
	data.IncludeScaling = types.BoolValue(env.IncludeScaling)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *environmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data environmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state from API
	env, err := r.client.GetEnvironment(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Environment",
			fmt.Sprintf("Could not read environment %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state (configuration fields only, no timestamps)
	// Handle empty description as null to avoid inconsistency when not provided in config
	if env.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(env.Description)
	}
	data.IncludeScaling = types.BoolValue(env.IncludeScaling)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
// Per the API behavior, PUT is idempotent and updates the environment.
func (r *environmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data environmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create API request (PUT is idempotent)
	createReq := &client.CreateEnvironmentRequest{
		Name:           data.Name.ValueString(),
		Type:           data.Type.ValueString(),
		Description:    data.Description.ValueString(),
		IncludeScaling: data.IncludeScaling.ValueBool(),
	}

	// Call API to update the environment
	if err := r.client.CreateEnvironment(ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Environment",
			fmt.Sprintf("Could not update environment %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// GET to populate state
	env, err := r.client.GetEnvironment(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Environment After Update",
			fmt.Sprintf("Could not read environment %q after update: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state (configuration fields only, no timestamps)
	// Handle empty description as null to avoid inconsistency when not provided in config
	if env.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(env.Description)
	}
	data.IncludeScaling = types.BoolValue(env.IncludeScaling)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
// Per the API behavior, this archives the environment (soft delete).
func (r *environmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data environmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Archive the environment
	if err := r.client.ArchiveEnvironment(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Environment",
			fmt.Sprintf("Could not archive environment %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// State is automatically removed by the framework
}

// ImportState imports an existing resource into Terraform state.
func (r *environmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by name
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
