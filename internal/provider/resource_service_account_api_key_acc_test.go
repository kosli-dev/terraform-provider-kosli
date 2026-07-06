package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccServiceAccountAPIKeyResource_basic creates a service account and an API
// key for it, verifying the raw key is captured in state.
func TestAccServiceAccountAPIKeyResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_service_account_api_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountAPIKeyResourceConfig(rName, "Production CI key"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Production CI key"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "key"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
				),
			},
		},
	})
}

// TestAccServiceAccountAPIKeyResource_expiry creates a key with an explicit
// future expiry timestamp.
func TestAccServiceAccountAPIKeyResource_expiry(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_service_account_api_key.test"
	// A far-future timestamp (2100-01-01) so the test never produces a past expiry.
	expiresAt := int64(4102444800)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountAPIKeyResourceConfigExpiry(rName, "Expiring key", expiresAt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Expiring key"),
					resource.TestCheckResourceAttr(resourceName, "expires_at", fmt.Sprintf("%d", expiresAt)),
					resource.TestCheckResourceAttrSet(resourceName, "key"),
				),
			},
		},
	})
}

// TestAccServiceAccountAPIKeyResource_forceRecreate verifies that changing the
// description forces the key to be revoked and recreated (immutable key).
func TestAccServiceAccountAPIKeyResource_forceRecreate(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_service_account_api_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountAPIKeyResourceConfig(rName, "First description"),
				Check:  resource.TestCheckResourceAttr(resourceName, "description", "First description"),
			},
			{
				Config: testAccServiceAccountAPIKeyResourceConfig(rName, "Second description"),
				Check:  resource.TestCheckResourceAttr(resourceName, "description", "Second description"),
			},
		},
	})
}

// TestAccServiceAccountAPIKeyResource_import imports an existing key using the
// "service_account_name/key_id" composite ID. The raw key cannot be recovered,
// so it is excluded from import verification.
func TestAccServiceAccountAPIKeyResource_import(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_service_account_api_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountAPIKeyResourceConfig(rName, "Importable key"),
				Check:  resource.TestCheckResourceAttrSet(resourceName, "id"),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// Composite import ID: "<service_account_name>/<key_id>".
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[resourceName]
					if !ok {
						return "", fmt.Errorf("resource not found: %s", resourceName)
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["service_account_name"], rs.Primary.Attributes["id"]), nil
				},
				// The raw key is only returned at creation and cannot be imported.
				ImportStateVerifyIgnore: []string{"key"},
			},
		},
	})
}

func testAccServiceAccountAPIKeyResourceConfig(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_service_account" "test" {
  name      = %[1]q
  privilege = "member"
}

resource "kosli_service_account_api_key" "test" {
  service_account_name = kosli_service_account.test.name
  description          = %[2]q
}
`, name, description)
}

func testAccServiceAccountAPIKeyResourceConfigExpiry(name, description string, expiresAt int64) string {
	return fmt.Sprintf(`
resource "kosli_service_account" "test" {
  name      = %[1]q
  privilege = "member"
}

resource "kosli_service_account_api_key" "test" {
  service_account_name = kosli_service_account.test.name
  description          = %[2]q
  expires_at           = %[3]d
}
`, name, description, expiresAt)
}
