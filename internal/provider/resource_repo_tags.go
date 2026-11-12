package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &repoTagsResource{}
var _ resource.ResourceWithImportState = &repoTagsResource{}

// NewRepoTagsResource creates a new repo tags resource.
func NewRepoTagsResource() resource.Resource {
	return &repoTagsResource{}
}

// repoTagsResource manages the tags of a repo. Kosli records repos from
// attestations, so the repo itself is outside Terraform's control.
type repoTagsResource struct {
	client *client.Client
}

// repoTagsResourceModel describes the resource data model.
type repoTagsResourceModel struct {
	RepoID types.String `tfsdk:"repo_id"`
	Tags   types.Map    `tfsdk:"tags"`
}

// Metadata returns the resource type name.
func (r *repoTagsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repo_tags"
}

// Schema defines the schema for the resource.
func (r *repoTagsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the tags of a Kosli repo. Kosli records a repo when an attestation or trail references it, " +
			"so this resource tags an existing repo and never creates or deletes one. Use the `kosli_repos` data source to find repo IDs.\n\n" +
			"~> **Note:** `tags` is authoritative: it manages every tag on the repo. Tags added outside Terraform, in the Kosli UI or with `kosli tag repo`, " +
			"show as drift and are removed on the next apply. Destroying the resource removes all of the repo's tags.\n\n" +
			"~> **Note:** Declare at most one `kosli_repo_tags` per repo. Two resources for the same repo remove each other's tags.",

		Attributes: map[string]schema.Attribute{
			"repo_id": schema.StringAttribute{
				MarkdownDescription: "Internal Kosli ID of the repo, as returned by the `kosli_repos` data source or `kosli get repo`. " +
					"Repo names repeat across VCS providers, so tags are keyed by ID. Changing this forces a new resource.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Key-value pairs to tag the repo with. These replace every existing tag on the repo; `{}` removes them all.",
				Required:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *repoTagsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create applies the planned tags to the repo.
func (r *repoTagsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx = resourceOperation(ctx, r, opCreate)
	var data repoTagsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.replaceTags(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with every tag on the repo.
func (r *repoTagsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx = resourceOperation(ctx, r, opRead)
	var data repoTagsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repo, err := r.client.GetRepo(ctx, data.RepoID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Repo",
			fmt.Sprintf("Could not read repo %q: %s", data.RepoID.ValueString(), err.Error()),
		)
		return
	}

	data.Tags = repoTagsValue(ctx, repo.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update applies the planned tags to the repo.
func (r *repoTagsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx = resourceOperation(ctx, r, opUpdate)
	var data repoTagsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.replaceTags(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes every tag from the repo. The repo itself stays.
func (r *repoTagsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx = resourceOperation(ctx, r, opDelete)
	var data repoTagsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoID := data.RepoID.ValueString()
	repo, err := r.client.GetRepo(ctx, repoID)
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Repo",
			fmt.Sprintf("Could not read repo %q before removing its tags: %s", repoID, err.Error()),
		)
		return
	}

	current := repoTagsValue(ctx, repo.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	applyTags(ctx, r.client, repoID, "repo", current, types.MapValueMust(types.StringType, map[string]attr.Value{}), &resp.Diagnostics)
}

// ImportState imports a repo's tags by the repo's internal ID.
func (r *repoTagsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("repo_id"), req, resp)
}

// replaceTags makes the repo's tags equal data.Tags and stores the result in
// data. The diff is taken against the repo's live tags, not prior state, so
// tags added since the last refresh are removed too.
func (r *repoTagsResource) replaceTags(ctx context.Context, data *repoTagsResourceModel, diags *diag.Diagnostics) {
	repoID := data.RepoID.ValueString()

	repo, err := r.client.GetRepo(ctx, repoID)
	if err != nil {
		if client.IsNotFound(err) {
			diags.AddError(
				"Repo Not Found",
				fmt.Sprintf("No repo with ID %q exists in the organization. Kosli records a repo when an attestation or trail references it; "+
					"use the kosli_repos data source or `kosli get repo` to find repo IDs.", repoID),
			)
			return
		}
		diags.AddError(
			"Error Reading Repo",
			fmt.Sprintf("Could not read repo %q: %s", repoID, err.Error()),
		)
		return
	}

	current := repoTagsValue(ctx, repo.Tags, diags)
	if diags.HasError() {
		return
	}
	applyTags(ctx, r.client, repoID, "repo", current, data.Tags, diags)
	if diags.HasError() {
		return
	}

	repo, err = r.client.GetRepo(ctx, repoID)
	if err != nil {
		diags.AddError(
			"Error Reading Repo After Tagging",
			fmt.Sprintf("Could not read repo %q after updating its tags: %s", repoID, err.Error()),
		)
		return
	}
	data.Tags = repoTagsValue(ctx, repo.Tags, diags)
}
