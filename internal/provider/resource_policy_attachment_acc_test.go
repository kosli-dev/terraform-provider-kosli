package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPolicyAttachmentResource_basic tests creating a policy attachment.
func TestAccPolicyAttachmentResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_policy_attachment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyAttachmentResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "environment_name", rName),
					resource.TestCheckResourceAttr(resourceName, "policy_name", rName),
				),
			},
		},
	})
}

// TestAccPolicyAttachmentResource_import tests terraform import by composite ID.
func TestAccPolicyAttachmentResource_import(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_policy_attachment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create the attachment
			{
				Config: testAccPolicyAttachmentResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "environment_name", rName),
					resource.TestCheckResourceAttr(resourceName, "policy_name", rName),
				),
			},
			// Step 2: Import using "environment_name/policy_name" format
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        fmt.Sprintf("%s/%s", rName, rName),
				ImportStateVerifyIdentifierAttribute: "environment_name",
			},
		},
	})
}

// testAccPolicyAttachmentResourceConfig returns config for a policy attachment
// with a prerequisite environment and policy using the same name.
func testAccPolicyAttachmentResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name = %[1]q
  type = "K8S"
}

resource "kosli_policy" "test" {
  name    = %[1]q
  content = %[2]q
}

resource "kosli_policy_attachment" "test" {
  environment_name = kosli_environment.test.name
  policy_name      = kosli_policy.test.name
}
`, name, testPolicyContent)
}
