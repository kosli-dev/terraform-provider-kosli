package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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
var _ resource.Resource = &notificationConfigResource{}
var _ resource.ResourceWithImportState = &notificationConfigResource{}
var _ resource.ResourceWithConfigValidators = &notificationConfigResource{}

// httpsURLRegexp mirrors the API's HttpsUrl constraint, so a plain-HTTP webhook
// fails at plan time instead of on apply.
var httpsURLRegexp = regexp.MustCompile(`^https://`)

// emailAddressRegexp is a loose shape check; the API performs full validation.
var emailAddressRegexp = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)

// NewNotificationConfigResource creates a new notification config resource.
func NewNotificationConfigResource() resource.Resource {
	return &notificationConfigResource{}
}

// notificationConfigResource defines the resource implementation.
type notificationConfigResource struct {
	client *client.Client
}

// notificationConfigResourceModel describes the resource data model.
type notificationConfigResourceModel struct {
	NotificationType types.String `tfsdk:"notification_type"`
	Emails           types.Set    `tfsdk:"emails"`
	SlackWebhooks    types.Set    `tfsdk:"slack_webhooks"`
	Webhooks         types.Set    `tfsdk:"webhooks"`
}

// Metadata returns the resource type name.
func (r *notificationConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_config"
}

// Schema defines the schema for the resource.
func (r *notificationConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages where Kosli sends one type of notification that Kosli raises itself, such as API key expiry warnings. " +
			"The configured targets replace the recipients Kosli would otherwise derive from what the notification is about.\n\n" +
			"~> **Note:** An organization has at most one configuration per `notification_type`. Creating this resource replaces any configuration " +
			"already set for that type (for example through the Kosli UI), and destroying it removes the configuration so Kosli derives the recipients again. " +
			"Requires a service account with **Admin** permissions.",

		Attributes: map[string]schema.Attribute{
			"notification_type": schema.StringAttribute{
				MarkdownDescription: "The type of notification to configure. Currently `" + client.NotificationTypeAPIKeyExpiry + "` " +
					"(warnings that a service account API key is about to expire). Changing this will force recreation of the resource.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"emails": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Email addresses the notification is sent to.",
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
				MarkdownDescription: "Slack incoming webhook URLs the notification is posted to. Must use HTTPS.",
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
				MarkdownDescription: "Generic webhook URLs the notification is POSTed to as JSON (payload version `1.0`). " +
					"Must use HTTPS.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(httpsURLRegexp, "must be an HTTPS URL"),
					),
				},
			},
		},
	}
}

// ConfigValidators requires at least one target: the API rejects an empty
// target list, and deleting the resource is how to restore the derived recipients.
func (r *notificationConfigResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("emails"),
			path.MatchRoot("slack_webhooks"),
			path.MatchRoot("webhooks"),
		),
	}
}

