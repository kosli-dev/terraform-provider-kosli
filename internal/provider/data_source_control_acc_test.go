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

// TestAccControlDataSource_version reads a historical version of a control
// after it has been updated.
func TestAccControlDataSource_version(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	dataSourceName := "data.kosli_control.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create version 1
			{
				Config: testAccControlResourceConfigFull(rIdentifier, "Binary provenance", "Initial description", "https://example.com/v1"),
			},
			// Step 2: Update to version 2, then read version 1 back through
			// the data source.
			{
				Config: testAccControlDataSourceVersionConfig(rIdentifier, "Binary provenance v2", "Updated description", "https://example.com/v2", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "version", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "Binary provenance"),
					resource.TestCheckResourceAttr(dataSourceName, "description", "Initial description"),
					resource.TestCheckResourceAttr(dataSourceName, "links.docs", "https://example.com/v1"),
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

func testAccControlDataSourceVersionConfig(identifier, name, description, docsLink string, version int) string {
	return fmt.Sprintf(`
resource "kosli_control" "test" {
  identifier  = %[1]q
  name        = %[2]q
  description = %[3]q
  links = {
    docs = %[4]q
  }
}

data "kosli_control" "test" {
  identifier = kosli_control.test.identifier
  version    = %[5]d
}
`, identifier, name, description, docsLink, version)
}

func testAccControlDataSourceNonExistentConfig(identifier string) string {
	return fmt.Sprintf(`
data "kosli_control" "test" {
  identifier = %[1]q
}
`, identifier)
}
