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
var _ resource.Resource = &serviceAccountResource{}
var _ resource.ResourceWithImportState = &serviceAccountResource{}

// NewServiceAccountResource creates a new service account resource.
func NewServiceAccountResource() resource.Resource {
	return &serviceAccountResource{}
}

// serviceAccountResource defines the resource implementation.
type serviceAccountResource struct {
	client *client.Client
}

// serviceAccountResourceModel describes the resource data model.
type serviceAccountResourceModel struct {
	Name           types.String  `tfsdk:"name"`
	Description    types.String  `tfsdk:"description"`
	Privilege      types.String  `tfsdk:"privilege"`
	DisplayName    types.String  `tfsdk:"display_name"`
	CreatingUserID types.String  `tfsdk:"creating_user_id"`
	CreatedAt      types.Float64 `tfsdk:"created_at"`
	ForWebhook     types.Bool    `tfsdk:"for_webhook"`
}

// Metadata returns the resource type name.
func (r *serviceAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

// Schema defines the schema for the resource.
func (r *serviceAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Kosli service account. Service accounts are non-human identities used to authenticate automation (such as CI/CD pipelines) against the Kosli API. API keys for a service account are managed separately via the `kosli_service_account_api_key` resource.\n\n" +
			"~> **Note:** Service accounts cannot be created in personal organizations. Only organization admins or the creating user can manage a service account.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the service account. Must be unique within the organization, contain only alphanumeric characters and hyphens (`^[a-zA-Z0-9\\-]+$`), and be at most 64 characters. Changing this will force recreation of the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Free-form description of the service account.",
				Optional:            true,
			},
			"privilege": schema.StringAttribute{
				MarkdownDescription: "Privilege (role) granted to the service account within the organization. Valid values: `admin`, `member`, `snapshotter`, `reader`. You can only create a service account with a privilege equal to or lower than your own.",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the service account, assigned by the server.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"creating_user_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the user who created the service account.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.Float64Attribute{
				MarkdownDescription: "Unix timestamp of when the service account was created.",
				Computed:            true,
			},
			"for_webhook": schema.BoolAttribute{
				MarkdownDescription: "Whether the service account was created for webhook usage.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *serviceAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *serviceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data serviceAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.CreateServiceAccountRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Privilege:   data.Privilege.ValueString(),
	}

	// The POST endpoint returns the full created service account.
	account, err := r.client.CreateServiceAccount(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Service Account",
			fmt.Sprintf("Could not create service account %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	mapServiceAccountToState(account, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *serviceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data serviceAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	account, err := r.client.GetServiceAccount(ctx, data.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Service account was deleted outside Terraform; remove from state
			// so Terraform can plan a recreation on the next apply.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Service Account",
			fmt.Sprintf("Could not read service account %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	mapServiceAccountToState(account, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *serviceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data serviceAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A null description in the plan clears the field server-side (PATCH sends
	// JSON null); a set value updates it.
	var description *string
	if !data.Description.IsNull() {
		v := data.Description.ValueString()
		description = &v
	}

	updateReq := &client.UpdateServiceAccountRequest{
		Description: description,
		Privilege:   data.Privilege.ValueString(),
	}

	account, err := r.client.UpdateServiceAccount(ctx, data.Name.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Service Account",
			fmt.Sprintf("Could not update service account %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	mapServiceAccountToState(account, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *serviceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data serviceAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteServiceAccount(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Service Account",
			fmt.Sprintf("Could not delete service account %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform state by name.
func (r *serviceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// mapServiceAccountToState maps an API response into the resource model.
func mapServiceAccountToState(account *client.ServiceAccount, data *serviceAccountResourceModel) {
	data.Name = types.StringValue(account.Name)
	// Treat an empty description as null to avoid drift when not configured.
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
}
