package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	testAccRegoPolicySource = `package kosli.tf_acc_test

import rego.v1

default allow := false

allow if input.is_compliant == true
`
	testAccRegoPolicySourceUpdated = `package kosli.tf_acc_test

import rego.v1

default allow := false

allow if {
	input.is_compliant == true
	input.artifact_fingerprint != ""
}
`
)

// Rego policies sit behind the Controls beta flag, so these tests are skipped
// when the test organization does not have it (see testAccControlsPreCheck).
// Kosli cannot delete a rego policy: each run leaves its policies behind.

// TestAccRegoPolicyResource_basic tests publishing a single-file policy.
func TestAccRegoPolicyResource_basic(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_rego_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRegoPolicyResourceConfig(rIdentifier, "Acceptance test policy", "", testAccRegoPolicySource),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "identifier", rIdentifier),
					resource.TestCheckResourceAttr(resourceName, "name", "Acceptance test policy"),
					resource.TestCheckResourceAttr(resourceName, "input_schema", "trail"),
					resource.TestCheckResourceAttr(resourceName, "version", "1"),
					resource.TestMatchResourceAttr(resourceName, "digest", regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)),
					resource.TestCheckResourceAttrSet(resourceName, "entrypoint"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "created_by"),
				),
			},
		},
	})
}

// TestAccRegoPolicyResource_update tests that a change of files publishes a
// new version and a change of metadata alone does not.
func TestAccRegoPolicyResource_update(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_rego_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRegoPolicyResourceConfig(rIdentifier, "Acceptance test policy", "", testAccRegoPolicySource),
				Check:  resource.TestCheckResourceAttr(resourceName, "version", "1"),
			},
			{
				Config: testAccRegoPolicyResourceConfig(rIdentifier, "Renamed policy", "Now described", testAccRegoPolicySource),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Renamed policy"),
					resource.TestCheckResourceAttr(resourceName, "description", "Now described"),
					resource.TestCheckResourceAttr(resourceName, "version", "1"),
				),
			},
			{
				Config: testAccRegoPolicyResourceConfig(rIdentifier, "Renamed policy", "", testAccRegoPolicySourceUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "description"),
					resource.TestCheckResourceAttr(resourceName, "version", "2"),
				),
			},
		},
	})
}

// TestAccRegoPolicyResource_import tests that an import reads the files of the
// current version into state.
func TestAccRegoPolicyResource_import(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_rego_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRegoPolicyResourceConfig(rIdentifier, "Acceptance test policy", "Imported", testAccRegoPolicySource),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        rIdentifier,
				ImportStateVerifyIdentifierAttribute: "identifier",
				// Kosli keeps the comment with the version and the resource
				// does not read it back.
				ImportStateVerifyIgnore: []string{"comment"},
			},
		},
	})
}

// TestAccRegoPolicyResource_rejected tests that a bundle Kosli rejects fails
// the apply with the validator's message.
func TestAccRegoPolicyResource_rejected(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRegoPolicyResourceConfig(rIdentifier, "Rejected policy", "", "package kosli.tf_acc_test\n\nallow if {\n"),
				ExpectError: regexp.MustCompile(`Rego Policy Version Rejected`),
			},
		},
	})
}

// testAccRegoPolicyResourceConfig returns a single-file policy configuration.
// An empty description leaves the attribute unset.
func testAccRegoPolicyResourceConfig(identifier, name, description, source string) string {
	descriptionLine := ""
	if description != "" {
		descriptionLine = fmt.Sprintf("description  = %q", description)
	}
	return fmt.Sprintf(`
resource "kosli_rego_policy" "test" {
  identifier   = %[1]q
  name         = %[2]q
  %[3]s
  input_schema = "trail"
  comment      = "Published by the provider acceptance tests"

  files = {
    "policy.rego" = %[4]q
  }
}
`, identifier, name, descriptionLine, source)
}
