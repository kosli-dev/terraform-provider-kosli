package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccNotificationConfigDataSource_basic tests reading a configuration
// managed by the kosli_notification_config resource. Like the resource tests,
// it replaces the test organization's api_key_expiry configuration.
func TestAccNotificationConfigDataSource_basic(t *testing.T) {
	dataSourceName := "data.kosli_notification_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccNotificationConfigPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNotificationConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
resource "kosli_notification_config" "test" {
  notification_type = "api_key_expiry"
  emails            = ["tf-acc-test@example.com"]
  webhooks          = ["https://ops.example.invalid/tf-acc-test"]
}

data "kosli_notification_config" "test" {
  notification_type = kosli_notification_config.test.notification_type
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "notification_type", "api_key_expiry"),
					resource.TestCheckResourceAttr(dataSourceName, "configured", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "emails.#", "1"),
					resource.TestCheckTypeSetElemAttr(dataSourceName, "emails.*", "tf-acc-test@example.com"),
					resource.TestCheckResourceAttr(dataSourceName, "slack_webhooks.#", "0"),
					resource.TestCheckResourceAttr(dataSourceName, "webhooks.#", "1"),
					resource.TestCheckTypeSetElemAttr(dataSourceName, "webhooks.*", "https://ops.example.invalid/tf-acc-test"),
				),
			},
		},
	})
}