// Configure adds the provider configured client to the resource.
func (r *notificationConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *notificationConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data notificationConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, diags := r.set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(mapNotificationConfigToModel(ctx, config, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *notificationConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data notificationConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetNotificationConfig(ctx, data.NotificationType.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Notification Config",
			fmt.Sprintf("Could not read notification config %q: %s", data.NotificationType.ValueString(), err.Error()),
		)
		return
	}

	// The API answers 200 with no targets for a type nobody has configured, so
	// an empty target list means the config was deleted outside Terraform.
	if len(config.Targets) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(mapNotificationConfigToModel(ctx, config, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *notificationConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data notificationConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, diags := r.set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(mapNotificationConfigToModel(ctx, config, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *notificationConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data notificationConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteNotificationConfig(ctx, data.NotificationType.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Notification Config",
			fmt.Sprintf("Could not delete notification config %q: %s", data.NotificationType.ValueString(), err.Error()),
		)
		return
	}
}

// ImportState imports an existing notification config by its notification type.
// Read removes the resource when the type has no targets configured, so importing
// an unconfigured type fails as a non-existent object.
func (r *notificationConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("notification_type"), req, resp)
}

// set PUTs the model's targets and returns the stored config.
func (r *notificationConfigResource) set(ctx context.Context, data *notificationConfigResourceModel) (*client.NotificationConfigResponse, diag.Diagnostics) {
	var diags diag.Diagnostics

	values, d := notificationTargetValuesFromModel(ctx, data)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	config, err := r.client.SetNotificationConfig(ctx, data.NotificationType.ValueString(), &client.NotificationConfigRequest{
		Targets: values.targets(),
	})
	if err != nil {
		diags.AddError(
			"Error Setting Notification Config",
			fmt.Sprintf("Could not set notification config %q: %s", data.NotificationType.ValueString(), err.Error()),
		)
		return nil, diags
	}

	return config, diags
}

// notificationTargetValues holds a notification config's target values,
// grouped by target type.
type notificationTargetValues struct {
	Emails        []string
	SlackWebhooks []string
	Webhooks      []string
}

// groupNotificationTargets flattens the API's typed target list. Multiple EMAIL
// targets (possible when configured outside Terraform) merge into one list, and
// target types this provider does not know are ignored.
func groupNotificationTargets(targets []client.NotificationTarget) notificationTargetValues {
	var values notificationTargetValues
	for _, target := range targets {
		switch target.Type {
		case client.NotificationTargetTypeEmail:
			values.Emails = append(values.Emails, target.Emails...)
		case client.NotificationTargetTypeSlack:
			values.SlackWebhooks = append(values.SlackWebhooks, target.Webhook)
		case client.NotificationTargetTypeWebhook:
			values.Webhooks = append(values.Webhooks, target.Webhook)
		}
	}
	return values
}

// targets builds the API's typed target list: one EMAIL target carrying every
// address, and one SLACK or WEBHOOK target per URL.
func (v notificationTargetValues) targets() []client.NotificationTarget {
	targets := []client.NotificationTarget{}
	if len(v.Emails) > 0 {
		targets = append(targets, client.NotificationTarget{
			Type:   client.NotificationTargetTypeEmail,
			Emails: v.Emails,
		})
	}
	for _, webhook := range v.SlackWebhooks {
		targets = append(targets, client.NotificationTarget{
			Type:    client.NotificationTargetTypeSlack,
			Webhook: webhook,
		})
	}
	for _, webhook := range v.Webhooks {
		targets = append(targets, client.NotificationTarget{
			Type:    client.NotificationTargetTypeWebhook,
			Webhook: webhook,
		})
	}
	return targets
}

// notificationTargetValuesFromModel reads the target sets out of the resource
// model. Null sets read as empty.
func notificationTargetValuesFromModel(ctx context.Context, data *notificationConfigResourceModel) (notificationTargetValues, diag.Diagnostics) {
	var diags diag.Diagnostics
	var values notificationTargetValues

	diags.Append(stringSetElements(ctx, data.Emails, &values.Emails)...)
	diags.Append(stringSetElements(ctx, data.SlackWebhooks, &values.SlackWebhooks)...)
	diags.Append(stringSetElements(ctx, data.Webhooks, &values.Webhooks)...)

	return values, diags
}

// mapNotificationConfigToModel writes an API response into the resource model.
//
// The API normalizes what it stores (it lowercases URL hosts and email
// domains, and strips trailing slashes from URLs). Where the API returns a
// value equivalent to one already in the model, the model's spelling is kept,
// so a config written as "https://Hooks.Example.com/x/" neither fails the
// post-apply consistency check nor shows a diff on every plan.
func mapNotificationConfigToModel(ctx context.Context, config *client.NotificationConfigResponse, data *notificationConfigResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	prior, d := notificationTargetValuesFromModel(ctx, data)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	values := groupNotificationTargets(config.Targets)

	data.NotificationType = types.StringValue(config.NotificationType)

	data.Emails, d = stringSetOrNull(ctx, preferPriorSpelling(values.Emails, prior.Emails))
	diags.Append(d...)
	data.SlackWebhooks, d = stringSetOrNull(ctx, preferPriorSpelling(values.SlackWebhooks, prior.SlackWebhooks))
	diags.Append(d...)
	data.Webhooks, d = stringSetOrNull(ctx, preferPriorSpelling(values.Webhooks, prior.Webhooks))
	diags.Append(d...)

	return diags
}

// notificationValuesEquivalent reports whether two target values differ only in
// the ways the API normalizes them: letter case and trailing slashes.
func notificationValuesEquivalent(a, b string) bool {
	return strings.EqualFold(strings.TrimRight(a, "/"), strings.TrimRight(b, "/"))
}

// preferPriorSpelling returns the API values with each one replaced by its
// equivalent prior value, when there is one.
func preferPriorSpelling(apiValues, priorValues []string) []string {
	result := make([]string, 0, len(apiValues))
	for _, apiValue := range apiValues {
		value := apiValue
		for _, priorValue := range priorValues {
			if notificationValuesEquivalent(apiValue, priorValue) {
				value = priorValue
				break
			}
		}
		result = append(result, value)
	}
	return result
}

// stringSetElements reads a string set into dst. A null or unknown set leaves dst untouched.
func stringSetElements(ctx context.Context, set types.Set, dst *[]string) diag.Diagnostics {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	return set.ElementsAs(ctx, dst, false)
}

// stringSetOrNull builds a string set, or a null set when values is empty.
// Every target attribute requires at least one element, so an empty API list
// can only correspond to an unset attribute.
func stringSetOrNull(ctx context.Context, values []string) (types.Set, diag.Diagnostics) {
	if len(values) == 0 {
		return types.SetNull(types.StringType), nil
	}
	return types.SetValueFrom(ctx, types.StringType, uniqueStrings(values))
}

// stringSetOrEmpty builds a string set, empty rather than null when values is empty.
func stringSetOrEmpty(ctx context.Context, values []string) (types.Set, diag.Diagnostics) {
	if len(values) == 0 {
		return types.SetValueMust(types.StringType, []attr.Value{}), nil
	}
	return types.SetValueFrom(ctx, types.StringType, uniqueStrings(values))
}

// uniqueStrings drops repeated values, keeping first occurrences in order.
// A set value must not hold duplicates, and the API stores whatever list it is given.
func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}
