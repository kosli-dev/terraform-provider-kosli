package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccEnvironmentDataSource_basic tests querying an existing environment
func TestAccEnvironmentDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds")
	resourceName := "kosli_environment.test"
	dataSourceName := "data.kosli_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentDataSourceConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes match resource
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "type", resourceName, "type"),
					resource.TestCheckResourceAttrPair(dataSourceName, "description", resourceName, "description"),
					resource.TestCheckResourceAttrPair(dataSourceName, "include_scaling", resourceName, "include_scaling"),
					// Verify timestamp fields are populated
					resource.TestCheckResourceAttrSet(dataSourceName, "last_modified_at"),
				),
			},
		},
	})
}

// TestAccEnvironmentDataSource_computedAttributes tests all computed attributes are populated
func TestAccEnvironmentDataSource_computedAttributes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds")
	resourceName := "kosli_environment.test"
	dataSourceName := "data.kosli_environment.test"
	description := "Test environment for data source verification"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentDataSourceConfigFull(rName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify all computed attributes
					resource.TestCheckResourceAttr(dataSourceName, "name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "type", "ECS"),
					resource.TestCheckResourceAttr(dataSourceName, "description", description),
					resource.TestCheckResourceAttr(dataSourceName, "include_scaling", "true"),
					resource.TestCheckResourceAttrSet(dataSourceName, "last_modified_at"),
					// Verify data source matches resource
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "type", resourceName, "type"),
					resource.TestCheckResourceAttrPair(dataSourceName, "description", resourceName, "description"),
					resource.TestCheckResourceAttrPair(dataSourceName, "include_scaling", resourceName, "include_scaling"),
				),
			},
		},
	})
}

// TestAccEnvironmentDataSource_notFound tests error handling for non-existent environment
func TestAccEnvironmentDataSource_notFound(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds-notfound")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccEnvironmentDataSourceConfigNotFound(rName),
				ExpectError: regexp.MustCompile(`Could not read environment`),
			},
		},
	})
}

// TestAccEnvironmentDataSource_types tests querying different environment types
func TestAccEnvironmentDataSource_types(t *testing.T) {
	types := []string{"K8S", "ECS", "S3", "docker", "server", "lambda"}

	for _, envType := range types {
		t.Run(envType, func(t *testing.T) {
			rName := acctest.RandomWithPrefix(fmt.Sprintf("tf-acc-test-ds-%s", envType))
			dataSourceName := "data.kosli_environment.test"

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccEnvironmentDataSourceConfigType(rName, envType),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(dataSourceName, "name", rName),
							resource.TestCheckResourceAttr(dataSourceName, "type", envType),
							resource.TestCheckResourceAttrSet(dataSourceName, "last_modified_at"),
						),
					},
				},
			})
		})
	}
}

// TestAccEnvironmentDataSource_integration tests using data source output as resource input
func TestAccEnvironmentDataSource_integration(t *testing.T) {
	sourceName := acctest.RandomWithPrefix("tf-acc-test-ds-source")
	derivedName := acctest.RandomWithPrefix("tf-acc-test-ds-derived")
	dataSourceName := "data.kosli_environment.source"
	derivedResourceName := "kosli_environment.derived"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentDataSourceConfigIntegration(sourceName, derivedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify source resource and data source
					resource.TestCheckResourceAttr(dataSourceName, "name", sourceName),
					resource.TestCheckResourceAttr(dataSourceName, "type", "K8S"),
					resource.TestCheckResourceAttrSet(dataSourceName, "last_modified_at"),
					// Verify derived resource uses data source description
					resource.TestCheckResourceAttr(derivedResourceName, "name", derivedName),
					resource.TestCheckResourceAttr(derivedResourceName, "description", fmt.Sprintf("Derived from: %s", sourceName)),
				),
			},
		},
	})
}

// TestAccEnvironmentDataSource_nullDescription tests handling of null description
func TestAccEnvironmentDataSource_nullDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds-nodesc")
	dataSourceName := "data.kosli_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentDataSourceConfigNoDescription(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "type", "docker"),
					resource.TestCheckNoResourceAttr(dataSourceName, "description"),
					resource.TestCheckResourceAttrSet(dataSourceName, "last_modified_at"),
				),
			},
		},
	})
}

// testAccEnvironmentDataSourceConfigBasic returns basic config with resource and data source
func testAccEnvironmentDataSourceConfigBasic(name string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name        = %[1]q
  type        = "K8S"
  description = "Test environment"
}

data "kosli_environment" "test" {
  name = kosli_environment.test.name
}
`, name)
}

// testAccEnvironmentDataSourceConfigFull returns full config with all attributes
func testAccEnvironmentDataSourceConfigFull(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name            = %[1]q
  type            = "ECS"
  description     = %[2]q
  include_scaling = true
}

data "kosli_environment" "test" {
  name = kosli_environment.test.name
}
`, name, description)
}

// testAccEnvironmentDataSourceConfigNotFound returns config querying non-existent environment
func testAccEnvironmentDataSourceConfigNotFound(name string) string {
	return fmt.Sprintf(`
data "kosli_environment" "test" {
  name = %[1]q
}
`, name)
}

// testAccEnvironmentDataSourceConfigType returns config for specific environment type
func testAccEnvironmentDataSourceConfigType(name, envType string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name = %[1]q
  type = %[2]q
}

data "kosli_environment" "test" {
  name = kosli_environment.test.name
}
`, name, envType)
}

// testAccEnvironmentDataSourceConfigIntegration returns config for integration test
func testAccEnvironmentDataSourceConfigIntegration(sourceName, derivedName string) string {
	return fmt.Sprintf(`
# First environment
resource "kosli_environment" "source" {
  name        = %[1]q
  type        = "K8S"
  description = "Source environment"
}

# Data source queries first environment
data "kosli_environment" "source" {
  name = kosli_environment.source.name
}

# Second environment uses data from first (via data source)
resource "kosli_environment" "derived" {
  name        = %[2]q
  type        = "docker"
  description = "Derived from: ${data.kosli_environment.source.name}"
}
`, sourceName, derivedName)
}

// testAccEnvironmentDataSourceConfigNoDescription returns config without description
func testAccEnvironmentDataSourceConfigNoDescription(name string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name = %[1]q
  type = "docker"
}

data "kosli_environment" "test" {
  name = kosli_environment.test.name
}
`, name)
}
