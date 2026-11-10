package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccActionResource_basic tests minimal required configuration
func TestAccActionResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	envName := acctest.RandomWithPrefix("tf-acc-test-env")
	resourceName := "kosli_action.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccActionResourceConfig(rName, envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "environments.0", envName),
					resource.TestCheckResourceAttr(resourceName, "triggers.0", "ON_NON_COMPLIANT_ENV"),
					resource.TestCheckResourceAttrSet(resourceName, "number"),
					resource.TestCheckResourceAttrSet(resourceName, "created_by"),
					resource.TestCheckResourceAttrSet(resourceName, "last_modified_at"),
				),
			},
		},
	})
}

// TestAccActionResource_update tests updating mutable fields (triggers, webhook_url).
// Note: changing the sole environment of an action would archive that environment and
// trigger a server-side cascade delete of the action. This test therefore keeps the
// same environment across both steps and only mutates triggers and webhook_url.
func TestAccActionResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	envName := acctest.RandomWithPrefix("tf-acc-test-env")
	resourceName := "kosli_action.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccActionResourceConfig(rName, envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "environments.0", envName),
					resource.TestCheckResourceAttr(resourceName, "triggers.0", "ON_NON_COMPLIANT_ENV"),
				),
			},
			{
				Config: testAccActionResourceConfigUpdated(rName, envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "environments.0", envName),
					resource.TestCheckResourceAttr(resourceName, "triggers.0", "ON_COMPLIANT_ENV"),
				),
			},
		},
	})
}

// TestAccActionResource_import tests terraform import functionality
func TestAccActionResource_import(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	envName := acctest.RandomWithPrefix("tf-acc-test-env")
	resourceName := "kosli_action.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccActionResourceConfig(rName, envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        rName,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"webhook_url"},
			},
		},
	})
}

// TestAccActionResource_targets tests email, Slack and webhook targets on one
// action, an in-place change of the target set, and import of the result.
func TestAccActionResource_targets(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	envName := acctest.RandomWithPrefix("tf-acc-test-env")
	resourceName := "kosli_action.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccActionResourceConfigTargets(rName, envName, `
  emails         = ["alerts@example.com", "oncall@example.com"]
  slack_webhooks = ["https://hooks.example.com/kosli-test-slack"]
  webhooks       = ["https://hooks.example.com/kosli-test-a", "https://hooks.example.com/kosli-test-b"]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "emails.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "emails.*", "alerts@example.com"),
					resource.TestCheckResourceAttr(resourceName, "slack_webhooks.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhooks.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "webhook_url"),
					resource.TestCheckResourceAttrSet(resourceName, "number"),
				),
			},
			{
				Config: testAccActionResourceConfigTargets(rName, envName, `
  emails = ["alerts@example.com"]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "emails.#", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "slack_webhooks.#"),
					resource.TestCheckNoResourceAttr(resourceName, "webhooks.#"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        rName,
				ImportStateVerifyIdentifierAttribute: "name",
				// Kosli stamps each delivery on the action, which moves
				// last_modified_at between the update and the import.
				ImportStateVerifyIgnore: []string{"last_modified_at"},
			},
		},
	})
}

// TestAccActionResource_webhookURLToWebhooks tests moving a config from
// webhook_url to webhooks without replacing the action.
func TestAccActionResource_webhookURLToWebhooks(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	envName := acctest.RandomWithPrefix("tf-acc-test-env")
	resourceName := "kosli_action.test"
	var number string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccActionResourceConfigTargets(rName, envName, `
  webhook_url = "https://hooks.example.com/kosli-test"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "webhook_url", "https://hooks.example.com/kosli-test"),
					resource.TestCheckNoResourceAttr(resourceName, "webhooks.#"),
					resource.TestCheckResourceAttrWith(resourceName, "number", func(value string) error {
						number = value
						return nil
					}),
				),
			},
			{
				Config: testAccActionResourceConfigTargets(rName, envName, `
  webhooks = ["https://hooks.example.com/kosli-test"]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "webhook_url"),
					resource.TestCheckResourceAttr(resourceName, "webhooks.#", "1"),
					resource.TestCheckResourceAttrWith(resourceName, "number", func(value string) error {
						if value != number {
							return fmt.Errorf("action was replaced: number changed from %s to %s", number, value)
						}
						return nil
					}),
				),
			},
		},
	})
}

// TestAccActionResource_requiresTarget tests that an action without any target
// is rejected at plan time.
func TestAccActionResource_requiresTarget(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccActionResourceConfigTargets("tf-acc-test-no-target", "tf-acc-test-env-no-target", ""),
				ExpectError: regexp.MustCompile(`At least one of these attributes must be configured`),
			},
		},
	})
}

// TestAccActionResource_forceRecreate tests that changing name forces recreation
func TestAccActionResource_forceRecreate(t *testing.T) {
	rName1 := acctest.RandomWithPrefix("tf-acc-test")
	rName2 := acctest.RandomWithPrefix("tf-acc-test")
	envName := acctest.RandomWithPrefix("tf-acc-test-env")
	resourceName := "kosli_action.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccActionResourceConfig(rName1, envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName1),
				),
			},
			{
				Config: testAccActionResourceConfig(rName2, envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName2),
				),
			},
		},
	})
}

// testAccActionResourceConfig returns a basic action configuration backed by a K8S environment.
func testAccActionResourceConfig(name, envName string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name = %[2]q
  type = "K8S"
}

resource "kosli_action" "test" {
  name         = %[1]q
  environments = [kosli_environment.test.name]
  triggers     = ["ON_NON_COMPLIANT_ENV"]
  webhook_url  = "https://hooks.example.com/kosli-test"
}
`, name, envName)
}

// testAccActionResourceConfigTargets returns an action configuration whose
// target attributes are given verbatim.
func testAccActionResourceConfigTargets(name, envName, targets string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name = %[2]q
  type = "K8S"
}

resource "kosli_action" "test" {
  name         = %[1]q
  environments = [kosli_environment.test.name]
  triggers     = ["ON_NON_COMPLIANT_ENV"]
%[3]s
}
`, name, envName, targets)
}

// testAccActionResourceConfigUpdated returns an updated action configuration.
func testAccActionResourceConfigUpdated(name, envName string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name = %[2]q
  type = "K8S"
}

resource "kosli_action" "test" {
  name         = %[1]q
  environments = [kosli_environment.test.name]
  triggers     = ["ON_COMPLIANT_ENV"]
  webhook_url  = "https://hooks.example.com/kosli-test-updated"
}
`, name, envName)
}
