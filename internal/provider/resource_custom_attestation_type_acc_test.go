package provider

import (
	"fmt"
	"os"
	"regexp"
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

// TestAccCustomAttestationTypeResource_optionalSchema tests creating resource with only jq_rules (no schema)
func TestAccCustomAttestationTypeResource_optionalSchema(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_custom_attestation_type.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomAttestationTypeResourceConfigOptionalSchema(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", "Attestation type with only jq rules"),
					resource.TestCheckNoResourceAttr(resourceName, "schema"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.0", ".age > 21"),
				),
			},
		},
	})
}

// TestAccCustomAttestationTypeResource_optionalJqRules tests creating resource with only schema (no jq_rules)
func TestAccCustomAttestationTypeResource_optionalJqRules(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_custom_attestation_type.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomAttestationTypeResourceConfigOptionalJqRules(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", "Attestation type with only schema"),
					resource.TestCheckResourceAttrSet(resourceName, "schema"),
					resource.TestCheckNoResourceAttr(resourceName, "jq_rules"),
				),
			},
		},
	})
}

// TestAccCustomAttestationTypeResource_updateRemoveSchema tests updating resource to remove schema
func TestAccCustomAttestationTypeResource_updateRemoveSchema(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_custom_attestation_type.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with schema
			{
				Config: testAccCustomAttestationTypeResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "schema"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.#", "1"),
				),
			},
			// Step 2: Remove schema, keep jq_rules
			{
				Config: testAccCustomAttestationTypeResourceConfigOptionalSchema(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckNoResourceAttr(resourceName, "schema"),
					resource.TestCheckResourceAttr(resourceName, "jq_rules.#", "1"),
				),
			},
		},
	})
}

// testAccCustomAttestationTypeResourceConfigOptionalSchema returns configuration with only jq_rules
func testAccCustomAttestationTypeResourceConfigOptionalSchema(name string) string {
	return fmt.Sprintf(`
resource "kosli_custom_attestation_type" "test" {
  name        = %[1]q
  description = "Attestation type with only jq rules"
  jq_rules    = [".age > 21"]
}
`, name)
}

// testAccCustomAttestationTypeResourceConfigOptionalJqRules returns configuration with only schema
func testAccCustomAttestationTypeResourceConfigOptionalJqRules(name string) string {
	return fmt.Sprintf(`
resource "kosli_custom_attestation_type" "test" {
  name        = %[1]q
  description = "Attestation type with only schema"
  schema = jsonencode({
    type = "object"
    properties = {
      age = {
        type = "number"
      }
    }
  })
}
`, name)
}

// TestAccCustomAttestationTypeResource_summary tests creating a resource with
// summary_json and verifies that the row ordering round-trips.
func TestAccCustomAttestationTypeResource_summary(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_custom_attestation_type.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomAttestationTypeResourceConfigSummary(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "summary_json",
						`[{"expression":".coverage","name":"Coverage"},{"expression":".report_url","name":"Report"}]`),
				),
			},
			// Import verifies that summary_json is populated from the API alone
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

// TestAccCustomAttestationTypeResource_summaryFormatting verifies that a
// summary_json written as pretty-printed JSON with non-alphabetical key order
// does not produce a perpetual diff against the compact, key-sorted form the API
// returns. This is what the semantic JSON equality of the attribute type buys:
// it stops the provider-returned value from clobbering the config as written.
func TestAccCustomAttestationTypeResource_summaryFormatting(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_custom_attestation_type.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomAttestationTypeResourceConfigSummaryFormatted(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// The config's own formatting is preserved in state
					resource.TestMatchResourceAttr(resourceName, "summary_json",
						regexp.MustCompile(`"name":\s+"Coverage"`)),
				),
			},
			// Re-planning the same config must be a no-op, i.e. the compact
			// form returned by the API did not overwrite the formatted config
			{
				Config:   testAccCustomAttestationTypeResourceConfigSummaryFormatted(rName),
				PlanOnly: true,
			},
		},
	})
}

// TestAccCustomAttestationTypeResource_updateSummary tests adding, changing and
// removing summary_json on an existing type. Removing it must clear the summary
// on the new version rather than inheriting the previous one.
func TestAccCustomAttestationTypeResource_updateSummary(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_custom_attestation_type.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create without a summary
			{
				Config: testAccCustomAttestationTypeResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "summary_json"),
				),
			},
			// Step 2: Add a summary
			{
				Config: testAccCustomAttestationTypeResourceConfigSummary(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "summary_json",
						`[{"expression":".coverage","name":"Coverage"},{"expression":".report_url","name":"Report"}]`),
				),
			},
			// Step 3: Change the summary rows
			{
				Config: testAccCustomAttestationTypeResourceConfigSummaryUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "summary_json",
						`[{"expression":".coverage","name":"Line coverage"}]`),
				),
			},
			// Step 4: Remove the summary again
			{
				Config: testAccCustomAttestationTypeResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "summary_json"),
				),
			},
		},
	})
}

// testAccCustomAttestationTypeResourceConfigSummary returns configuration with summary_json
func testAccCustomAttestationTypeResourceConfigSummary(name string) string {
	return fmt.Sprintf(`
resource "kosli_custom_attestation_type" "test" {
  name = %[1]q
  schema = jsonencode({
    type = "object"
    properties = {
      coverage = {
        type = "number"
      }
      report_url = {
        type = "string"
      }
    }
  })
  jq_rules = [".coverage >= 80"]
  summary_json = jsonencode([
    { name = "Coverage", expression = ".coverage" },
    { name = "Report", expression = ".report_url" },
  ])
}
`, name)
}

// testAccCustomAttestationTypeResourceConfigSummaryFormatted writes summary_json
// as a pretty-printed heredoc with "name" before "expression", i.e. neither the
// compact spacing nor the alphabetical key order the API returns.
func testAccCustomAttestationTypeResourceConfigSummaryFormatted(name string) string {
	return fmt.Sprintf(`
resource "kosli_custom_attestation_type" "test" {
  name = %[1]q
  schema = jsonencode({
    type = "object"
    properties = {
      coverage = {
        type = "number"
      }
      report_url = {
        type = "string"
      }
    }
  })
  jq_rules = [".coverage >= 80"]
  summary_json = <<-EOT
    [
      {
        "name": "Coverage",
        "expression": ".coverage"
      },
      {
        "name": "Report",
        "expression": ".report_url"
      }
    ]
  EOT
}
`, name)
}

// testAccCustomAttestationTypeResourceConfigSummaryUpdated returns configuration
// with a different set of summary rows
func testAccCustomAttestationTypeResourceConfigSummaryUpdated(name string) string {
	return fmt.Sprintf(`
resource "kosli_custom_attestation_type" "test" {
  name = %[1]q
  schema = jsonencode({
    type = "object"
    properties = {
      coverage = {
        type = "number"
      }
      report_url = {
        type = "string"
      }
    }
  })
  jq_rules = [".coverage >= 80"]
  summary_json = jsonencode([
    { name = "Line coverage", expression = ".coverage" },
  ])
}
`, name)
}
