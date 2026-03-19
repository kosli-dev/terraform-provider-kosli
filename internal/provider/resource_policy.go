package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &policyResource{}
var _ resource.ResourceWithImportState = &policyResource{}
var _ resource.ResourceWithModifyPlan = &policyResource{}

// NewPolicyResource creates a new policy resource.
func NewPolicyResource() resource.Resource {
	return &policyResource{}
}

// policyResource defines the resource implementation.
type policyResource struct {
	client *client.Client
}

// policyResourceModel describes the resource data model.
type policyResourceModel struct {
	Name          types.String  `tfsdk:"name"`
	Description   types.String  `tfsdk:"description"`
	Content       types.String  `tfsdk:"content"`
	LatestVersion types.Int64   `tfsdk:"latest_version"`
	CreatedAt     types.Float64 `tfsdk:"created_at"`
}

// Metadata returns the resource type name.
func (r *policyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

// Schema defines the schema for the resource.
func (r *policyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Kosli policy. Policies define artifact compliance requirements " +
			"(provenance, trail-compliance, attestations) that can be attached to environments.\n\n" +
			"Policies are versioned and immutable: updating `content` or `description` creates a new " +
			"version rather than modifying the existing one.\n\n" +
			"~> **Note:** Deleting this resource removes it from Terraform state only. " +
			"Kosli has no API endpoint to delete policies, so the policy will remain in Kosli after `terraform destroy`. " +
			"To attach policies to environments, use the `kosli_policy_attachment` resource.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the policy. Must be unique within the organization. Changing this will force recreation of the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the policy.",
				Optional:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "YAML content of the policy, conforming to the Kosli policy schema " +
					"(`_schema: https://kosli.com/schemas/policy/environment/v1`). " +
					"Supports heredoc syntax for multi-line YAML. Updating this value creates a new policy version.",
				Required: true,
			},
			"latest_version": schema.Int64Attribute{
				MarkdownDescription: "The version number of the latest policy version. Null if the policy has no versions.",
				Computed:            true,
			},
			"created_at": schema.Float64Attribute{
				MarkdownDescription: "Unix timestamp of when the policy was first created.",
				Computed:            true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *policyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan marks latest_version as unknown when content is changing so Terraform
// doesn't flag an inconsistency when the API increments the version number.
func (r *policyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Only applies to updates (both state and plan present).
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Content.Equal(state.Content) || !plan.Description.Equal(state.Description) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("latest_version"), types.Int64Unknown())...)
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data policyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.CreatePolicyRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Content:     data.Content.ValueString(),
	}

	if err := r.client.CreatePolicy(ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Policy",
			fmt.Sprintf("Could not create policy %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// PUT returns "created"; GET to populate computed fields.
	policy, err := r.client.GetPolicy(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Policy After Creation",
			fmt.Sprintf("Could not read policy %q after creation: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	mapPolicyToModel(policy, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data policyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetPolicy(ctx, data.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Policy",
			fmt.Sprintf("Could not read policy %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	mapPolicyToModel(policy, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource. Updating a policy creates a new immutable version.
func (r *policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data policyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &client.CreatePolicyRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Content:     data.Content.ValueString(),
	}

	if err := r.client.CreatePolicy(ctx, updateReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Policy",
			fmt.Sprintf("Could not update policy %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	policy, err := r.client.GetPolicy(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Policy After Update",
			fmt.Sprintf("Could not read policy %q after update: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	mapPolicyToModel(policy, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the resource from Terraform state.
// Kosli has no API endpoint to delete policies, so the policy itself is not deleted.
func (r *policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No-op: Kosli does not support deleting policies via the API.
	// The resource is removed from Terraform state only.
}

// ImportState imports an existing policy by name.
// The content attribute is populated from the API response.
func (r *policyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// latestPolicyVersion returns the PolicyVersion with the highest Version number
// and true. Returns the zero value and false if versions is empty.
func latestPolicyVersion(versions []client.PolicyVersion) (client.PolicyVersion, bool) {
	if len(versions) == 0 {
		return client.PolicyVersion{}, false
	}
	latest := versions[0]
	for _, v := range versions[1:] {
		if v.Version > latest.Version {
			latest = v
		}
	}
	return latest, true
}

// mapPolicyToModel maps an API Policy response to the resource model.
func mapPolicyToModel(policy *client.Policy, data *policyResourceModel) {
	data.Name = types.StringValue(policy.Name)

	if policy.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(policy.Description)
	}

	data.CreatedAt = types.Float64Value(policy.CreatedAt)

	if latest, ok := latestPolicyVersion(policy.Versions); ok {
		data.LatestVersion = types.Int64Value(int64(latest.Version))
		data.Content = types.StringValue(latest.Content)
	} else {
		data.LatestVersion = types.Int64Null()
		data.Content = types.StringNull()
	}
}
