package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &logicalEnvironmentResource{}
var _ resource.ResourceWithImportState = &logicalEnvironmentResource{}

// NewLogicalEnvironmentResource creates a new logical environment resource.
func NewLogicalEnvironmentResource() resource.Resource {
	return &logicalEnvironmentResource{}
}

// logicalEnvironmentResource defines the resource implementation.
type logicalEnvironmentResource struct {
	client *client.Client
}

// logicalEnvironmentResourceModel describes the resource data model.
type logicalEnvironmentResourceModel struct {
	Name                 types.String `tfsdk:"name"`
	Type                 types.String `tfsdk:"type"`
	Description          types.String `tfsdk:"description"`
	IncludedEnvironments types.List   `tfsdk:"included_environments"`
}

// Metadata returns the resource type name.
func (r *logicalEnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logical_environment"
}

// Schema defines the schema for the resource.
func (r *logicalEnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Kosli logical environment. Logical environments aggregate multiple physical environments for organizational purposes.\n\n" +
			"~> **Important:** Logical environments can ONLY contain physical environments (K8S, ECS, S3, docker, server, lambda), not other logical environments. " +
			"Attempting to include a logical environment will result in an error from the Kosli API. See [ADR-004](https://github.com/kosli-dev/terraform-provider-kosli/blob/main/adrs/004-logical-environment-validation.md) for validation strategy.\n\n" +
			"~> **Note:** This resource manages logical environment configuration only. For querying environment metadata such as `last_modified_at` and `archived` status, use the `kosli_logical_environment` data source.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the logical environment. Must be unique within the organization. Changing this will force recreation of the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of the environment. Always set to `logical` (computed by provider, not user-configurable).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the logical environment. Explains the purpose and aggregation strategy.",
				Optional:            true,
			},
			"included_environments": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of physical environment names to aggregate. Only physical environments are allowed (K8S, ECS, S3, docker, server, lambda). Can be empty.",
				Required:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *logicalEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *logicalEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data logicalEnvironmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract included_environments from types.List to []string
	var includedEnvironments []string
	resp.Diagnostics.Append(data.IncludedEnvironments.ElementsAs(ctx, &includedEnvironments, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create API request
	createReq := &client.CreateEnvironmentRequest{
		Name:                 data.Name.ValueString(),
		Type:                 "logical",
		Description:          data.Description.ValueString(),
		IncludedEnvironments: includedEnvironments,
	}

	// Call API to create the environment
	// Per ADR-004, validation is performed by the API, not client-side
	if err := r.client.CreateEnvironment(ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Logical Environment",
			fmt.Sprintf("Could not create logical environment %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Per ADR 002: PUT returns "OK", so we must GET to populate state
	// WORKAROUND: Preserve included_environments from plan because API doesn't return it
	preservedIncludedEnvs := data.IncludedEnvironments

	env, err := r.client.GetEnvironment(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Logical Environment After Creation",
			fmt.Sprintf("Could not read logical environment %q after creation: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state
	data.Type = types.StringValue(env.Type)
	// Handle empty description as null to avoid inconsistency when not provided in config
	if env.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(env.Description)
	}

	// WORKAROUND: Kosli API doesn't return included_environments in GET responses
	// Keep the value from plan instead of trying to read from API
	data.IncludedEnvironments = preservedIncludedEnvs

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *logicalEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data logicalEnvironmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the current included_environments from state (API limitation workaround)
	// The Kosli API doesn't return included_environments in GET responses
	preservedIncludedEnvs := data.IncludedEnvironments

	// Get current state from API
	env, err := r.client.GetEnvironment(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Logical Environment",
			fmt.Sprintf("Could not read logical environment %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state
	data.Type = types.StringValue(env.Type)
	// Handle empty description as null to avoid inconsistency when not provided in config
	if env.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(env.Description)
	}

	// WORKAROUND: Kosli API doesn't return included_environments in GET responses
	// Keep the value from current state instead of trying to read from API
	data.IncludedEnvironments = preservedIncludedEnvs

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
// Per the API behavior, PUT is idempotent and updates the environment.
func (r *logicalEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data logicalEnvironmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract included_environments from types.List to []string
	var includedEnvironments []string
	resp.Diagnostics.Append(data.IncludedEnvironments.ElementsAs(ctx, &includedEnvironments, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create API request (PUT is idempotent)
	createReq := &client.CreateEnvironmentRequest{
		Name:                 data.Name.ValueString(),
		Type:                 "logical",
		Description:          data.Description.ValueString(),
		IncludedEnvironments: includedEnvironments,
	}

	// Call API to update the environment
	if err := r.client.CreateEnvironment(ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Logical Environment",
			fmt.Sprintf("Could not update logical environment %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// GET to populate state
	// WORKAROUND: Preserve included_environments from plan because API doesn't return it
	preservedIncludedEnvs := data.IncludedEnvironments

	env, err := r.client.GetEnvironment(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Logical Environment After Update",
			fmt.Sprintf("Could not read logical environment %q after update: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state
	data.Type = types.StringValue(env.Type)
	// Handle empty description as null to avoid inconsistency when not provided in config
	if env.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(env.Description)
	}

	// WORKAROUND: Kosli API doesn't return included_environments in GET responses
	// Keep the value from plan instead of trying to read from API
	data.IncludedEnvironments = preservedIncludedEnvs

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
// Per the API behavior, this archives the environment (soft delete).
func (r *logicalEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data logicalEnvironmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Archive the environment
	if err := r.client.ArchiveEnvironment(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Logical Environment",
			fmt.Sprintf("Could not archive logical environment %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// State is automatically removed by the framework
}

// ImportState imports an existing resource into Terraform state.
func (r *logicalEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by name
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
