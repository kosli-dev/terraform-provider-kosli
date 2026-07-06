package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccServiceAccountAPIKeysDataSource_basic creates a service account with an
// API key and reads the key list back through the data source.
func TestAccServiceAccountAPIKeysDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	dataSourceName := "data.kosli_service_account_api_keys.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountAPIKeysDataSourceConfig(rName, "Audited key"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "service_account_name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "keys.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.0.description", "Audited key"),
					resource.TestCheckResourceAttrSet(dataSourceName, "keys.0.id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "keys.0.created_at"),
				),
			},
		},
	})
}

func testAccServiceAccountAPIKeysDataSourceConfig(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_service_account" "test" {
  name      = %[1]q
  privilege = "member"
}

resource "kosli_service_account_api_key" "test" {
  service_account_name = kosli_service_account.test.name
  description          = %[2]q
}

data "kosli_service_account_api_keys" "test" {
  service_account_name = kosli_service_account.test.name

  # Ensure the key exists before listing.
  depends_on = [kosli_service_account_api_key.test]
}
`, name, description)
}
