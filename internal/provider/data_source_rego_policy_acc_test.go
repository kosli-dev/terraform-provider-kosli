package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
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

// TestAccRegoPolicyDataSource_archived destroys a policy, which archives it,
// and reads it back: with archived = true the read succeeds and still returns
// the files, and without it the read is an error.
//
// A data source is read while planning, before the same apply destroys
// anything, so the destroy and the reads are separate steps. The read steps
// skip themselves on a Kosli server that cannot archive a rego policy, where
// the destroy leaves the policy active.
func TestAccRegoPolicyDataSource_archived(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	dataSourceName := "data.kosli_rego_policy.test"
	skipUnlessArchived := testAccRegoPolicyNotArchived(rIdentifier)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRegoPolicyResourceConfig(rIdentifier, "Acceptance test policy", "", testAccRegoPolicySource),
			},
			{
				Config:  testAccRegoPolicyResourceConfig(rIdentifier, "Acceptance test policy", "", testAccRegoPolicySource),
				Destroy: true,
			},
			{
				SkipFunc: skipUnlessArchived,
				Config: fmt.Sprintf(`
data "kosli_rego_policy" "test" {
  identifier = %q
  archived   = true
}
`, rIdentifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "archived", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "version", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "status", "valid"),
					resource.TestCheckResourceAttr(dataSourceName, "files.policy.rego", testAccRegoPolicySource),
				),
			},
			{
				SkipFunc: skipUnlessArchived,
				Config: fmt.Sprintf(`
data "kosli_rego_policy" "test" {
  identifier = %q
}
`, rIdentifier),
				ExpectError: regexp.MustCompile(`Rego Policy Archived`),
			},
		},
	})
}

// testAccRegoPolicyNotArchived returns a step SkipFunc that skips when the
// policy is not archived.
func testAccRegoPolicyNotArchived(identifier string) func() (bool, error) {
	return func() (bool, error) {
		var opts []client.ClientOption
		if apiURL := os.Getenv("KOSLI_API_URL"); apiURL != "" {
			opts = append(opts, client.WithBaseURL(apiURL))
		}
		c, err := client.NewClient(os.Getenv("KOSLI_API_TOKEN"), os.Getenv("KOSLI_ORG"), opts...)
		if err != nil {
			return false, fmt.Errorf("failed to create client: %w", err)
		}

		policy, err := c.GetRegoPolicy(context.Background(), identifier)
		if err != nil {
			return false, fmt.Errorf("failed to read rego policy %q: %w", identifier, err)
		}
		return !policy.Archived, nil
	}
}
