package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccActionDataSource_basic tests querying an existing action
func TestAccActionDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds")
	envName := acctest.RandomWithPrefix("tf-acc-test-env")
	resourceName := "kosli_action.test"
	dataSourceName := "data.kosli_action.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccActionDataSourceConfig(rName, envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "environments.0", resourceName, "environments.0"),
					resource.TestCheckResourceAttrPair(dataSourceName, "triggers.0", resourceName, "triggers.0"),
					resource.TestCheckResourceAttrPair(dataSourceName, "number", resourceName, "number"),
					resource.TestCheckResourceAttrPair(dataSourceName, "created_by", resourceName, "created_by"),
					resource.TestCheckResourceAttrSet(dataSourceName, "last_modified_at"),
				),
			},
		},
	})
}

// TestAccActionDataSource_notFound tests error handling for a non-existent action
func TestAccActionDataSource_notFound(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds-notfound")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccActionDataSourceConfigNotFound(rName),
				ExpectError: regexp.MustCompile(`Could not read action`),
			},
		},
	})
}

// testAccActionDataSourceConfig returns a config with a resource and data source
func testAccActionDataSourceConfig(name, envName string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name = %[2]q
  type = "K8S"
}

resource "kosli_action" "test" {
  name         = %[1]q
  environments = [kosli_environment.test.name]
  triggers     = ["ON_NON_COMPLIANT_ENV"]
  webhook_url  = "https://hooks.example.com/kosli-test"
}

data "kosli_action" "test" {
  name = kosli_action.test.name
}
`, name, envName)
}

// testAccActionDataSourceConfigNotFound returns a config querying a non-existent action
func testAccActionDataSourceConfigNotFound(name string) string {
	return fmt.Sprintf(`
data "kosli_action" "test" {
  name = %[1]q
}
`, name)
}
