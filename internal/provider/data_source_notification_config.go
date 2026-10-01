package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &notificationConfigDataSource{}

// NewNotificationConfigDataSource creates a new notification config data source.
func NewNotificationConfigDataSource() datasource.DataSource {
	return &notificationConfigDataSource{}
}

// notificationConfigDataSource defines the data source implementation.
type notificationConfigDataSource struct {
	client *client.Client
}

// notificationConfigDataSourceModel describes the data source data model.
type notificationConfigDataSourceModel struct {
	NotificationType types.String `tfsdk:"notification_type"`
	Configured       types.Bool   `tfsdk:"configured"`
	Emails           types.Set    `tfsdk:"emails"`
	SlackWebhooks    types.Set    `tfsdk:"slack_webhooks"`
	Webhooks         types.Set    `tfsdk:"webhooks"`
	LastNotifiedAt   types.String `tfsdk:"last_notified_at"`
	LastFailureAt    types.String `tfsdk:"last_failure_at"`
	LastFailureError types.String `tfsdk:"last_failure_error"`
}

// Metadata returns the data source type name.
func (d *notificationConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_config"
}

// Schema defines the schema for the data source.
func (d *notificationConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches where Kosli sends one type of notification that Kosli raises itself, and when that notification was last delivered. " +
			"A type the organization has not configured reads with `configured = false` and empty target sets: Kosli then derives the recipients from what the notification is about.",

		Attributes: map[string]schema.Attribute{
			"notification_type": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The type of notification to query. Currently `" + client.NotificationTypeAPIKeyExpiry + "` " +
					"(warnings that a service account API key is about to expire).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"configured": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the organization has configured targets for this notification type.",
			},
			"emails": schema.SetAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "Email addresses the notification is sent to.",
			},
			"slack_webhooks": schema.SetAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Slack incoming webhook URLs the notification is posted to.",
			},
			"webhooks": schema.SetAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Generic webhook URLs the notification is POSTed to.",
			},
			"last_notified_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When a notification of this type was last sent for the organization (RFC3339). Null if none has been sent.",
			},
			"last_failure_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the most recent failed delivery happened (RFC3339). Null unless that failure is more recent than the last successful delivery.",
			},
			"last_failure_error": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Why the most recent failed delivery failed. Null unless that failure is more recent than the last successful delivery.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *notificationConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

// Read refreshes the Terraform state with the latest data.
func (d *notificationConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data notificationConfigDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := d.client.GetNotificationConfig(ctx, data.NotificationType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Notification Config",
			fmt.Sprintf("Could not read notification config %q: %s", data.NotificationType.ValueString(), err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(mapNotificationConfigToDataSourceModel(ctx, config, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// mapNotificationConfigToDataSourceModel writes an API response into the data source model.
func mapNotificationConfigToDataSourceModel(ctx context.Context, config *client.NotificationConfigResponse, data *notificationConfigDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	var d diag.Diagnostics

	values := groupNotificationTargets(config.Targets)

	data.NotificationType = types.StringValue(config.NotificationType)
	data.Configured = types.BoolValue(len(config.Targets) > 0)

	data.Emails, d = stringSetOrEmpty(ctx, values.Emails)
	diags.Append(d...)
	data.SlackWebhooks, d = stringSetOrEmpty(ctx, values.SlackWebhooks)
	diags.Append(d...)
	data.Webhooks, d = stringSetOrEmpty(ctx, values.Webhooks)
	diags.Append(d...)

	data.LastNotifiedAt = types.StringNull()
	if config.LastNotifiedAt != nil {
		data.LastNotifiedAt = timestampToState(*config.LastNotifiedAt)
	}

	data.LastFailureAt = types.StringNull()
	data.LastFailureError = types.StringNull()
	if config.LastFailure != nil {
		data.LastFailureAt = timestampToState(config.LastFailure.FailedAt)
		data.LastFailureError = types.StringValue(config.LastFailure.Error)
	}

	return diags
}
