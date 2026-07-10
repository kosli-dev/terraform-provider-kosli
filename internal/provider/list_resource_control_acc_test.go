package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// TestAccControlListResource_query provisions controls sharing a unique
// prefix, then runs `terraform query` against them: a search-filtered list
// (with resource data included) and a filterless list.
func TestAccControlListResource_query(t *testing.T) {
	prefix := acctest.RandomWithPrefix("tf-acc-query")
	id1 := prefix + "-a"
	id2 := prefix + "-b"

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccControlsPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			// terraform query / list resources require Terraform 1.14+.
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1 (config mode): provision two controls to be discovered.
			{
				Config: testAccControlListResourceConfig(id1, id2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kosli_control.a", "identifier", id1),
					resource.TestCheckResourceAttr("kosli_control.b", "identifier", id2),
				),
			},
			// Step 2 (query mode): Config is written verbatim to a
			// .tfquery.hcl file; the provider block resolves credentials from
			// KOSLI_API_TOKEN / KOSLI_ORG env vars.
			{
				Query: true,
				Config: fmt.Sprintf(`
provider "kosli" {}

list "kosli_control" "test" {
  provider         = kosli
  include_resource = true

  config {
    search = %[1]q
  }
}

list "kosli_control" "all" {
  provider = kosli
}
`, prefix),
				QueryResultChecks: []querycheck.QueryResultCheck{
					// The random prefix keeps the filtered count exact even
					// against a shared test organization.
					querycheck.ExpectLength("kosli_control.test", 2),
					querycheck.ExpectIdentity("kosli_control.test", map[string]knownvalue.Check{
						"identifier": knownvalue.StringExact(id1),
					}),
					querycheck.ExpectIdentity("kosli_control.test", map[string]knownvalue.Check{
						"identifier": knownvalue.StringExact(id2),
					}),
					querycheck.ExpectResourceKnownValues("kosli_control.test",
						queryfilter.ByResourceIdentity(map[string]knownvalue.Check{
							"identifier": knownvalue.StringExact(id1),
						}),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact("Control A")},
							{Path: tfjsonpath.New("description"), KnownValue: knownvalue.StringExact("Control A description")},
						},
					),
					querycheck.ExpectLengthAtLeast("kosli_control.all", 2),
				},
			},
		},
	})
}

func testAccControlListResourceConfig(id1, id2 string) string {
	return fmt.Sprintf(`
resource "kosli_control" "a" {
  identifier  = %[1]q
  name        = "Control A"
  description = "Control A description"
}

resource "kosli_control" "b" {
  identifier = %[2]q
  name       = "Control B"
}
`, id1, id2)
}
