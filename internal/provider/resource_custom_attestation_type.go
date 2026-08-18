package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &customAttestationTypeResource{}
var _ resource.ResourceWithImportState = &customAttestationTypeResource{}

// NewCustomAttestationTypeResource creates a new custom attestation type resource.
func NewCustomAttestationTypeResource() resource.Resource {
	return &customAttestationTypeResource{}
}

// customAttestationTypeResource defines the resource implementation.
type customAttestationTypeResource struct {
	client *client.Client
}

// customAttestationTypeResourceModel describes the resource data model.
type customAttestationTypeResourceModel struct {
	Name        types.String         `tfsdk:"name"`
	Description types.String         `tfsdk:"description"`
	Schema      jsontypes.Normalized `tfsdk:"schema"`
	JqRules     types.List           `tfsdk:"jq_rules"`
	Summary     jsontypes.Normalized `tfsdk:"summary"`
}

// Metadata returns the resource type name.
func (r *customAttestationTypeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_attestation_type"
}

// Schema defines the schema for the resource.
func (r *customAttestationTypeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom attestation type in Kosli. Custom attestation types define how Kosli validates and evaluates evidence from proprietary tools, custom metrics, or specialized compliance requirements.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the custom attestation type. Must start with a letter or number and can only contain letters, numbers, periods, hyphens, underscores, and tildes. Changing this will force recreation of the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the custom attestation type. Explains what this attestation type validates.",
				Optional:            true,
			},
			"schema": schema.StringAttribute{
				MarkdownDescription: "JSON Schema definition that defines the structure of attestation data. Can be provided inline using heredoc syntax or loaded from a file using `file()`. If omitted, no schema validation is performed. Semantic equality is used for comparison, so formatting differences are ignored.",
				Optional:            true,
				CustomType:          jsontypes.NormalizedType{},
			},
			"jq_rules": schema.ListAttribute{
				MarkdownDescription: "List of jq evaluation rules. Each rule is a jq expression that must evaluate to true for the attestation to be considered compliant. Example: `[\".coverage >= 80\"]`. If omitted, no evaluation is performed.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"summary": schema.StringAttribute{
				MarkdownDescription: "JSON array of ordered, labelled jq expressions rendered as rows on the attestation detail page in Kosli. Each element is an object with a `name` (the row label) and an `expression` (a jq expression evaluated against the attestation data); values that are valid URLs render as links. Can be provided inline using `jsonencode()`/heredoc syntax or loaded from a file using `file()`, so the same JSON can be kept in one place and shared with other tooling. Example: `jsonencode([{ name = \"Coverage\", expression = \".coverage\" }])`. If omitted, the attestation detail page falls back to showing the jq evaluation results as a pass/fail checklist; removing it from a type that had one clears the summary. Semantic JSON equality is used when reading the value back from Kosli, so your formatting is preserved rather than being rewritten to the API's compact form.",
				Optional:            true,
				CustomType:          jsontypes.NormalizedType{},
			},
		},
	}
}

// toCreateRequest builds the API request from the model. Updates go through the
// same request type because the API allocates a new version on every POST.
func (data *customAttestationTypeResourceModel) toCreateRequest(ctx context.Context, diags *diag.Diagnostics) *client.CreateCustomAttestationTypeRequest {
	// Extract jq_rules from the list if not null
	var jqRules []string
	if !data.JqRules.IsNull() {
		diags.Append(data.JqRules.ElementsAs(ctx, &jqRules, false)...)
		if diags.HasError() {
			return nil
		}
	}

	// Get schema value, handling null
	var schemaValue string
	if !data.Schema.IsNull() {
		schemaValue = data.Schema.ValueString()
	}

	// Get summary value, handling null. When it is null the client sends no
	// summary key, which clears any summary on the new version.
	var summaryValue string
	if !data.Summary.IsNull() {
		summaryValue = data.Summary.ValueString()
	}

	return &client.CreateCustomAttestationTypeRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Schema:      schemaValue,
		JqRules:     jqRules,
		Summary:     summaryValue,
	}
}

