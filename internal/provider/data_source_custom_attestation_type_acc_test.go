package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccCustomAttestationTypeDataSource_basic tests querying an existing custom attestation type
func TestAccCustomAttestationTypeDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds")
	resourceName := "kosli_custom_attestation_type.test"
	dataSourceName := "data.kosli_custom_attestation_type.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomAttestationTypeDataSourceConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes match resource
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "schema", resourceName, "schema"),
					resource.TestCheckResourceAttrPair(dataSourceName, "jq_rules.#", resourceName, "jq_rules.#"),
					resource.TestCheckResourceAttrPair(dataSourceName, "jq_rules.0", resourceName, "jq_rules.0"),
					// Verify archived is false
					resource.TestCheckResourceAttr(dataSourceName, "archived", "false"),
				),
			},
		},
	})
}

// TestAccCustomAttestationTypeDataSource_computedAttributes tests all computed attributes are populated
func TestAccCustomAttestationTypeDataSource_computedAttributes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds")
	resourceName := "kosli_custom_attestation_type.test"
	dataSourceName := "data.kosli_custom_attestation_type.test"
	description := "Test attestation type for data source verification"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomAttestationTypeDataSourceConfigFull(rName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify all computed attributes
					resource.TestCheckResourceAttr(dataSourceName, "name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "description", description),
					resource.TestCheckResourceAttrSet(dataSourceName, "schema"),
					resource.TestCheckResourceAttr(dataSourceName, "jq_rules.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "jq_rules.0", ".coverage >= 80"),
					resource.TestCheckResourceAttr(dataSourceName, "jq_rules.1", ".branch == \"main\" or .branch == \"develop\""),
					resource.TestCheckResourceAttr(dataSourceName, "archived", "false"),
					// Verify data source matches resource
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "description", resourceName, "description"),
					resource.TestCheckResourceAttrPair(dataSourceName, "schema", resourceName, "schema"),
					resource.TestCheckResourceAttrPair(dataSourceName, "jq_rules.#", resourceName, "jq_rules.#"),
				),
			},
		},
	})
}

// TestAccCustomAttestationTypeDataSource_notFound tests error handling for non-existent attestation type
func TestAccCustomAttestationTypeDataSource_notFound(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds-notfound")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCustomAttestationTypeDataSourceConfigNotFound(rName),
				ExpectError: regexp.MustCompile(`Could not read custom attestation type`),
			},
		},
	})
}

// TestAccCustomAttestationTypeDataSource_archived tests querying archived attestation types
func TestAccCustomAttestationTypeDataSource_archived(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds-archived")
	dataSourceName := "data.kosli_custom_attestation_type.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create resource
			{
				Config: testAccCustomAttestationTypeDataSourceConfigArchived(rName, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kosli_custom_attestation_type.test", "name", rName),
				),
			},
			// Step 2: Destroy resource (which archives it)
			{
				Config:  testAccCustomAttestationTypeDataSourceConfigArchived(rName, 1),
				Destroy: true,
			},
			// Step 3: Query archived resource with data source
			// After server fix for #44, archived types are returned successfully
			// with archived=true instead of returning an error
			{
				Config: testAccCustomAttestationTypeDataSourceConfigArchived(rName, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "archived", "true"),
					resource.TestCheckResourceAttrSet(dataSourceName, "schema"),
					resource.TestCheckResourceAttr(dataSourceName, "jq_rules.#", "1"),
				),
			},
		},
	})
}

// TestAccCustomAttestationTypeDataSource_integration tests using data source output as resource input
func TestAccCustomAttestationTypeDataSource_integration(t *testing.T) {
	sourceName := acctest.RandomWithPrefix("tf-acc-test-ds-source")
	derivedName := acctest.RandomWithPrefix("tf-acc-test-ds-derived")
	dataSourceName := "data.kosli_custom_attestation_type.source"
	derivedResourceName := "kosli_custom_attestation_type.derived"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomAttestationTypeDataSourceConfigIntegration(sourceName, derivedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify source resource and data source
					resource.TestCheckResourceAttr(dataSourceName, "name", sourceName),
					resource.TestCheckResourceAttrSet(dataSourceName, "schema"),
					resource.TestCheckResourceAttr(dataSourceName, "jq_rules.#", "1"),
					// Verify derived resource uses data source values
					resource.TestCheckResourceAttr(derivedResourceName, "name", derivedName),
					resource.TestCheckResourceAttr(derivedResourceName, "description", fmt.Sprintf("Derived from: %s", sourceName)),
					// Verify schema and jq_rules match source
					resource.TestCheckResourceAttrPair(derivedResourceName, "schema", dataSourceName, "schema"),
					resource.TestCheckResourceAttrPair(derivedResourceName, "jq_rules.#", dataSourceName, "jq_rules.#"),
					resource.TestCheckResourceAttrPair(derivedResourceName, "jq_rules.0", dataSourceName, "jq_rules.0"),
				),
			},
		},
	})
}

// testAccCustomAttestationTypeDataSourceConfigBasic returns basic config with resource and data source
func testAccCustomAttestationTypeDataSourceConfigBasic(name string) string {
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

data "kosli_custom_attestation_type" "test" {
  name = kosli_custom_attestation_type.test.name
}
`, name)
}

// testAccCustomAttestationTypeDataSourceConfigFull returns full config with all attributes
func testAccCustomAttestationTypeDataSourceConfigFull(name, description string) string {
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

data "kosli_custom_attestation_type" "test" {
  name = kosli_custom_attestation_type.test.name
}
`, name, description)
}

// testAccCustomAttestationTypeDataSourceConfigNotFound returns config querying non-existent type
func testAccCustomAttestationTypeDataSourceConfigNotFound(name string) string {
	return fmt.Sprintf(`
data "kosli_custom_attestation_type" "test" {
  name = %[1]q
}
`, name)
}

// testAccCustomAttestationTypeDataSourceConfigArchived returns config for archived type test
func testAccCustomAttestationTypeDataSourceConfigArchived(name string, step int) string {
	if step == 1 {
		// Step 1: Create resource (also used for Step 2 with Destroy: true)
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

	// Step 2: Query archived resource with data source
	return fmt.Sprintf(`
data "kosli_custom_attestation_type" "test" {
  name = %[1]q
}
`, name)
}

// testAccCustomAttestationTypeDataSourceConfigIntegration returns config for integration test
func testAccCustomAttestationTypeDataSourceConfigIntegration(sourceName, derivedName string) string {
	return fmt.Sprintf(`
# First resource
resource "kosli_custom_attestation_type" "source" {
  name        = %[1]q
  description = "Source attestation type"
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

# Data source queries first resource
data "kosli_custom_attestation_type" "source" {
  name = kosli_custom_attestation_type.source.name
}

# Second resource uses data from first (via data source)
resource "kosli_custom_attestation_type" "derived" {
  name        = %[2]q
  description = "Derived from: ${data.kosli_custom_attestation_type.source.name}"
  schema      = data.kosli_custom_attestation_type.source.schema
  jq_rules    = data.kosli_custom_attestation_type.source.jq_rules
}
`, sourceName, derivedName)
}
