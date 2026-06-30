package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccServiceAccountResource_basic tests minimal required configuration.
func TestAccServiceAccountResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_service_account.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountResourceConfig(rName, "reader"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "privilege", "reader"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
				),
			},
		},
	})
}

// TestAccServiceAccountResource_full tests all attributes including description.
func TestAccServiceAccountResource_full(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_service_account.test"
	description := "Service account for acceptance testing"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountResourceConfigFull(rName, description, "member"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "privilege", "member"),
				),
			},
		},
	})
}

// TestAccServiceAccountResource_update tests updating description and privilege.
func TestAccServiceAccountResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_service_account.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccServiceAccountResourceConfigFull(rName, "Initial description", "reader"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Initial description"),
					resource.TestCheckResourceAttr(resourceName, "privilege", "reader"),
				),
			},
			// Step 2: Update description and privilege
			{
				Config: testAccServiceAccountResourceConfigFull(rName, "Updated description", "member"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Updated description"),
					resource.TestCheckResourceAttr(resourceName, "privilege", "member"),
				),
			},
		},
	})
}

// TestAccServiceAccountResource_clearDescription verifies the description can be
// removed from an existing service account by deleting the attribute (PATCH
// sends a JSON null to clear it).
func TestAccServiceAccountResource_clearDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_service_account.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with a description
			{
				Config: testAccServiceAccountResourceConfigFull(rName, "Initial description", "reader"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Initial description"),
				),
			},
			// Step 2: Remove the description attribute - should clear it
			{
				Config: testAccServiceAccountResourceConfig(rName, "reader"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckNoResourceAttr(resourceName, "description"),
				),
			},
		},
	})
}

// TestAccServiceAccountResource_import tests terraform import functionality.
func TestAccServiceAccountResource_import(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_service_account.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountResourceConfigFull(rName, "Imported account", "member"),
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
			},
		},
	})
}

// TestAccServiceAccountResource_forceRecreate verifies that changing the name
// forces recreation.
func TestAccServiceAccountResource_forceRecreate(t *testing.T) {
	rName1 := acctest.RandomWithPrefix("tf-acc-test")
	rName2 := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_service_account.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountResourceConfig(rName1, "reader"),
				Check:  resource.TestCheckResourceAttr(resourceName, "name", rName1),
			},
			{
				Config: testAccServiceAccountResourceConfig(rName2, "reader"),
				Check:  resource.TestCheckResourceAttr(resourceName, "name", rName2),
			},
		},
	})
}

func testAccServiceAccountResourceConfig(name, privilege string) string {
	return fmt.Sprintf(`
resource "kosli_service_account" "test" {
  name      = %[1]q
  privilege = %[2]q
}
`, name, privilege)
}

func testAccServiceAccountResourceConfigFull(name, description, privilege string) string {
	return fmt.Sprintf(`
resource "kosli_service_account" "test" {
  name        = %[1]q
  description = %[2]q
  privilege   = %[3]q
}
`, name, description, privilege)
}
