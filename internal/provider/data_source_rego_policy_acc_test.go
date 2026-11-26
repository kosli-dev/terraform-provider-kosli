package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRegoPolicyDataSource_basic publishes a policy and reads its current
// version back through the data source, files included.
func TestAccRegoPolicyDataSource_basic(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	dataSourceName := "data.kosli_rego_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRegoPolicyResourceConfig(rIdentifier, "Acceptance test policy", "Read by a data source", testAccRegoPolicySource) + `
data "kosli_rego_policy" "test" {
  identifier = kosli_rego_policy.test.identifier
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "identifier", rIdentifier),
					resource.TestCheckResourceAttr(dataSourceName, "name", "Acceptance test policy"),
					resource.TestCheckResourceAttr(dataSourceName, "description", "Read by a data source"),
					resource.TestCheckResourceAttr(dataSourceName, "version", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "status", "valid"),
					resource.TestCheckResourceAttr(dataSourceName, "input_schema", "trail"),
					resource.TestCheckResourceAttr(dataSourceName, "archived", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "files.%", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "files.policy.rego", testAccRegoPolicySource),
					resource.TestCheckResourceAttrPair(dataSourceName, "digest", "kosli_rego_policy.test", "digest"),
					resource.TestCheckResourceAttrPair(dataSourceName, "entrypoint", "kosli_rego_policy.test", "entrypoint"),
					resource.TestCheckResourceAttrSet(dataSourceName, "created_at"),
					resource.TestCheckResourceAttrSet(dataSourceName, "created_by"),
				),
			},
		},
	})
}

// TestAccRegoPolicyDataSource_version reads the first version of a policy
// after a second version has been published.
func TestAccRegoPolicyDataSource_version(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	dataSourceName := "data.kosli_rego_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRegoPolicyResourceConfig(rIdentifier, "Acceptance test policy", "", testAccRegoPolicySource),
			},
			{
				Config: testAccRegoPolicyResourceConfig(rIdentifier, "Acceptance test policy", "", testAccRegoPolicySourceUpdated) + `
data "kosli_rego_policy" "test" {
  identifier = kosli_rego_policy.test.identifier
  version    = 1
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kosli_rego_policy.test", "version", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "version", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "status", "valid"),
					resource.TestCheckResourceAttr(dataSourceName, "files.policy.rego", testAccRegoPolicySource),
				),
			},
		},
	})
}

// TestAccRegoPolicyDataSource_notFound verifies the error for a policy that
// does not exist.
func TestAccRegoPolicyDataSource_notFound(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test-missing")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "kosli_rego_policy" "test" {
  identifier = %q
}
`, rIdentifier),
				ExpectError: regexp.MustCompile(`Rego Policy Not Found`),
			},
		},
	})
}
