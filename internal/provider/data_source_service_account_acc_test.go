package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccServiceAccountDataSource_basic creates a service account and reads it
// back through the data source.
func TestAccServiceAccountDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	dataSourceName := "data.kosli_service_account.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountDataSourceConfig(rName, "CI service account", "member"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "description", "CI service account"),
					resource.TestCheckResourceAttr(dataSourceName, "privilege", "member"),
					resource.TestCheckResourceAttrSet(dataSourceName, "created_at"),
				),
			},
		},
	})
}

func testAccServiceAccountDataSourceConfig(name, description, privilege string) string {
	return fmt.Sprintf(`
resource "kosli_service_account" "test" {
  name        = %[1]q
  description = %[2]q
  privilege   = %[3]q
}

data "kosli_service_account" "test" {
  name = kosli_service_account.test.name
}
`, name, description, privilege)
}
