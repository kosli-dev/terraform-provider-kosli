package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
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
	// The server caps key lifetime at 365 days and silently clamps anything
	// beyond it, so a far-future timestamp comes back rewritten and trips
	// Terraform's "inconsistent result after apply" check. A relative 30 days
	// stays inside the cap and is still in the future whenever the suite runs.
	expiresAt := time.Now().UTC().AddDate(0, 0, 30).Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountAPIKeyResourceConfigExpiry(rName, "Expiring key", expiresAt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Expiring key"),
					resource.TestCheckResourceAttr(resourceName, "expires_at", expiresAt),
					resource.TestCheckResourceAttrSet(resourceName, "key"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
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

// TestAccServiceAccountAPIKeyResource_disappears revokes the key out-of-band
// (directly via the API, outside Terraform) and expects the next refresh to
// produce a non-empty plan. This proves against the live API that a revoked
// key's GET returns a real 404 (not a 200/400) and that Read self-heals via
// RemoveResource instead of erroring.
func TestAccServiceAccountAPIKeyResource_disappears(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_service_account_api_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountAPIKeyResourceConfig(rName, "Disappearing key"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					testAccCheckServiceAccountAPIKeyDisappears(resourceName),
				),
				// The out-of-band revoke must surface as a plan to recreate the
				// key, not as a refresh error.
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// testAccCheckServiceAccountAPIKeyDisappears revokes the API key directly via
// the Kosli API, bypassing Terraform, using the same credentials as the
// provider under test.
func testAccCheckServiceAccountAPIKeyDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}

		serviceAccountName := rs.Primary.Attributes["service_account_name"]
		keyID := rs.Primary.Attributes["id"]
		if serviceAccountName == "" || keyID == "" {
			return fmt.Errorf("missing service_account_name or id in state for %s", resourceName)
		}

		var opts []client.ClientOption
		if apiURL := os.Getenv("KOSLI_API_URL"); apiURL != "" {
			opts = append(opts, client.WithBaseURL(apiURL))
		}
		c, err := client.NewClient(os.Getenv("KOSLI_API_TOKEN"), os.Getenv("KOSLI_ORG"), opts...)
		if err != nil {
			return fmt.Errorf("failed to create out-of-band client: %w", err)
		}

		if err := c.RevokeServiceAccountAPIKey(context.Background(), serviceAccountName, keyID); err != nil {
			return fmt.Errorf("failed to revoke API key out-of-band: %w", err)
		}

		return nil
	}
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

func testAccServiceAccountAPIKeyResourceConfigExpiry(name, description, expiresAt string) string {
	return fmt.Sprintf(`
resource "kosli_service_account" "test" {
  name      = %[1]q
  privilege = "member"
}

resource "kosli_service_account_api_key" "test" {
  service_account_name = kosli_service_account.test.name
  description          = %[2]q
  expires_at           = %[3]q
}
`, name, description, expiresAt)
}
