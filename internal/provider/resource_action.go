package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &actionResource{}
var _ resource.ResourceWithImportState = &actionResource{}
var _ resource.ResourceWithConfigValidators = &actionResource{}

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
	Emails         types.Set     `tfsdk:"emails"`
	SlackWebhooks  types.Set     `tfsdk:"slack_webhooks"`
	Webhooks       types.Set     `tfsdk:"webhooks"`
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
		MarkdownDescription: "Manages a Kosli action. Actions send notifications to webhooks, Slack and email when artifact or environment events occur in the monitored environments. " +
			"At least one of `webhook_url`, `webhooks`, `slack_webhooks` or `emails` is required.\n\n" +
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
			// The list of values below is mirrored by the "## Triggers" table in
			// templates/resources/action.md.tmpl, which owns their meanings. Keep
			// the two value lists in sync when the API gains or drops a trigger.
			"triggers": schema.ListAttribute{
				ElementType: types.StringType,
				MarkdownDescription: "List of trigger event types that activate this action. " +
					"One or more of `ON_STARTED_ARTIFACT`, `ON_EXITED_ARTIFACT`, `ON_SCALED_ARTIFACT`, " +
					"`ON_ALLOWED_ARTIFACT`, `ON_COMPLIANT_ENV`, `ON_NON_COMPLIANT_ENV`. " +
					"See the Triggers section of the `kosli_action` documentation for what each one fires on.",
				Required: true,
			},
			"webhook_url": schema.StringAttribute{
				MarkdownDescription: "A single generic webhook URL to send notifications to. " +
					"Use `webhooks` for more than one; the two cannot be set together.",
				Optional:  true,
				Sensitive: true,
			},
			"emails": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Email addresses the notifications are sent to.",
				Optional:            true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(emailAddressRegexp, "must be an email address"),
					),
				},
			},
			"slack_webhooks": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Slack incoming webhook URLs the notifications are posted to. Must use HTTPS.",
				Optional:            true,
				Sensitive:           true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(httpsURLRegexp, "must be an HTTPS URL"),
					),
				},
			},
			"webhooks": schema.SetAttribute{
				ElementType: types.StringType,
				MarkdownDescription: "Generic webhook URLs the notifications are POSTed to as JSON (payload version `" +
					client.ActionPayloadVersionV1 + "`). Must use HTTPS.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(httpsURLRegexp, "must be an HTTPS URL"),
					),
				},
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

// ConfigValidators requires at least one target, since the API rejects an
// action without one, and keeps webhook_url and webhooks apart so each webhook
// target maps back to exactly one attribute.
func (r *actionResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("webhook_url"),
			path.MatchRoot("webhooks"),
			path.MatchRoot("slack_webhooks"),
			path.MatchRoot("emails"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("webhook_url"),
			path.MatchRoot("webhooks"),
		),
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
	ctx = resourceOperation(ctx, r, opCreate)
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
	ctx = resourceOperation(ctx, r, opRead)
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
	ctx = resourceOperation(ctx, r, opUpdate)
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
	ctx = resourceOperation(ctx, r, opDelete)
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
	ctx = resourceOperation(ctx, r, opImport)
	action, err := r.client.GetActionByName(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Action",
			fmt.Sprintf("Could not find action named %q: %s", req.ID, err.Error()),
		)
		return
	}

	if msg := slackAppActionImportError(action); msg != "" {
		resp.Diagnostics.AddError("Error Importing Action", msg)
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

	targets, d := actionTargetsFromModel(ctx, data)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	return &client.ActionRequest{
		Name: data.Name.ValueString(),
		// Type is always "env": the API stores flow actions but never fires them.
		Type:         "env",
		Environments: environments,
		Triggers:     triggers,
		Targets:      targets,
	}, diags
}

// actionTargetsFromModel builds the API's typed target list: one EMAIL target
// carrying every address, and one SLACK or WEBHOOK target per URL.
func actionTargetsFromModel(ctx context.Context, data *actionResourceModel) ([]client.ActionTarget, diag.Diagnostics) {
	var diags diag.Diagnostics
	var emails, slackWebhooks, webhooks []string

	diags.Append(stringSetElements(ctx, data.Emails, &emails)...)
	diags.Append(stringSetElements(ctx, data.SlackWebhooks, &slackWebhooks)...)
	diags.Append(stringSetElements(ctx, data.Webhooks, &webhooks)...)
	if diags.HasError() {
		return nil, diags
	}
	if !data.WebhookURL.IsNull() && !data.WebhookURL.IsUnknown() {
		webhooks = append([]string{data.WebhookURL.ValueString()}, webhooks...)
	}

	targets := []client.ActionTarget{}
	if len(emails) > 0 {
		targets = append(targets, client.ActionTarget{Type: client.ActionTargetTypeEmail, Emails: emails})
	}
	for _, webhook := range slackWebhooks {
		targets = append(targets, client.ActionTarget{Type: client.ActionTargetTypeSlack, Webhook: webhook})
	}
	for _, webhook := range webhooks {
		targets = append(targets, client.ActionTarget{
			Type:           client.ActionTargetTypeWebhook,
			Webhook:        webhook,
			PayloadVersion: client.ActionPayloadVersionV1,
		})
	}
	return targets, diags
}

