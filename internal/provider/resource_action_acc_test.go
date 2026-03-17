package provider

import (
	"fmt"
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
				// webhook_url is sensitive and not returned by the API, so skip verification
				ImportStateVerifyIgnore: []string{"webhook_url"},
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
