package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &serviceAccountAPIKeyResource{}
var _ resource.ResourceWithImportState = &serviceAccountAPIKeyResource{}

// NewServiceAccountAPIKeyResource creates a new service account API key resource.
func NewServiceAccountAPIKeyResource() resource.Resource {
	return &serviceAccountAPIKeyResource{}
}

// serviceAccountAPIKeyResource defines the resource implementation.
type serviceAccountAPIKeyResource struct {
	client *client.Client
}

// serviceAccountAPIKeyResourceModel describes the resource data model.
type serviceAccountAPIKeyResourceModel struct {
	ServiceAccountName types.String  `tfsdk:"service_account_name"`
	Description        types.String  `tfsdk:"description"`
	ExpiresAt          types.Int64   `tfsdk:"expires_at"`
	ID                 types.String  `tfsdk:"id"`
	Key                types.String  `tfsdk:"key"`
	CreatedAt          types.Float64 `tfsdk:"created_at"`
	LastUsedAt         types.Float64 `tfsdk:"last_used_at"`
}

// Metadata returns the resource type name.
func (r *serviceAccountAPIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_api_key"
}

// Schema defines the schema for the resource.
func (r *serviceAccountAPIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an API key for a Kosli service account. The raw key value is returned only once, at creation time, and is stored in Terraform state as a sensitive value — it can never be retrieved from the API again.\n\n" +
			"~> **Note:** API keys are immutable. Changing any argument forces the key to be revoked and a new one created. On `terraform import`, the `key` attribute cannot be populated because the raw value is not retrievable.",

		Attributes: map[string]schema.Attribute{
			"service_account_name": schema.StringAttribute{
				MarkdownDescription: "Name of the service account this API key belongs to. Changing this forces creation of a new key.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the API key (at least one character). Changing this forces creation of a new key.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"expires_at": schema.Int64Attribute{
				MarkdownDescription: "Unix timestamp (seconds) at which the key expires. Omit (or set to `0`) for a key that never expires. Must not be in the past. Changing this forces creation of a new key.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplaceIfConfigured(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier of the API key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The raw API key value. Only available at creation time and stored as a sensitive value. Empty when the resource is imported.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.Float64Attribute{
				MarkdownDescription: "Unix timestamp of when the API key was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"last_used_at": schema.Float64Attribute{
				MarkdownDescription: "Unix timestamp of when the API key was last used. `0` if never used.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *serviceAccountAPIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *serviceAccountAPIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data serviceAccountAPIKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.CreateAPIKeyRequest{
		Description: data.Description.ValueString(),
		ExpiresAt:   data.ExpiresAt.ValueInt64(),
	}

	key, err := r.client.CreateServiceAccountAPIKey(ctx, data.ServiceAccountName.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Service Account API Key",
			fmt.Sprintf("Could not create API key for service account %q: %s", data.ServiceAccountName.ValueString(), err.Error()),
		)
		return
	}

	// The create response is the only time the raw key value is returned.
	data.Key = types.StringValue(key.Key)
	mapAPIKeyToState(key, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *serviceAccountAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data serviceAccountAPIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.GetServiceAccountAPIKeyByID(ctx, data.ServiceAccountName.ValueString(), data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Key was revoked outside Terraform; remove from state so Terraform
			// can plan a recreation on the next apply.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Service Account API Key",
			fmt.Sprintf("Could not read API key %q for service account %q: %s", data.ID.ValueString(), data.ServiceAccountName.ValueString(), err.Error()),
		)
		return
	}

	// The raw key is never returned by the list endpoint, so data.Key is
	// intentionally left untouched (preserved from prior state).
	mapAPIKeyToState(key, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is required by the resource interface but is effectively a no-op:
// every configurable attribute forces replacement, so Update is never invoked
// in practice. It carries the plan through to state defensively.
func (r *serviceAccountAPIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceAccountAPIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state serviceAccountAPIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve computed values that are only known from prior state.
	plan.ID = state.ID
	plan.Key = state.Key
	plan.CreatedAt = state.CreatedAt
	plan.LastUsedAt = state.LastUsedAt

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete revokes the API key.
func (r *serviceAccountAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data serviceAccountAPIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.RevokeServiceAccountAPIKey(ctx, data.ServiceAccountName.ValueString(), data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error Revoking Service Account API Key",
			fmt.Sprintf("Could not revoke API key %q for service account %q: %s", data.ID.ValueString(), data.ServiceAccountName.ValueString(), err.Error()),
		)
		return
	}
}

// ImportState imports an existing API key using the composite ID
// "service_account_name/key_id". The raw key value cannot be recovered.
func (r *serviceAccountAPIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in the format \"service_account_name/key_id\", got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_account_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// mapAPIKeyToState maps the non-secret fields of an API response into the model.
// It never sets the Key field, which is only available at creation time.
func mapAPIKeyToState(key *client.ServiceAccountAPIKey, data *serviceAccountAPIKeyResourceModel) {
	data.ID = types.StringValue(key.ID)
	// description is a required argument (min 1 char), so it is always present.
	data.Description = types.StringValue(key.Description)
	data.ExpiresAt = types.Int64Value(key.ExpiresAt)
	data.CreatedAt = types.Float64Value(key.CreatedAt)
	data.LastUsedAt = types.Float64Value(key.LastUsedAt)
}