// slackAppActionImportError returns why an action cannot be imported, or "" when
// it can. An action created from the Kosli Slack app carries a channel
// subscription on its Slack target that this resource does not model, so
// managing it here would detach it from that channel on the first update.
func slackAppActionImportError(action *client.ActionResponse) string {
	if !action.IsCreatedFromSlackApp {
		return ""
	}
	return fmt.Sprintf(
		"Action %q was created from the Kosli Slack app and is tied to a Slack channel subscription. "+
			"Manage it from Slack, or create a separate action with slack_webhooks.",
		action.Name,
	)
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

	diags.Append(mapActionTargetsToModel(ctx, action.Targets, data)...)

	return diags
}

// actionTargetValues holds an action's target values grouped by target type.
// A Withheld flag records that the API returned a target of that type without
// its URL, so the returned list for the type is incomplete.
type actionTargetValues struct {
	Emails                []string
	SlackWebhooks         []string
	Webhooks              []string
	SlackWebhooksWithheld bool
	WebhooksWithheld      bool
}

// groupActionTargets flattens the API's typed target list. Multiple EMAIL
// targets merge into one list, and target types this provider does not know
// are ignored.
func groupActionTargets(targets []client.ActionTarget) actionTargetValues {
	var values actionTargetValues
	for _, target := range targets {
		switch target.Type {
		case client.ActionTargetTypeEmail:
			values.Emails = append(values.Emails, target.Emails...)
		case client.ActionTargetTypeSlack:
			if target.Webhook == "" {
				values.SlackWebhooksWithheld = true
				continue
			}
			values.SlackWebhooks = append(values.SlackWebhooks, target.Webhook)
		case client.ActionTargetTypeWebhook:
			if target.Webhook == "" {
				values.WebhooksWithheld = true
				continue
			}
			values.Webhooks = append(values.Webhooks, target.Webhook)
		}
	}
	return values
}

// mapActionTargetsToModel writes the API's targets into the target attributes.
// Where the API returns a value equivalent to one already in the model, the
// model's spelling is kept, because the API normalizes the URLs and addresses
// it stores.
//
// A WEBHOOK target goes to webhook_url when the model already uses that
// attribute, or when the model has no targets yet (an import) and the action
// has exactly one webhook. Every other webhook goes to webhooks, so a target
// added outside Terraform shows up as a diff.
//
// When the API withholds a URL, the attributes for that target type keep the
// model's values: the returned list cannot be told apart from a changed one,
// and writing it would drop URLs the configuration still declares.
func mapActionTargetsToModel(ctx context.Context, targets []client.ActionTarget, data *actionResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	var priorEmails, priorSlackWebhooks, priorWebhooks []string

	diags.Append(stringSetElements(ctx, data.Emails, &priorEmails)...)
	diags.Append(stringSetElements(ctx, data.SlackWebhooks, &priorSlackWebhooks)...)
	diags.Append(stringSetElements(ctx, data.Webhooks, &priorWebhooks)...)
	if diags.HasError() {
		return diags
	}

	values := groupActionTargets(targets)
	webhooks := values.Webhooks

	usesWebhookURL := !data.WebhookURL.IsNull() && !data.WebhookURL.IsUnknown()
	hasNoTargets := !usesWebhookURL && data.Emails.IsNull() && data.SlackWebhooks.IsNull() && data.Webhooks.IsNull()

	switch {
	case values.WebhooksWithheld:
	case usesWebhookURL && len(webhooks) > 0:
		// With no match the configured URL was changed outside Terraform, and
		// the first webhook stands in for it so the plan shows the difference.
		index := 0
		for i, webhook := range webhooks {
			if notificationValuesEquivalent(webhook, data.WebhookURL.ValueString()) {
				index = i
				break
			}
		}
		if !notificationValuesEquivalent(webhooks[index], data.WebhookURL.ValueString()) {
			data.WebhookURL = types.StringValue(webhooks[index])
		}
		webhooks = append(webhooks[:index:index], webhooks[index+1:]...)
	case hasNoTargets && len(webhooks) == 1:
		data.WebhookURL = types.StringValue(webhooks[0])
		webhooks = nil
	}

	var d diag.Diagnostics
	data.Emails, d = stringSetOrNull(ctx, preferPriorSpelling(values.Emails, priorEmails))
	diags.Append(d...)
	if !values.SlackWebhooksWithheld {
		data.SlackWebhooks, d = stringSetOrNull(ctx, preferPriorSpelling(values.SlackWebhooks, priorSlackWebhooks))
		diags.Append(d...)
	}
	if !values.WebhooksWithheld {
		data.Webhooks, d = stringSetOrNull(ctx, preferPriorSpelling(webhooks, priorWebhooks))
		diags.Append(d...)
	}

	return diags
}
