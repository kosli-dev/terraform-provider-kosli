package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccLogicalEnvironmentDataSource_basic tests querying an existing logical environment
func TestAccLogicalEnvironmentDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical-ds")
	envName1 := acctest.RandomWithPrefix("tf-acc-test-env1")
	envName2 := acctest.RandomWithPrefix("tf-acc-test-env2")
	resourceName := "kosli_logical_environment.test"
	dataSourceName := "data.kosli_logical_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLogicalEnvironmentDataSourceConfigBasic(rName, envName1, envName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes match resource
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "type", resourceName, "type"),
					resource.TestCheckResourceAttrPair(dataSourceName, "description", resourceName, "description"),
					// NOTE: API limitation - included_environments is not returned by GET endpoint
					resource.TestCheckResourceAttr(dataSourceName, "included_environments.#", "0"),
					// Verify timestamp field is populated
					resource.TestCheckResourceAttrSet(dataSourceName, "last_modified_at"),
				),
			},
		},
	})
}

// TestAccLogicalEnvironmentDataSource_computedAttributes tests all computed attributes are populated
func TestAccLogicalEnvironmentDataSource_computedAttributes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical-ds")
	envName1 := acctest.RandomWithPrefix("tf-acc-test-env1")
	envName2 := acctest.RandomWithPrefix("tf-acc-test-env2")
	resourceName := "kosli_logical_environment.test"
	dataSourceName := "data.kosli_logical_environment.test"
	description := "Test logical environment for data source verification"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLogicalEnvironmentDataSourceConfigFull(rName, envName1, envName2, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify all computed attributes
					resource.TestCheckResourceAttr(dataSourceName, "name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "type", "logical"),
					resource.TestCheckResourceAttr(dataSourceName, "description", description),
					// NOTE: API limitation - included_environments is not returned by GET endpoint
					resource.TestCheckResourceAttr(dataSourceName, "included_environments.#", "0"),
					resource.TestCheckResourceAttrSet(dataSourceName, "last_modified_at"),
					// Verify data source matches resource
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "type", resourceName, "type"),
					resource.TestCheckResourceAttrPair(dataSourceName, "description", resourceName, "description"),
				),
			},
		},
	})
}

// TestAccLogicalEnvironmentDataSource_notFound tests error handling for non-existent logical environment
func TestAccLogicalEnvironmentDataSource_notFound(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical-ds-notfound")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccLogicalEnvironmentDataSourceConfigNotFound(rName),
				ExpectError: regexp.MustCompile(`Could not read logical environment`),
			},
		},
	})
}

// TestAccLogicalEnvironmentDataSource_typeValidation tests that querying a physical environment fails
func TestAccLogicalEnvironmentDataSource_typeValidation(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-physical-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccLogicalEnvironmentDataSourceConfigTypeValidation(rName),
				ExpectError: regexp.MustCompile(`Invalid Environment Type`),
			},
		},
	})
}

// TestAccLogicalEnvironmentDataSource_emptyIncludedEnvironments tests handling of empty included_environments
func TestAccLogicalEnvironmentDataSource_emptyIncludedEnvironments(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical-ds-empty")
	dataSourceName := "data.kosli_logical_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLogicalEnvironmentDataSourceConfigEmpty(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "type", "logical"),
					resource.TestCheckResourceAttr(dataSourceName, "included_environments.#", "0"),
					resource.TestCheckResourceAttrSet(dataSourceName, "last_modified_at"),
				),
			},
		},
	})
}

// TestAccLogicalEnvironmentDataSource_nullDescription tests handling of null description
func TestAccLogicalEnvironmentDataSource_nullDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical-ds-nodesc")
	envName1 := acctest.RandomWithPrefix("tf-acc-test-env1")
	envName2 := acctest.RandomWithPrefix("tf-acc-test-env2")
	dataSourceName := "data.kosli_logical_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLogicalEnvironmentDataSourceConfigNoDescription(rName, envName1, envName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "type", "logical"),
					resource.TestCheckNoResourceAttr(dataSourceName, "description"),
					// NOTE: API limitation - included_environments is not returned by GET endpoint
					resource.TestCheckResourceAttr(dataSourceName, "included_environments.#", "0"),
					resource.TestCheckResourceAttrSet(dataSourceName, "last_modified_at"),
				),
			},
		},
	})
}

// TestAccLogicalEnvironmentDataSource_integration tests using data source output as resource input
func TestAccLogicalEnvironmentDataSource_integration(t *testing.T) {
	sourceName := acctest.RandomWithPrefix("tf-acc-test-logical-ds-source")
	derivedName := acctest.RandomWithPrefix("tf-acc-test-logical-ds-derived")
	envName1 := acctest.RandomWithPrefix("tf-acc-test-env1")
	envName2 := acctest.RandomWithPrefix("tf-acc-test-env2")
	envName3 := acctest.RandomWithPrefix("tf-acc-test-env3")
	dataSourceName := "data.kosli_logical_environment.source"
	derivedResourceName := "kosli_logical_environment.derived"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLogicalEnvironmentDataSourceConfigIntegration(sourceName, derivedName, envName1, envName2, envName3),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify source resource and data source
					resource.TestCheckResourceAttr(dataSourceName, "name", sourceName),
					resource.TestCheckResourceAttr(dataSourceName, "type", "logical"),
					// NOTE: API limitation - included_environments is not returned by GET endpoint
					resource.TestCheckResourceAttr(dataSourceName, "included_environments.#", "0"),
					resource.TestCheckResourceAttrSet(dataSourceName, "last_modified_at"),
					// Verify derived resource uses data source description
					resource.TestCheckResourceAttr(derivedResourceName, "name", derivedName),
					resource.TestCheckResourceAttr(derivedResourceName, "description", fmt.Sprintf("Derived from: %s", sourceName)),
				),
			},
		},
	})
}

