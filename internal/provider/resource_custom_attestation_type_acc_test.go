package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccPreCheck validates required environment variables for acceptance tests
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv("KOSLI_API_TOKEN"); v == "" {
		t.Fatal("KOSLI_API_TOKEN must be set for acceptance tests")
	}
	if v := os.Getenv("KOSLI_ORG"); v == "" {
		t.Fatal("KOSLI_ORG must be set for acceptance tests")
	}
}

// TestAccCustomAttestationTypeResource_basic tests minimal required configuration
func TestAccCustomAttestationTypeResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_custom_attestation_type.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomAttestationTypeResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "schema"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.0", ".coverage >= 80"),
				),
			},
		},
	})
}

// TestAccCustomAttestationTypeResource_full tests all attributes including optional description
func TestAccCustomAttestationTypeResource_full(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_custom_attestation_type.test"
	description := "Test attestation type for coverage validation"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomAttestationTypeResourceConfigFull(rName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttrSet(resourceName, "schema"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.0", ".coverage >= 80"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.1", ".branch == \"main\" or .branch == \"develop\""),
				),
			},
		},
	})
}

// TestAccCustomAttestationTypeResource_update tests resource updates create new versions
func TestAccCustomAttestationTypeResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_custom_attestation_type.test"
	description1 := "Test attestation type for coverage validation"
	description2 := "Updated description for coverage validation"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial configuration
			{
				Config: testAccCustomAttestationTypeResourceConfigFull(rName, description1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description1),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.0", ".coverage >= 80"),
				),
			},
			// Step 2: Update description and jq_rules
			{
				Config: testAccCustomAttestationTypeResourceConfigUpdate1(rName, description2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description2),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.0", ".coverage >= 90"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.1", ".branch == \"main\""),
				),
			},
			// Step 3: Update schema and jq_rules to add new property
			{
				Config: testAccCustomAttestationTypeResourceConfigUpdate2(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "schema"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.0", ".coverage >= 90"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.1", ".vulnerabilities == 0"),
				),
			},
		},
	})
}

// TestAccCustomAttestationTypeResource_import tests terraform import functionality
func TestAccCustomAttestationTypeResource_import(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_custom_attestation_type.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create resource
			{
				Config: testAccCustomAttestationTypeResourceConfig(rName),
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

// testAccCustomAttestationTypeResourceConfig returns basic configuration
func testAccCustomAttestationTypeResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "kosli_custom_attestation_type" "test" {
  name = %[1]q
  schema = jsonencode({
    type = "object"
    properties = {
      coverage = {
        type = "number"
      }
    }
  })
  jq_rules = [".coverage >= 80"]
}
`, name)
}

// testAccCustomAttestationTypeResourceConfigFull returns full configuration with all attributes
func testAccCustomAttestationTypeResourceConfigFull(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_custom_attestation_type" "test" {
  name        = %[1]q
  description = %[2]q
  schema = jsonencode({
    type = "object"
    properties = {
      coverage = {
        type = "number"
      }
      branch = {
        type = "string"
      }
    }
    required = ["coverage"]
  })
  jq_rules = [
    ".coverage >= 80",
    ".branch == \"main\" or .branch == \"develop\""
  ]
}
`, name, description)
}

// testAccCustomAttestationTypeResourceConfigUpdate1 returns updated configuration for first update
func testAccCustomAttestationTypeResourceConfigUpdate1(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_custom_attestation_type" "test" {
  name        = %[1]q
  description = %[2]q
  schema = jsonencode({
    type = "object"
    properties = {
      coverage = {
        type = "number"
      }
      branch = {
        type = "string"
      }
    }
    required = ["coverage"]
  })
  jq_rules = [
    ".coverage >= 90",
    ".branch == \"main\""
  ]
}
`, name, description)
}

// testAccCustomAttestationTypeResourceConfigUpdate2 returns updated configuration for second update
func testAccCustomAttestationTypeResourceConfigUpdate2(name string) string {
	return fmt.Sprintf(`
resource "kosli_custom_attestation_type" "test" {
  name        = %[1]q
  description = "Updated with vulnerabilities check"
  schema = jsonencode({
    type = "object"
    properties = {
      coverage = {
        type = "number"
      }
      branch = {
        type = "string"
      }
      vulnerabilities = {
        type = "number"
      }
    }
    required = ["coverage", "vulnerabilities"]
  })
  jq_rules = [
    ".coverage >= 90",
    ".vulnerabilities == 0"
  ]
}
`, name)
}
