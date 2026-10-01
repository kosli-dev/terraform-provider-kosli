package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Notification configs are org-wide singletons keyed by notification type, so
// these tests cannot use random names: every test configures api_key_expiry on
// the test organization, replacing whatever is set there, and must not run in
// parallel with another test that does the same. Webhook targets use the
// reserved .invalid TLD (RFC 2606) so nothing is ever delivered to a real host.

// testAccNotificationConfigClient builds an API client from the acceptance test environment.
func testAccNotificationConfigClient() (*client.Client, error) {
	var opts []client.ClientOption
	if apiURL := os.Getenv("KOSLI_API_URL"); apiURL != "" {
		opts = append(opts, client.WithBaseURL(apiURL))
	}
	return client.NewClient(os.Getenv("KOSLI_API_TOKEN"), os.Getenv("KOSLI_ORG"), opts...)
}

// testAccNotificationConfigPreCheck skips when the target Kosli instance does
// not serve the notification config API yet.
func testAccNotificationConfigPreCheck(t *testing.T) {
	testAccPreCheck(t)

	c, err := testAccNotificationConfigClient()
	if err != nil {
		t.Fatalf("failed to create client for precheck: %v", err)
	}
	if _, err := c.GetNotificationConfig(context.Background(), client.NotificationTypeAPIKeyExpiry); client.IsNotFound(err) {
		t.Skip("The notification config API is not available on the target Kosli instance")
	}
}

// testAccCheckNotificationConfigDestroy verifies destroy removed the org's
// configuration, so the API reports no targets for the type.
func testAccCheckNotificationConfigDestroy(s *terraform.State) error {
	c, err := testAccNotificationConfigClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kosli_notification_config" {
			continue
		}

		notificationType := rs.Primary.Attributes["notification_type"]
		config, err := c.GetNotificationConfig(context.Background(), notificationType)
		if err != nil {
			return fmt.Errorf("error reading notification config %q after destroy: %w", notificationType, err)
		}
		if len(config.Targets) > 0 {
			return fmt.Errorf("notification config %q still has %d targets after destroy", notificationType, len(config.Targets))
		}
	}

	return nil
}

// TestAccNotificationConfigResource_basic tests a minimal email-only configuration.
func TestAccNotificationConfigResource_basic(t *testing.T) {
	resourceName := "kosli_notification_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccNotificationConfigPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNotificationConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationConfigResourceConfigEmails(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "notification_type", "api_key_expiry"),
					resource.TestCheckResourceAttr(resourceName, "emails.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "emails.*", "tf-acc-test@example.com"),
					resource.TestCheckNoResourceAttr(resourceName, "slack_webhooks"),
					resource.TestCheckNoResourceAttr(resourceName, "webhooks"),
				),
			},
		},
	})
}

// TestAccNotificationConfigResource_update tests adding and removing target types in place.
func TestAccNotificationConfigResource_update(t *testing.T) {
	resourceName := "kosli_notification_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccNotificationConfigPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNotificationConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationConfigResourceConfigEmails(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "emails.#", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "webhooks"),
				),
			},
			{
				Config: testAccNotificationConfigResourceConfigAllTargets(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "emails.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "emails.*", "tf-acc-test@example.com"),
					resource.TestCheckTypeSetElemAttr(resourceName, "emails.*", "tf-acc-test-2@example.com"),
					resource.TestCheckResourceAttr(resourceName, "slack_webhooks.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "slack_webhooks.*", "https://hooks.slack.invalid/services/T0/B0/tf-acc-test"),
					resource.TestCheckResourceAttr(resourceName, "webhooks.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "webhooks.*", "https://ops.example.invalid/tf-acc-test"),
				),
			},
			{
				Config: testAccNotificationConfigResourceConfigWebhookOnly(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "emails"),
					resource.TestCheckNoResourceAttr(resourceName, "slack_webhooks"),
					resource.TestCheckResourceAttr(resourceName, "webhooks.#", "1"),
				),
			},
		},
	})
}

// TestAccNotificationConfigResource_normalizedValues tests that values the API
// normalizes (uppercase host, trailing slash) apply cleanly and leave no diff.
func TestAccNotificationConfigResource_normalizedValues(t *testing.T) {
	resourceName := "kosli_notification_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccNotificationConfigPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNotificationConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
resource "kosli_notification_config" "test" {
  notification_type = "api_key_expiry"
  emails            = ["tf-acc-test@Example.COM"]
  webhooks          = ["https://Ops.Example.invalid/tf-acc-test/"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemAttr(resourceName, "emails.*", "tf-acc-test@Example.COM"),
					resource.TestCheckTypeSetElemAttr(resourceName, "webhooks.*", "https://Ops.Example.invalid/tf-acc-test/"),
				),
				// The test framework fails the step if a plan after apply is not empty.
			},
		},
	})
}

// TestAccNotificationConfigResource_import tests importing by notification type.
func TestAccNotificationConfigResource_import(t *testing.T) {
	resourceName := "kosli_notification_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccNotificationConfigPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNotificationConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationConfigResourceConfigAllTargets(),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        "api_key_expiry",
				ImportStateVerifyIdentifierAttribute: "notification_type",
			},
		},
	})
}

// TestAccNotificationConfigResource_deletedOutsideTerraform tests that a config
// removed outside Terraform is planned for recreation.
func TestAccNotificationConfigResource_deletedOutsideTerraform(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccNotificationConfigPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNotificationConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationConfigResourceConfigEmails(),
				Check: func(s *terraform.State) error {
					c, err := testAccNotificationConfigClient()
					if err != nil {
						return err
					}
					return c.DeleteNotificationConfig(context.Background(), client.NotificationTypeAPIKeyExpiry)
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccNotificationConfigResourceConfigEmails() string {
	return `
resource "kosli_notification_config" "test" {
  notification_type = "api_key_expiry"
  emails            = ["tf-acc-test@example.com"]
}
`
}

func testAccNotificationConfigResourceConfigAllTargets() string {
	return `
resource "kosli_notification_config" "test" {
  notification_type = "api_key_expiry"
  emails            = ["tf-acc-test@example.com", "tf-acc-test-2@example.com"]
  slack_webhooks    = ["https://hooks.slack.invalid/services/T0/B0/tf-acc-test"]
  webhooks          = ["https://ops.example.invalid/tf-acc-test"]
}
`
}

func testAccNotificationConfigResourceConfigWebhookOnly() string {
	return `
resource "kosli_notification_config" "test" {
  notification_type = "api_key_expiry"
  webhooks          = ["https://ops.example.invalid/tf-acc-test"]
}
`
}