// testAccLogicalEnvironmentDataSourceConfigBasic returns basic config with resource and data source
func testAccLogicalEnvironmentDataSourceConfigBasic(name, env1, env2 string) string {
	return fmt.Sprintf(`
# Create prerequisite physical environments
resource "kosli_environment" "env1" {
  name = %[2]q
  type = "K8S"
}

resource "kosli_environment" "env2" {
  name = %[3]q
  type = "ECS"
}

# Create logical environment
resource "kosli_logical_environment" "test" {
  name        = %[1]q
  description = "Test logical environment"
  included_environments = [
    kosli_environment.env1.name,
    kosli_environment.env2.name,
  ]
}

# Query logical environment via data source
data "kosli_logical_environment" "test" {
  name = kosli_logical_environment.test.name
}
`, name, env1, env2)
}

// testAccLogicalEnvironmentDataSourceConfigFull returns full config with all attributes
func testAccLogicalEnvironmentDataSourceConfigFull(name, env1, env2, description string) string {
	return fmt.Sprintf(`
# Create prerequisite physical environments
resource "kosli_environment" "env1" {
  name = %[2]q
  type = "K8S"
}

resource "kosli_environment" "env2" {
  name = %[3]q
  type = "docker"
}

# Create logical environment with all attributes
resource "kosli_logical_environment" "test" {
  name        = %[1]q
  description = %[4]q
  included_environments = [
    kosli_environment.env1.name,
    kosli_environment.env2.name,
  ]
}

# Query logical environment via data source
data "kosli_logical_environment" "test" {
  name = kosli_logical_environment.test.name
}
`, name, env1, env2, description)
}

// testAccLogicalEnvironmentDataSourceConfigNotFound returns config querying non-existent logical environment
func testAccLogicalEnvironmentDataSourceConfigNotFound(name string) string {
	return fmt.Sprintf(`
data "kosli_logical_environment" "test" {
  name = %[1]q
}
`, name)
}

// testAccLogicalEnvironmentDataSourceConfigTypeValidation returns config querying a physical environment
func testAccLogicalEnvironmentDataSourceConfigTypeValidation(name string) string {
	return fmt.Sprintf(`
# Create a physical environment
resource "kosli_environment" "test" {
  name = %[1]q
  type = "K8S"
}

# Try to query it as logical environment (should fail)
data "kosli_logical_environment" "test" {
  name = kosli_environment.test.name
}
`, name)
}

// testAccLogicalEnvironmentDataSourceConfigEmpty returns config with empty included_environments
func testAccLogicalEnvironmentDataSourceConfigEmpty(name string) string {
	return fmt.Sprintf(`
# Create logical environment with no included environments
resource "kosli_logical_environment" "test" {
  name                  = %[1]q
  included_environments = []
}

# Query logical environment via data source
data "kosli_logical_environment" "test" {
  name = kosli_logical_environment.test.name
}
`, name)
}

// testAccLogicalEnvironmentDataSourceConfigNoDescription returns config without description
func testAccLogicalEnvironmentDataSourceConfigNoDescription(name, env1, env2 string) string {
	return fmt.Sprintf(`
# Create prerequisite physical environments
resource "kosli_environment" "env1" {
  name = %[2]q
  type = "K8S"
}

resource "kosli_environment" "env2" {
  name = %[3]q
  type = "ECS"
}

# Create logical environment without description
resource "kosli_logical_environment" "test" {
  name = %[1]q
  included_environments = [
    kosli_environment.env1.name,
    kosli_environment.env2.name,
  ]
}

# Query logical environment via data source
data "kosli_logical_environment" "test" {
  name = kosli_logical_environment.test.name
}
`, name, env1, env2)
}

// testAccLogicalEnvironmentDataSourceConfigIntegration returns config for integration test
func testAccLogicalEnvironmentDataSourceConfigIntegration(sourceName, derivedName, env1, env2, env3 string) string {
	return fmt.Sprintf(`
# Create prerequisite physical environments
resource "kosli_environment" "env1" {
  name = %[3]q
  type = "K8S"
}

resource "kosli_environment" "env2" {
  name = %[4]q
  type = "ECS"
}

resource "kosli_environment" "env3" {
  name = %[5]q
  type = "docker"
}

# First logical environment
resource "kosli_logical_environment" "source" {
  name        = %[1]q
  description = "Source logical environment"
  included_environments = [
    kosli_environment.env1.name,
    kosli_environment.env2.name,
  ]
}

# Data source queries first logical environment
data "kosli_logical_environment" "source" {
  name = kosli_logical_environment.source.name
}

# Second logical environment uses data from first (via data source)
resource "kosli_logical_environment" "derived" {
  name        = %[2]q
  description = "Derived from: ${data.kosli_logical_environment.source.name}"
  included_environments = [
    kosli_environment.env3.name,
  ]
}
`, sourceName, derivedName, env1, env2, env3)
}
