package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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

// controlIdentifierRegexp mirrors the standard Kosli naming rule: must start
// with a letter or number and contain only letters, numbers, periods,
// hyphens, underscores, and tildes.
var controlIdentifierRegexp = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._~-]*$`)

// controlBetaHint is appended to API error diagnostics when the server
// returns 403: Controls is a beta feature gated per organization.
const controlBetaHint = " Note: Controls is a beta feature; a 403 response may mean it is not enabled for your organization."

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &controlResource{}
var _ resource.ResourceWithImportState = &controlResource{}

// NewControlResource creates a new control resource.
func NewControlResource() resource.Resource {
	return &controlResource{}
}

// controlResource defines the resource implementation.
type controlResource struct {
	client *client.Client
}

// controlResourceModel describes the resource data model.
type controlResourceModel struct {
	Identifier          types.String `tfsdk:"identifier"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Links               types.Map    `tfsdk:"links"`
	Version             types.Int64  `tfsdk:"version"`
	CreatedAt           types.String `tfsdk:"created_at"`
	CreatedBy           types.String `tfsdk:"created_by"`
	Tags                types.Map    `tfsdk:"tags"`
	PoliciesReferencing types.List   `tfsdk:"policies_referencing"`
}

// Metadata returns the resource type name.
func (r *controlResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_control"
}

