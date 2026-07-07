package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccControlDataSource_basic creates a control and reads it back through
// the data source.
func TestAccControlDataSource_basic(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	dataSourceName := "data.kosli_control.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlDataSourceConfig(rIdentifier, "Binary provenance", "Control for acceptance testing"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "identifier", rIdentifier),
					resource.TestCheckResourceAttr(dataSourceName, "name", "Binary provenance"),
					resource.TestCheckResourceAttr(dataSourceName, "description", "Control for acceptance testing"),
					resource.TestCheckResourceAttr(dataSourceName, "archived", "false"),
					resource.TestCheckResourceAttrSet(dataSourceName, "version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "created_at"),
					resource.TestCheckResourceAttrSet(dataSourceName, "created_by"),
				),
			},
		},
	})
}

// TestAccControlDataSource_nonExistent verifies the data source errors for a
// control that does not exist.
func TestAccControlDataSource_nonExistent(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test-missing")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccControlDataSourceNonExistentConfig(rIdentifier),
				ExpectError: regexp.MustCompile(`Control Not Found`),
			},
		},
	})
}

func testAccControlDataSourceConfig(identifier, name, description string) string {
	return fmt.Sprintf(`
resource "kosli_control" "test" {
  identifier  = %[1]q
  name        = %[2]q
  description = %[3]q
}

data "kosli_control" "test" {
  identifier = kosli_control.test.identifier
}
`, identifier, name, description)
}

func testAccControlDataSourceNonExistentConfig(identifier string) string {
	return fmt.Sprintf(`
data "kosli_control" "test" {
  identifier = %[1]q
}
`, identifier)
}
