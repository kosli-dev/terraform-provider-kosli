package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
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

const (
	// regoPolicyPollInterval is the wait between two reads of a pending version.
	regoPolicyPollInterval = 2 * time.Second
	// regoPolicyPollTimeout is how long an apply waits for Kosli to validate
	// a published version.
	regoPolicyPollTimeout = 5 * time.Minute

	// regoPolicyBetaHint is appended to API error diagnostics when the server
	// returns 403: rego policies share the Controls beta flag.
	regoPolicyBetaHint = " Note: rego policies are part of the Controls beta feature; a 403 response may mean it is not enabled for your organization."
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &regoPolicyResource{}
var _ resource.ResourceWithImportState = &regoPolicyResource{}
var _ resource.ResourceWithModifyPlan = &regoPolicyResource{}

// NewRegoPolicyResource creates a new rego policy resource.
func NewRegoPolicyResource() resource.Resource {
	return &regoPolicyResource{
		pollInterval: regoPolicyPollInterval,
		pollTimeout:  regoPolicyPollTimeout,
	}
}

// regoPolicyResource defines the resource implementation.
type regoPolicyResource struct {
	client       *client.Client
	pollInterval time.Duration
	pollTimeout  time.Duration
}

// regoPolicyResourceModel describes the resource data model.
type regoPolicyResourceModel struct {
	Identifier   types.String `tfsdk:"identifier"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	InputSchema  types.String `tfsdk:"input_schema"`
	Comment      types.String `tfsdk:"comment"`
	Files        types.Map    `tfsdk:"files"`
	Version      types.Int64  `tfsdk:"version"`
	Digest       types.String `tfsdk:"digest"`
	Entrypoint   types.String `tfsdk:"entrypoint"`
	ParamsSchema types.String `tfsdk:"params_schema"`
	CreatedAt    types.String `tfsdk:"created_at"`
	CreatedBy    types.String `tfsdk:"created_by"`
}

// Metadata returns the resource type name.
func (r *regoPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rego_policy"
}

// Schema defines the schema for the resource.
func (r *regoPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Kosli rego policy: a versioned bundle of Rego files that Kosli evaluates trails against. " +
			"Each change to `files` or `input_schema` publishes a new version, and the apply waits until Kosli has validated it.\n\n" +
			"~> **Note:** Rego policies are part of the Controls **beta** feature, which must be enabled for your organization; API requests return `403 Forbidden` otherwise.\n\n" +
			"~> **Note:** Publishing requires a service account with **Admin** permissions.\n\n" +
			"-> Kosli has no way to delete a rego policy or one of its versions. Destroying this resource only removes it from the Terraform state; the policy and every published version stay in Kosli.",

		Attributes: map[string]schema.Attribute{
			"identifier": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the policy within the organization. Must start with a letter or number and contain only letters, numbers, periods (`.`), hyphens (`-`), underscores (`_`), and tildes (`~`). " +
					"Kosli compares identifiers without regard to case. Changing this to a different policy will force recreation of the resource.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
							resp.RequiresReplace = !strings.EqualFold(req.StateValue.ValueString(), req.PlanValue.ValueString())
						},
						"A change of case only addresses the same policy.",
						"A change of case only addresses the same policy.",
					),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(controlIdentifierRegexp, "must start with a letter or number and contain only letters, numbers, periods, hyphens, underscores, and tildes"),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable display name of the policy. Can be changed without publishing a new version.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Free-form description of the policy. Can be changed without publishing a new version.",
				Optional:            true,
			},
			"input_schema": schema.StringAttribute{
				MarkdownDescription: "Shape of the input the policy is written against. Known values: `trail` (the policy evaluates one trail) and `trails` (a set of trails). The Kosli API validates the value.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"comment": schema.StringAttribute{
				MarkdownDescription: "Why this version is published. Kosli stores the comment with the version an apply creates; " +
					"changing only the comment publishes nothing, and the value is not read back from Kosli.",
				Optional: true,
			},
			"files": schema.MapAttribute{
				MarkdownDescription: "The Rego bundle, as a map of each file's relative path to its source. A single-file policy is one entry. " +
					"Kosli limits a bundle to 100 files and 1 MiB in total, and rejects paths that leave the bundle.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
			},
			"version": schema.Int64Attribute{
				MarkdownDescription: "Number of the policy's current version: the highest-numbered valid version, which is the one an evaluation runs.",
				Computed:            true,
			},
			"digest": schema.StringAttribute{
				MarkdownDescription: "Digest of the current version's files, as `sha256:` and a hex digest. A decision made with the policy cites this value.",
				Computed:            true,
			},
			"entrypoint": schema.StringAttribute{
				MarkdownDescription: "Policy package Kosli found as the entrypoint of the current version.",
				Computed:            true,
			},
			"params_schema": schema.StringAttribute{
				MarkdownDescription: "JSON Schema, as a JSON string, that the current version declares for its params. Null when the policy declares none.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC3339 UTC timestamp of when the policy was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Display name of the user who created the policy.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *regoPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan keeps the current version's attributes known when an update
// publishes no new version, so a change of name alone does not show them as
// "known after apply".
func (r *regoPolicyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state regoPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keepRegoPolicyVersionAttributes(&plan, &state)
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// keepRegoPolicyVersionAttributes copies the version attributes from state
// into plan when the plan publishes the bundle already in state.
func keepRegoPolicyVersionAttributes(plan, state *regoPolicyResourceModel) {
	if state.Version.IsNull() || !plan.Files.Equal(state.Files) || !plan.InputSchema.Equal(state.InputSchema) {
		return
	}
	plan.Version = state.Version
	plan.Digest = state.Digest
	plan.Entrypoint = state.Entrypoint
	plan.ParamsSchema = state.ParamsSchema
}

// Create publishes the first version and sets the initial Terraform state.
func (r *regoPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx = resourceOperation(ctx, r, opCreate)
	var data regoPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.publish(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the policy's current version.
func (r *regoPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx = resourceOperation(ctx, r, opRead)
	var data regoPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, diags := r.refresh(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update publishes the planned bundle and metadata. The API stores a new
// version only when the files or the input schema changed.
func (r *regoPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx = resourceOperation(ctx, r, opUpdate)
	var data regoPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.publish(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete only removes the resource from state: the API cannot delete a rego
// policy or its versions.
func (r *regoPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data regoPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning(
		"Rego Policy Left In Kosli",
		fmt.Sprintf("Kosli cannot delete a rego policy. Policy %q and its versions remain in Kosli; only the Terraform state entry was removed.", data.Identifier.ValueString()),
	)
}

// ImportState imports an existing policy by identifier. The Read that follows
// downloads the current version's files into state.
func (r *regoPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("identifier"), req, resp)
}

// publish sends the model's bundle and metadata, waits for the returned
// version to be validated, and writes the outcome into the model. A version
// that is rejected, failed or still pending at the timeout is an error.
func (r *regoPolicyResource) publish(ctx context.Context, data *regoPolicyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	identifier := data.Identifier.ValueString()

	var files map[string]string
	diags.Append(data.Files.ElementsAs(ctx, &files, false)...)
	if diags.HasError() {
		return diags
	}

	publishReq := &client.PublishRegoPolicyRequest{
		Name:        data.Name.ValueString(),
		InputSchema: data.InputSchema.ValueString(),
		Comment:     data.Comment.ValueString(),
		Files:       files,
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		description := data.Description.ValueString()
		publishReq.Description = &description
	}

	version, err := r.client.PublishRegoPolicy(ctx, identifier, publishReq)
	if err != nil {
		detail := fmt.Sprintf("Could not publish rego policy %q: %s", identifier, err.Error())
		if client.IsForbidden(err) {
			detail += regoPolicyBetaHint
		}
		diags.AddError("Error Publishing Rego Policy", detail)
		return diags
	}

	version, err = r.waitForValidation(ctx, identifier, version)
	if err != nil {
		diags.AddError("Error Reading Rego Policy Version", fmt.Sprintf(
			"Could not read version %d of rego policy %q while waiting for its validation: %s", version.Version, identifier, err.Error()))
		return diags
	}

	switch version.Status {
	case client.RegoPolicyVersionStatusValid:
	case client.RegoPolicyVersionStatusRejected:
		diags.AddError("Rego Policy Version Rejected", fmt.Sprintf(
			"Kosli rejected version %d of rego policy %q%s. The policy keeps running its previous valid version, if it has one. Fix the files and apply again.",
			version.Version, identifier, regoPolicyValidationDetail(version)))
		return diags
	case client.RegoPolicyVersionStatusFailed:
		diags.AddError("Rego Policy Version Not Validated", fmt.Sprintf(
			"Kosli could not validate version %d of rego policy %q%s. The policy is not at fault; apply again to publish the files for another validation.",
			version.Version, identifier, regoPolicyValidationDetail(version)))
		return diags
	default:
		diags.AddError("Rego Policy Version Still Pending", fmt.Sprintf(
			"Version %d of rego policy %q is still pending after %s (status %q). Apply again to keep waiting; the version is not published twice.",
			version.Version, identifier, r.pollTimeout, version.Status))
		return diags
	}

	policy, err := r.client.GetRegoPolicy(ctx, identifier)
	if err != nil {
		diags.AddError("Error Reading Rego Policy After Publish", fmt.Sprintf(
			"Could not read rego policy %q after publishing version %d: %s", identifier, version.Version, err.Error()))
		return diags
	}

	mapRegoPolicyToModel(policy, data)
	mapRegoPolicyVersionToModel(version, data)
	return diags
}

// waitForValidation reads the version until it leaves pending or the timeout
// passes, and returns the last version read.
func (r *regoPolicyResource) waitForValidation(ctx context.Context, identifier string, version *client.RegoPolicyVersion) (*client.RegoPolicyVersion, error) {
	deadline := time.Now().Add(r.pollTimeout)

	for version.Status == client.RegoPolicyVersionStatusPending && time.Now().Before(deadline) {
		timer := time.NewTimer(r.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return version, ctx.Err()
		case <-timer.C:
		}

		next, err := r.client.GetRegoPolicyVersion(ctx, identifier, version.Version)
		if err != nil {
			return version, err
		}
		version = next
	}

	return version, nil
}

// regoPolicyValidationDetail renders a version's validation error, or "" when
// the API gave none.
func regoPolicyValidationDetail(version *client.RegoPolicyVersion) string {
	if version.Error == nil {
		return ""
	}
	return fmt.Sprintf(" (%s: %s)", version.Error.Kind, version.Error.Message)
}

// refresh writes the policy and its current version into the model, and
// reports whether the policy still exists.
//
// State follows the current version, which is the newest valid one. A newer
// version that was rejected or is still pending is therefore not reflected:
// the configuration that published it keeps showing a diff until an apply
// ends with a valid version.
func (r *regoPolicyResource) refresh(ctx context.Context, data *regoPolicyResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	identifier := data.Identifier.ValueString()

	policy, err := r.client.GetRegoPolicy(ctx, identifier)
	if err != nil {
		if client.IsNotFound(err) {
			return false, diags
		}
		detail := fmt.Sprintf("Could not read rego policy %q: %s", identifier, err.Error())
		if client.IsForbidden(err) {
			detail += regoPolicyBetaHint
		}
		diags.AddError("Error Reading Rego Policy", detail)
		return false, diags
	}

	mapRegoPolicyToModel(policy, data)

	if policy.CurrentVersion == nil {
		data.Files = types.MapNull(types.StringType)
		data.Version = types.Int64Null()
		data.Digest = types.StringNull()
		data.Entrypoint = types.StringNull()
		data.ParamsSchema = types.StringNull()
		return true, diags
	}

	version, err := r.client.GetRegoPolicyVersion(ctx, identifier, policy.CurrentVersion.Version)
	if err != nil {
		diags.AddError("Error Reading Rego Policy Version", fmt.Sprintf(
			"Could not read version %d of rego policy %q: %s", policy.CurrentVersion.Version, identifier, err.Error()))
		return false, diags
	}

	var stateFiles map[string]string
	if !data.Files.IsNull() && !data.Files.IsUnknown() {
		diags.Append(data.Files.ElementsAs(ctx, &stateFiles, false)...)
		if diags.HasError() {
			return false, diags
		}
	}

	// The files are downloaded only when the digest says they differ from
	// the ones in state: a bundle can hold 100 files, each its own request.
	if stateFiles == nil || client.RegoPolicyDigest(stateFiles) != version.Digest {
		files, err := r.downloadFiles(ctx, identifier, version.Version)
		if err != nil {
			diags.AddError("Error Reading Rego Policy Files", fmt.Sprintf(
				"Could not read the files of version %d of rego policy %q: %s", version.Version, identifier, err.Error()))
			return false, diags
		}
		filesValue, d := types.MapValueFrom(ctx, types.StringType, files)
		diags.Append(d...)
		if diags.HasError() {
			return false, diags
		}
		data.Files = filesValue
	}

	mapRegoPolicyVersionToModel(version, data)
	return true, diags
}

// downloadFiles fetches the source of every file in a version's bundle.
func (r *regoPolicyResource) downloadFiles(ctx context.Context, identifier string, version int64) (map[string]string, error) {
	listing, err := r.client.ListRegoPolicyFiles(ctx, identifier, version)
	if err != nil {
		return nil, err
	}

	files := make(map[string]string, len(listing.Files))
	for _, file := range listing.Files {
		source, err := r.client.GetRegoPolicyFile(ctx, identifier, version, file.Path)
		if err != nil {
			return nil, fmt.Errorf("file %q: %w", file.Path, err)
		}
		files[file.Path] = source
	}
	return files, nil
}

// mapRegoPolicyToModel writes the policy's own fields into the model.
func mapRegoPolicyToModel(policy *client.RegoPolicy, data *regoPolicyResourceModel) {
	// The API returns the identifier as first published and compares it
	// without regard to case, so a configured spelling that differs only in
	// case is kept.
	if data.Identifier.IsNull() || data.Identifier.IsUnknown() || !strings.EqualFold(data.Identifier.ValueString(), policy.Identifier) {
		data.Identifier = types.StringValue(policy.Identifier)
	}
	data.Name = types.StringValue(policy.Name)

	// The API stores an empty description as none, so a configured "" is
	// kept to match the plan.
	switch {
	case policy.Description != "":
		data.Description = types.StringValue(policy.Description)
	case !data.Description.IsNull() && !data.Description.IsUnknown() && data.Description.ValueString() == "":
		data.Description = types.StringValue("")
	default:
		data.Description = types.StringNull()
	}

	data.CreatedAt = timestampToState(policy.CreatedAt)
	data.CreatedBy = types.StringValue(policy.CreatedBy)
}

// mapRegoPolicyVersionToModel writes a valid version's fields into the model.
func mapRegoPolicyVersionToModel(version *client.RegoPolicyVersion, data *regoPolicyResourceModel) {
	data.InputSchema = types.StringValue(version.InputSchema)
	data.Version = types.Int64Value(version.Version)
	data.Digest = types.StringValue(version.Digest)

	data.Entrypoint = types.StringNull()
	if version.Entrypoint != "" {
		data.Entrypoint = types.StringValue(version.Entrypoint)
	}

	data.ParamsSchema = types.StringNull()
	if schema := bytes.TrimSpace(version.ParamsSchema); len(schema) > 0 && string(schema) != "null" {
		var compact bytes.Buffer
		if err := json.Compact(&compact, schema); err == nil {
			data.ParamsSchema = types.StringValue(compact.String())
		} else {
			data.ParamsSchema = types.StringValue(string(schema))
		}
	}
}