// Schema defines the schema for the resource.
func (r *controlResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Kosli control. Controls are org-level definitions of SDLC requirements (for example `SDLC-001` \"Binary provenance\") whose compliance is evaluated from attestations and enforced through environment policies.\n\n" +
			"~> **Note:** Controls is a **beta** feature and must be enabled for your organization; API requests return `403 Forbidden` otherwise.\n\n" +
			"-> Deleting this resource **archives** the control in Kosli rather than hard-deleting it. Creating a new control with the identifier of an archived control fails with a conflict; unarchive the control in Kosli and import it instead.",

		Attributes: map[string]schema.Attribute{
			"identifier": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the control within the organization (e.g. `SDLC-001`). Must start with a letter or number and contain only letters, numbers, periods (`.`), hyphens (`-`), underscores (`_`), and tildes (`~`). Changing this will force recreation of the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(controlIdentifierRegexp, "must start with a letter or number and contain only letters, numbers, periods, hyphens, underscores, and tildes"),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable display name of the control (e.g. `Binary provenance`). Can be changed without recreating the control.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Free-form description of the control.",
				Optional:            true,
			},
			"links": schema.MapAttribute{
				MarkdownDescription: "Named links related to the control (e.g. documentation or runbook URLs), as a map of link name to URL.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"version": schema.Int64Attribute{
				MarkdownDescription: "Version number of the control, assigned by the server and incremented on every update.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC3339 UTC timestamp of when the control was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Identifier of the user who created the control.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags on the control, as a map of tag key to value. Tags are managed in Kosli and cannot be set via this resource.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"policies_referencing": schema.ListAttribute{
				MarkdownDescription: "Names of the environment policies that reference this control.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *controlResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *controlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data controlResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	links := linksFromState(ctx, data.Links, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.CreateControlRequest{
		Identifier:  data.Identifier.ValueString(),
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Links:       links,
	}

	// The POST endpoint returns the full created control.
	control, err := r.client.CreateControl(ctx, createReq)
	if err != nil {
		detail := fmt.Sprintf("Could not create control %q: %s", data.Identifier.ValueString(), err.Error())
		switch {
		case client.IsConflict(err):
			detail += " A control with this identifier already exists — possibly archived. Unarchive it in Kosli and use `terraform import` instead."
		case client.IsForbidden(err):
			detail += controlBetaHint
		}
		resp.Diagnostics.AddError("Error Creating Control", detail)
		return
	}

	mapControlToState(ctx, control, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *controlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data controlResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	control, err := r.client.GetControl(ctx, data.Identifier.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Control was deleted outside Terraform; remove from state so
			// Terraform can plan a recreation on the next apply.
			resp.State.RemoveResource(ctx)
			return
		}
		detail := fmt.Sprintf("Could not read control %q: %s", data.Identifier.ValueString(), err.Error())
		if client.IsForbidden(err) {
			detail += controlBetaHint
		}
		resp.Diagnostics.AddError("Error Reading Control", detail)
		return
	}

	// Archiving is this resource's delete mechanism, but archived controls are
	// still returned by the API with 200. Treat an out-of-band archive as a
	// deletion so drift is detected and a recreation is planned.
	if control.Archived {
		resp.State.RemoveResource(ctx)
		return
	}

	mapControlToState(ctx, control, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *controlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data controlResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	links := linksFromState(ctx, data.Links, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// PUT replaces the mutable fields wholesale: an unset description or links
	// is sent as its empty value, which clears the field server-side.
	updateReq := &client.UpdateControlRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Links:       links,
	}

	control, err := r.client.UpdateControl(ctx, data.Identifier.ValueString(), updateReq)
	if err != nil {
		detail := fmt.Sprintf("Could not update control %q: %s", data.Identifier.ValueString(), err.Error())
		if client.IsForbidden(err) {
			detail += controlBetaHint
		}
		resp.Diagnostics.AddError("Error Updating Control", detail)
		return
	}

	mapControlToState(ctx, control, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete archives the control and removes the Terraform state on success.
// The API has no hard delete; archiving is the delete mechanism.
func (r *controlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data controlResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.ArchiveControl(ctx, data.Identifier.ValueString()); err != nil {
		// Already gone (e.g. out-of-band); deletion is idempotent.
		if client.IsNotFound(err) {
			return
		}
		detail := fmt.Sprintf("Could not archive control %q: %s", data.Identifier.ValueString(), err.Error())
		if client.IsForbidden(err) {
			detail += controlBetaHint
		}
		resp.Diagnostics.AddError("Error Deleting Control", detail)
		return
	}
}

// ImportState imports an existing resource into Terraform state by identifier.
func (r *controlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("identifier"), req, resp)
}

// linksFromState converts the links map attribute to the API representation.
// A null or unknown attribute yields a nil map.
func linksFromState(ctx context.Context, links types.Map, diags *diag.Diagnostics) map[string]string {
	if links.IsNull() || links.IsUnknown() {
		return nil
	}
	var result map[string]string
	diags.Append(links.ElementsAs(ctx, &result, false)...)
	return result
}

// mapControlToState maps an API response into the resource model.
func mapControlToState(ctx context.Context, control *client.Control, data *controlResourceModel, diags *diag.Diagnostics) {
	data.Identifier = types.StringValue(control.Identifier)
	data.Name = types.StringValue(control.Name)
	// Map an empty API description to null so an unset attribute doesn't
	// drift — but preserve a known empty string so an explicitly configured
	// description = "" round-trips as-is instead of triggering a "Provider
	// produced inconsistent result after apply" error.
	switch {
	case control.Description != "":
		data.Description = types.StringValue(control.Description)
	case !data.Description.IsNull() && !data.Description.IsUnknown() && data.Description.ValueString() == "":
		data.Description = types.StringValue("")
	default:
		data.Description = types.StringNull()
	}

	// Same normalization for links: empty from the API maps to null unless an
	// empty map was explicitly configured.
	switch {
	case len(control.Links) > 0:
		linksValue, d := types.MapValueFrom(ctx, types.StringType, control.Links)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Links = linksValue
	case !data.Links.IsNull() && !data.Links.IsUnknown():
		emptyLinks, d := types.MapValueFrom(ctx, types.StringType, map[string]string{})
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Links = emptyLinks
	default:
		data.Links = types.MapNull(types.StringType)
	}

	data.Version = types.Int64Value(control.Version)
	data.CreatedAt = timestampToState(control.CreatedAt)
	data.CreatedBy = types.StringValue(control.CreatedBy)

	// Computed collections: normalize nil to empty so values are always known.
	tags := control.Tags
	if tags == nil {
		tags = map[string]string{}
	}
	tagsValue, d := types.MapValueFrom(ctx, types.StringType, tags)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	data.Tags = tagsValue

	policies := control.PoliciesReferencing
	if policies == nil {
		policies = []string{}
	}
	policiesValue, d := types.ListValueFrom(ctx, types.StringType, policies)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	data.PoliciesReferencing = policiesValue
}
