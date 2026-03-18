package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testPolicyContent = `_schema: https://kosli.com/schemas/policy/environment/v1
artifacts:
  provenance:
    required: true
`

const testPolicyContentUpdated = `_schema: https://kosli.com/schemas/policy/environment/v1
artifacts:
  provenance:
    required: true
  trail-compliance:
    required: true
`

// TestAccPolicyResource_basic tests minimal required configuration
func TestAccPolicyResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "content"),
					resource.TestCheckResourceAttrSet(resourceName, "latest_version"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckNoResourceAttr(resourceName, "description"),
				),
			},
		},
	})
}

// TestAccPolicyResource_full tests all attributes including optional description
func TestAccPolicyResource_full(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_policy.test"
	description := "Test policy for acceptance testing"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyResourceConfigFull(rName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttrSet(resourceName, "content"),
					resource.TestCheckResourceAttrSet(resourceName, "latest_version"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
				),
			},
		},
	})
}

// TestAccPolicyResource_update tests that updating content creates a new version
func TestAccPolicyResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_policy.test"

	var versionAfterCreate string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial content, capture the version
			{
				Config: testAccPolicyResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "latest_version"),
					testAccCheckResourceAttrStore(resourceName, "latest_version", &versionAfterCreate),
				),
			},
			// Step 2: Update content — should increment version by 1
			{
				Config: testAccPolicyResourceConfigUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					testAccCheckVersionIncremented(resourceName, "latest_version", &versionAfterCreate),
				),
			},
		},
	})
}

// TestAccPolicyResource_import tests terraform import by name
func TestAccPolicyResource_import(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create resource
			{
				Config: testAccPolicyResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
				),
			},
			// Step 2: Import by name and verify state matches
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

// testAccCheckResourceAttrStore stores the value of resourceName.attr into dest for use in later steps.
func testAccCheckResourceAttrStore(resourceName, attr string, dest *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		v, ok := rs.Primary.Attributes[attr]
		if !ok {
			return fmt.Errorf("attribute %q not found on %s", attr, resourceName)
		}
		*dest = v
		return nil
	}
}

// testAccCheckVersionIncremented verifies that resourceName.attr equals *prev + 1.
func testAccCheckVersionIncremented(resourceName, attr string, prev *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		current, ok := rs.Primary.Attributes[attr]
		if !ok {
			return fmt.Errorf("attribute %q not found on %s", attr, resourceName)
		}
		prevInt, err := strconv.Atoi(*prev)
		if err != nil {
			return fmt.Errorf("previous version %q is not an integer: %w", *prev, err)
		}
		currentInt, err := strconv.Atoi(current)
		if err != nil {
			return fmt.Errorf("current version %q is not an integer: %w", current, err)
		}
		if currentInt != prevInt+1 {
			return fmt.Errorf("expected version %d (prev %d + 1), got %d", prevInt+1, prevInt, currentInt)
		}
		return nil
	}
}

// testAccPolicyResourceConfig returns basic configuration (name and content only)
func testAccPolicyResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "kosli_policy" "test" {
  name    = %[1]q
  content = %[2]q
}
`, name, testPolicyContent)
}

// testAccPolicyResourceConfigFull returns full configuration with all attributes
func testAccPolicyResourceConfigFull(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_policy" "test" {
  name        = %[1]q
  description = %[2]q
  content     = %[3]q
}
`, name, description, testPolicyContent)
}

// testAccPolicyResourceConfigUpdated returns configuration with updated content
func testAccPolicyResourceConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "kosli_policy" "test" {
  name    = %[1]q
  content = %[2]q
}
`, name, testPolicyContentUpdated)
}
