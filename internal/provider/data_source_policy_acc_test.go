package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPolicyDataSource_basic tests querying an existing policy
func TestAccPolicyDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds")
	resourceName := "kosli_policy.test"
	dataSourceName := "data.kosli_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDataSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "latest_version", resourceName, "latest_version"),
					resource.TestCheckResourceAttrPair(dataSourceName, "created_at", resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(dataSourceName, "content"),
				),
			},
		},
	})
}

// TestAccPolicyDataSource_withDescription tests querying a policy that has a description
func TestAccPolicyDataSource_withDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds")
	description := "Acceptance test policy"
	dataSourceName := "data.kosli_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDataSourceConfigWithDescription(rName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "description", description),
					resource.TestCheckResourceAttrSet(dataSourceName, "content"),
					resource.TestCheckResourceAttrSet(dataSourceName, "latest_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "created_at"),
				),
			},
		},
	})
}

// TestAccPolicyDataSource_notFound tests error handling for a non-existent policy
func TestAccPolicyDataSource_notFound(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-ds-notfound")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicyDataSourceConfigNotFound(rName),
				ExpectError: regexp.MustCompile(`Could not read policy`),
			},
		},
	})
}

// testAccPolicyDataSourceConfig returns a config with a resource and data source
func testAccPolicyDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "kosli_policy" "test" {
  name    = %[1]q
  content = %[2]q
}

data "kosli_policy" "test" {
  name = kosli_policy.test.name
}
`, name, testPolicyContent)
}

// testAccPolicyDataSourceConfigWithDescription returns config with description
func testAccPolicyDataSourceConfigWithDescription(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_policy" "test" {
  name        = %[1]q
  description = %[2]q
  content     = %[3]q
}

data "kosli_policy" "test" {
  name = kosli_policy.test.name
}
`, name, description, testPolicyContent)
}

// testAccPolicyDataSourceConfigNotFound returns a config querying a non-existent policy
func testAccPolicyDataSourceConfigNotFound(name string) string {
	return fmt.Sprintf(`
data "kosli_policy" "test" {
  name = %[1]q
}
`, name)
}
