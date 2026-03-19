package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &policyAttachmentResource{}
var _ resource.ResourceWithImportState = &policyAttachmentResource{}

// NewPolicyAttachmentResource creates a new policy attachment resource.
func NewPolicyAttachmentResource() resource.Resource {
	return &policyAttachmentResource{}
}

// policyAttachmentResource defines the resource implementation.
type policyAttachmentResource struct {
	client *client.Client
}

// policyAttachmentResourceModel describes the resource data model.
type policyAttachmentResourceModel struct {
	EnvironmentName types.String `tfsdk:"environment_name"`
	PolicyName      types.String `tfsdk:"policy_name"`
}

// Metadata returns the resource type name.
func (r *policyAttachmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_attachment"
}

// Schema defines the schema for the resource.
func (r *policyAttachmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Attaches a Kosli policy to an environment (physical or logical). " +
			"When this resource is destroyed, the policy is detached from the environment.\n\n" +
			"Both `environment_name` and `policy_name` are immutable: changing either attribute " +
			"will destroy the existing attachment and create a new one.\n\n" +
			"**Import:** Use `terraform import kosli_policy_attachment.<name> <environment_name>/<policy_name>`.",

		Attributes: map[string]schema.Attribute{
			"environment_name": schema.StringAttribute{
				MarkdownDescription: "Name of the environment to attach the policy to. Changing this will force recreation of the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy_name": schema.StringAttribute{
				MarkdownDescription: "Name of the policy to attach. Changing this will force recreation of the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *policyAttachmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create attaches the policy to the environment.
func (r *policyAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data policyAttachmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	environmentName := data.EnvironmentName.ValueString()
	policyName := data.PolicyName.ValueString()

	if err := r.client.AttachPolicy(ctx, environmentName, policyName); err != nil {
		resp.Diagnostics.AddError(
			"Error Attaching Policy",
			fmt.Sprintf("Could not attach policy %q to environment %q: %s", policyName, environmentName, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read verifies the policy attachment still exists.
func (r *policyAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data policyAttachmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	environmentName := data.EnvironmentName.ValueString()
	policyName := data.PolicyName.ValueString()

	policies, err := r.client.GetEnvironmentPolicies(ctx, environmentName)
	if err != nil {
		if client.IsNotFound(err) {
			// Environment no longer exists; remove attachment from state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Policy Attachment",
			fmt.Sprintf("Could not read policies for environment %q: %s", environmentName, err.Error()),
		)
		return
	}

	// Check if the specific policy is still attached.
	found := false
	for _, p := range policies {
		if p.Name == policyName {
			found = true
			break
		}
	}

	if !found {
		// Policy was detached externally; remove from state.
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is not used because both attributes require replacement.
func (r *policyAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Both attributes are ForceNew, so Update is never called.
}

// Delete detaches the policy from the environment.
func (r *policyAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data policyAttachmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	environmentName := data.EnvironmentName.ValueString()
	policyName := data.PolicyName.ValueString()

	if err := r.client.DetachPolicy(ctx, environmentName, policyName); err != nil {
		if client.IsNotFound(err) {
			// Already gone; nothing to do.
			return
		}
		resp.Diagnostics.AddError(
			"Error Detaching Policy",
			fmt.Sprintf("Could not detach policy %q from environment %q: %s", policyName, environmentName, err.Error()),
		)
		return
	}
}

// ImportState imports an existing policy attachment by its composite ID.
// The expected import ID format is: {environment_name}/{policy_name}
func (r *policyAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'environment_name/policy_name', got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_name"), parts[1])...)
}