// applyAPIResponse maps an API response onto the model. Empty values are stored
// as null so that attributes absent from config don't show up as diffs.
func (data *customAttestationTypeResourceModel) applyAPIResponse(ctx context.Context, attestationType *client.CustomAttestationType, diags *diag.Diagnostics) {
	if attestationType.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(attestationType.Description)
	}

	if attestationType.Schema == "" || attestationType.Schema == "None" {
		data.Schema = jsontypes.NewNormalizedNull()
	} else {
		data.Schema = jsontypes.NewNormalizedValue(attestationType.Schema)
	}

	if attestationType.Summary == "" {
		data.Summary = jsontypes.NewNormalizedNull()
	} else {
		data.Summary = jsontypes.NewNormalizedValue(attestationType.Summary)
	}

	if len(attestationType.JqRules) == 0 {
		data.JqRules = types.ListNull(types.StringType)
	} else {
		jqRulesList, listDiags := types.ListValueFrom(ctx, types.StringType, attestationType.JqRules)
		diags.Append(listDiags...)
		if diags.HasError() {
			return
		}
		data.JqRules = jqRulesList
	}
}

// Configure adds the provider configured client to the resource.
func (r *customAttestationTypeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *customAttestationTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data customAttestationTypeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request
	createReq := data.toCreateRequest(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to create the custom attestation type
	if err := r.client.CreateCustomAttestationType(ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Custom Attestation Type",
			fmt.Sprintf("Could not create custom attestation type %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Per ADR 002: POST returns "OK", so we must GET to populate state.
	// We pass nil for the rePut callback because CreateCustomAttestationType
	// is a POST that allocates a new version on every call — re-issuing it
	// during a rename-race retry (issue #121) would silently bump the
	// version counter on the underlying type. The retry still surfaces the
	// rename-race hint so users know to use `terraform state mv`; full
	// recovery requires `state mv` or API support for upsert-on-archive.
	attestationType, err := retryReadAfterCreate(ctx,
		nil,
		func(ctx context.Context) (*client.CustomAttestationType, error) {
			return r.client.GetCustomAttestationType(ctx, createReq.Name, nil)
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			afterCreateSummary("Custom Attestation Type", err),
			renameRaceDetail("custom attestation type", createReq.Name, err),
		)
		return
	}

	// Map API response to Terraform state
	data.applyAPIResponse(ctx, attestationType, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *customAttestationTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data customAttestationTypeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state from API
	attestationType, err := r.client.GetCustomAttestationType(ctx, data.Name.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Custom Attestation Type",
			fmt.Sprintf("Could not read custom attestation type %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state
	data.applyAPIResponse(ctx, attestationType, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
// Per the API behavior, updates create a new version of the attestation type.
func (r *customAttestationTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data customAttestationTypeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request (updates create a new version)
	createReq := data.toCreateRequest(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to create new version
	if err := r.client.CreateCustomAttestationType(ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Custom Attestation Type",
			fmt.Sprintf("Could not update custom attestation type %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// GET to populate state with new version
	attestationType, err := r.client.GetCustomAttestationType(ctx, data.Name.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Custom Attestation Type After Update",
			fmt.Sprintf("Could not read custom attestation type %q after update: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to Terraform state
	data.applyAPIResponse(ctx, attestationType, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
// Per the API behavior, this archives the attestation type (soft delete).
func (r *customAttestationTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data customAttestationTypeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Archive the custom attestation type
	if err := r.client.ArchiveCustomAttestationType(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Custom Attestation Type",
			fmt.Sprintf("Could not archive custom attestation type %q: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	// State is automatically removed by the framework
}

// ImportState imports an existing resource into Terraform state.
func (r *customAttestationTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by name
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
