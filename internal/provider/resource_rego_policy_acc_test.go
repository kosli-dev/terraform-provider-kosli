package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Kosli type-checks a policy against its input schema, so the sources only
// read fields the "trail" schema defines.
const (
	testAccRegoPolicySource = `package policy

import rego.v1

default allow := false

allow if count(input.trail.compliance_status.artifacts_statuses) > 0
`
	testAccRegoPolicySourceUpdated = `package policy

import rego.v1

default allow := false

allow if count(input.trail.compliance_status.artifacts_statuses) > 0

violations contains "the trail has no artifacts" if {
	count(input.trail.compliance_status.artifacts_statuses) == 0
}
`
)

// Kosli cannot delete a rego policy: each run leaves its policies archived.

// TestAccRegoPolicyResource_basic tests publishing a single-file policy.
func TestAccRegoPolicyResource_basic(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_rego_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRegoPolicyArchived(rIdentifier),
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRegoPolicyResourceConfig(rIdentifier, "Rejected policy", "", "package policy\n\nallow if {\n"),
				ExpectError: regexp.MustCompile(`Rego Policy Version Rejected`),
			},
		},
	})
}

// TestAccRegoPolicyResource_recreateArchived tests that creating a policy
// whose identifier belongs to an archived policy unarchives it and continues
// its version numbers.
func TestAccRegoPolicyResource_recreateArchived(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_rego_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRegoPolicyResourceConfig(rIdentifier, "Acceptance test policy", "", testAccRegoPolicySource),
				Check:  resource.TestCheckResourceAttr(resourceName, "version", "1"),
			},
			{
				Config:  testAccRegoPolicyResourceConfig(rIdentifier, "Acceptance test policy", "", testAccRegoPolicySource),
				Destroy: true,
			},
			{
				Config: testAccRegoPolicyResourceConfig(rIdentifier, "Acceptance test policy", "", testAccRegoPolicySourceUpdated),
				Check:  resource.TestCheckResourceAttr(resourceName, "version", "2"),
			},
		},
	})
}

// testAccCheckRegoPolicyArchived verifies that destroy archived the policy,
// on a server that can archive one.
func testAccCheckRegoPolicyArchived(identifier string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		var opts []client.ClientOption
		if apiURL := os.Getenv("KOSLI_API_URL"); apiURL != "" {
			opts = append(opts, client.WithBaseURL(apiURL))
		}
		c, err := client.NewClient(os.Getenv("KOSLI_API_TOKEN"), os.Getenv("KOSLI_ORG"), opts...)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		policy, err := c.GetRegoPolicy(context.Background(), identifier)
		if err != nil {
			return fmt.Errorf("failed to read rego policy %q after destroy: %w", identifier, err)
		}
		if policy.Archived {
			return nil
		}

		// A server that predates the archive routes answers 404 for a policy
		// that exists. Destroy then leaves the policy active with a warning,
		// which is not a failure of the provider.
		if _, err := c.ArchiveRegoPolicy(context.Background(), identifier); client.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("rego policy %q is not archived after destroy", identifier)
	}
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
