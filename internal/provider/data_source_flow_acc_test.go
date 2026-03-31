package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccFlowDataSource_basic tests querying an existing flow
func TestAccFlowDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds")
	resourceName := "kosli_flow.test"
	dataSourceName := "data.kosli_flow.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowDataSourceConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckNoResourceAttr(dataSourceName, "description"),
				),
			},
		},
	})
}

// TestAccFlowDataSource_withDescription tests querying a flow with a description
func TestAccFlowDataSource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds")
	description := "Test flow for data source verification"
	resourceName := "kosli_flow.test"
	dataSourceName := "data.kosli_flow.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowDataSourceConfigWithDescription(rName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "description", description),
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "description", resourceName, "description"),
				),
			},
		},
	})
}

// TestAccFlowDataSource_withTemplate tests querying a flow with a YAML template
func TestAccFlowDataSource_withTemplate(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds")
	resourceName := "kosli_flow.test"
	dataSourceName := "data.kosli_flow.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowDataSourceConfigWithTemplate(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrSet(dataSourceName, "template"),
				),
			},
		},
	})
}

// TestAccFlowDataSource_notFound tests error handling for a non-existent flow
func TestAccFlowDataSource_notFound(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds-notfound")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccFlowDataSourceConfigNotFound(rName),
				ExpectError: regexp.MustCompile(`Could not read flow`),
			},
		},
	})
}

// testAccFlowDataSourceConfigBasic returns a config with a resource and data source (no description)
func testAccFlowDataSourceConfigBasic(name string) string {
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name = %[1]q
}

data "kosli_flow" "test" {
  name = kosli_flow.test.name
}
`, name)
}

// testAccFlowDataSourceConfigWithDescription returns a config with a description
func testAccFlowDataSourceConfigWithDescription(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name        = %[1]q
  description = %[2]q
}

data "kosli_flow" "test" {
  name = kosli_flow.test.name
}
`, name, description)
}

// testAccFlowDataSourceConfigWithTemplate returns a config with a YAML template
func testAccFlowDataSourceConfigWithTemplate(name string) string {
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name     = %[1]q
  template = <<-YAML
version: 1
trail:
  artifacts:
    - name: application
      attestations:
        - name: unit-tests
          type: junit
YAML
}

data "kosli_flow" "test" {
  name = kosli_flow.test.name
}
`, name)
}

// testAccFlowDataSourceConfigNotFound returns a config querying a non-existent flow
func testAccFlowDataSourceConfigNotFound(name string) string {
	return fmt.Sprintf(`
data "kosli_flow" "test" {
  name = %[1]q
}
`, name)
}
