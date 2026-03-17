package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &actionResource{}
var _ resource.ResourceWithImportState = &actionResource{}

// NewActionResource creates a new action resource.
func NewActionResource() resource.Resource {
	return &actionResource{}
}

// actionResource defines the resource implementation.
type actionResource struct {
	client *client.Client
}

// actionResourceModel describes the resource data model.
type actionResourceModel struct {
	Name           types.String  `tfsdk:"name"`
	Environments   types.List    `tfsdk:"environments"`
	Triggers       types.List    `tfsdk:"triggers"`
	WebhookURL     types.String  `tfsdk:"webhook_url"`
	PayloadVersion types.String  `tfsdk:"payload_version"`
	Number         types.Int64   `tfsdk:"number"`
	CreatedBy      types.String  `tfsdk:"created_by"`
	LastModifiedAt types.Float64 `tfsdk:"last_modified_at"`
}

// Metadata returns the resource type name.
func (r *actionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action"
}

// Schema defines the schema for the resource.
func (r *actionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Kosli action. Actions define webhook notifications triggered by environment compliance events.\n\n" +
			"~> **Note:** Actions are identified internally by a server-assigned `number`. The `name` is used during import to look up the number.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the action. Must be unique within the organization. Changing this will force recreation of the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environments": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of environment names this action monitors.",
				Required:            true,
			},
			"triggers": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of trigger event types that activate this action (e.g. `ON_NON_COMPLIANT_ENV`, `ON_COMPLIANT_ENV`).",
				Required:            true,
			},
			"webhook_url": schema.StringAttribute{
				MarkdownDescription: "Webhook URL to send notifications to.",
				Required:            true,
				Sensitive:           true,
			},
			"payload_version": schema.StringAttribute{
				MarkdownDescription: "Webhook payload version. Defaults to `1.0`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("1.0"),
			},
			"number": schema.Int64Attribute{
				MarkdownDescription: "Server-assigned numeric identifier for the action.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "User who created the action.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_modified_at": schema.Float64Attribute{
				MarkdownDescription: "Unix timestamp of when the action was last modified.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *actionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// Create creates the resource and sets the initial Terraform state.
func (r *actionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data actionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionReq, diags := buildActionRequest(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.CreateOrUpdateAction(ctx, actionReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Action",
			fmt.Sprintf("Could not create action %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// PUT returns "OK"; the API has no direct GET-by-name endpoint, so we do a
	// full list scan via GetActionByName to populate state (including number).
	action, err := r.client.GetActionByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Action After Creation",
			fmt.Sprintf("Could not read action %q after creation: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(mapActionResponseToModel(ctx, action, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *actionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data actionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	action, err := r.client.GetActionByNumber(ctx, int(data.Number.ValueInt64()))
	if err != nil {
		if client.IsNotFound(err) {
			// Action was deleted outside Terraform; remove from state so Terraform
			// can plan a recreation on the next apply.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Action",
			fmt.Sprintf("Could not read action number %d: %s", data.Number.ValueInt64(), err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(mapActionResponseToModel(ctx, action, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *actionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data actionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Carry number from state since plan won't have it for updates
	var state actionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Number = state.Number

	actionReq, diags := buildActionRequest(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use the numbered PUT endpoint to update in-place. The base PUT endpoint
	// creates a new action for non-Slack actions, which would change the number.
	if err := r.client.UpdateAction(ctx, int(data.Number.ValueInt64()), actionReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Action",
			fmt.Sprintf("Could not update action %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	action, err := r.client.GetActionByNumber(ctx, int(data.Number.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Action After Update",
			fmt.Sprintf("Could not read action number %d after update: %s", data.Number.ValueInt64(), err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(mapActionResponseToModel(ctx, action, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *actionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data actionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAction(ctx, int(data.Number.ValueInt64())); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Action",
			fmt.Sprintf("Could not delete action number %d: %s", data.Number.ValueInt64(), err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform state by name.
// ImportStatePassthroughID cannot be used here because the resource tracks state
// by the server-assigned number, not the name. We use a full list scan via
// GetActionByName (the API has no direct GET-by-name endpoint) to resolve the number.
func (r *actionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	action, err := r.client.GetActionByName(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Action",
			fmt.Sprintf("Could not find action named %q: %s", req.ID, err.Error()),
		)
		return
	}

	var data actionResourceModel
	resp.Diagnostics.Append(mapActionResponseToModel(ctx, action, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// buildActionRequest constructs an ActionRequest from the resource model.
func buildActionRequest(ctx context.Context, data *actionResourceModel) (*client.ActionRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	var environments []string
	diags.Append(data.Environments.ElementsAs(ctx, &environments, false)...)
	if diags.HasError() {
		return nil, diags
	}

	var triggers []string
	diags.Append(data.Triggers.ElementsAs(ctx, &triggers, false)...)
	if diags.HasError() {
		return nil, diags
	}

	return &client.ActionRequest{
		Name: data.Name.ValueString(),
		// Type is always "env" — the only action type currently supported by the Kosli API.
		Type:         "env",
		Environments: environments,
		Triggers:     triggers,
		Targets: []client.ActionTarget{
			{
				Type:           "WEBHOOK",
				Webhook:        data.WebhookURL.ValueString(),
				PayloadVersion: data.PayloadVersion.ValueString(),
			},
		},
	}, diags
}

// mapActionResponseToModel maps an API response into the resource model.
func mapActionResponseToModel(ctx context.Context, action *client.ActionResponse, data *actionResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.Name = types.StringValue(action.Name)
	data.Number = types.Int64Value(int64(action.Number))
	data.CreatedBy = types.StringValue(action.CreatedBy)
	data.LastModifiedAt = types.Float64Value(action.LastModifiedAt)

	environments := action.Environments
	if environments == nil {
		environments = []string{}
	}
	envList, d := types.ListValueFrom(ctx, types.StringType, environments)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	data.Environments = envList

	triggers := action.Triggers
	if triggers == nil {
		triggers = []string{}
	}
	trigList, d := types.ListValueFrom(ctx, types.StringType, triggers)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	data.Triggers = trigList

	// Extract webhook_url and payload_version from the first WEBHOOK target.
	// The API does not echo back sensitive fields (webhook URL) on GET responses,
	// so only overwrite state when the API returns a non-empty value. This preserves
	// the value the user configured and prevents a permanent plan diff on every refresh.
	if len(action.Targets) > 0 {
		if action.Targets[0].Webhook != "" {
			data.WebhookURL = types.StringValue(action.Targets[0].Webhook)
		}
		if action.Targets[0].PayloadVersion != "" {
			data.PayloadVersion = types.StringValue(action.Targets[0].PayloadVersion)
		}
	}

	return diags
}
