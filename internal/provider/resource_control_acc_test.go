package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// testAccControlsPreCheck skips control acceptance tests when the Controls
// beta feature is not enabled for the test organization (every Controls
// endpoint returns 403 Forbidden in that case).
func testAccControlsPreCheck(t *testing.T) {
	testAccPreCheck(t)

	var opts []client.ClientOption
	if apiURL := os.Getenv("KOSLI_API_URL"); apiURL != "" {
		opts = append(opts, client.WithBaseURL(apiURL))
	}
	c, err := client.NewClient(os.Getenv("KOSLI_API_TOKEN"), os.Getenv("KOSLI_ORG"), opts...)
	if err != nil {
		t.Fatalf("failed to create client for precheck: %v", err)
	}

	if _, err := c.ListControls(context.Background(), nil); client.IsForbidden(err) {
		t.Skip("Controls is a beta feature and is not enabled for the test organization")
	}
}

// TestAccControlResource_basic tests minimal required configuration.
func TestAccControlResource_basic(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_control.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlResourceConfig(rIdentifier, "Binary provenance"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "identifier", rIdentifier),
					resource.TestCheckResourceAttr(resourceName, "name", "Binary provenance"),
					resource.TestCheckResourceAttrSet(resourceName, "version"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "created_by"),
				),
			},
		},
	})
}

// TestAccControlResource_full tests all configurable attributes.
func TestAccControlResource_full(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_control.test"
	description := "Control for acceptance testing"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlResourceConfigFull(rIdentifier, "Binary provenance", description, "https://example.com/docs"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "identifier", rIdentifier),
					resource.TestCheckResourceAttr(resourceName, "name", "Binary provenance"),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "links.docs", "https://example.com/docs"),
				),
			},
		},
	})
}

// TestAccControlResource_update tests updating name, description, and links
// in place without recreating the control.
func TestAccControlResource_update(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_control.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccControlResourceConfigFull(rIdentifier, "Binary provenance", "Initial description", "https://example.com/v1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Binary provenance"),
					resource.TestCheckResourceAttr(resourceName, "description", "Initial description"),
					resource.TestCheckResourceAttr(resourceName, "version", "1"),
				),
			},
			// Step 2: Update all mutable attributes; the identifier is
			// unchanged so the control must be updated in place.
			{
				Config: testAccControlResourceConfigFull(rIdentifier, "Binary provenance v2", "Updated description", "https://example.com/v2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "identifier", rIdentifier),
					resource.TestCheckResourceAttr(resourceName, "name", "Binary provenance v2"),
					resource.TestCheckResourceAttr(resourceName, "description", "Updated description"),
					resource.TestCheckResourceAttr(resourceName, "links.docs", "https://example.com/v2"),
					resource.TestCheckResourceAttr(resourceName, "version", "2"),
				),
			},
		},
	})
}

// TestAccControlResource_clearDescription verifies the description can be
// removed from an existing control by deleting the attribute (PUT replaces
// the mutable fields wholesale).
func TestAccControlResource_clearDescription(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_control.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with a description
			{
				Config: testAccControlResourceConfigFull(rIdentifier, "Binary provenance", "Initial description", "https://example.com/docs"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Initial description"),
				),
			},
			// Step 2: Remove the description and links attributes - should clear them
			{
				Config: testAccControlResourceConfig(rIdentifier, "Binary provenance"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "identifier", rIdentifier),
					resource.TestCheckNoResourceAttr(resourceName, "description"),
					resource.TestCheckNoResourceAttr(resourceName, "links"),
				),
			},
		},
	})
}

// TestAccControlResource_tags tests setting, changing, and clearing tags,
// which are managed via the dedicated tags PATCH endpoint.
func TestAccControlResource_tags(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_control.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with a tag
			{
				Config: testAccControlResourceConfigTags(rIdentifier, `{ team = "platform" }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.team", "platform"),
				),
			},
			// Step 2: Change and add tags
			{
				Config: testAccControlResourceConfigTags(rIdentifier, `{ team = "security", framework = "finos-sdlc" }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.team", "security"),
					resource.TestCheckResourceAttr(resourceName, "tags.framework", "finos-sdlc"),
				),
			},
			// Step 3: Clear tags with an explicit empty map
			{
				Config: testAccControlResourceConfigTags(rIdentifier, `{}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
				),
			},
		},
	})
}

// TestAccControlResource_import tests terraform import functionality.
func TestAccControlResource_import(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_control.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlResourceConfigFull(rIdentifier, "Binary provenance", "Imported control", "https://example.com/docs"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "identifier", rIdentifier),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        rIdentifier,
				ImportStateVerifyIdentifierAttribute: "identifier",
			},
		},
	})
}

// TestAccControlResource_identity verifies the resource identity is stored in
// state and that a control can be imported via an identity-based import block.
func TestAccControlResource_identity(t *testing.T) {
	rIdentifier := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_control.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccControlsPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			// Resource identity requires Terraform 1.12+.
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlResourceConfig(rIdentifier, "Binary provenance"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(resourceName, map[string]knownvalue.Check{
						"identifier": knownvalue.StringExact(rIdentifier),
					}),
				},
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateKind:                      resource.ImportBlockWithResourceIdentity,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "identifier",
			},
		},
	})
}

// TestAccControlResource_forceRecreate verifies that changing the identifier
// forces recreation.
func TestAccControlResource_forceRecreate(t *testing.T) {
	rIdentifier1 := acctest.RandomWithPrefix("tf-acc-test")
	rIdentifier2 := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_control.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccControlsPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlResourceConfig(rIdentifier1, "Binary provenance"),
				Check:  resource.TestCheckResourceAttr(resourceName, "identifier", rIdentifier1),
			},
			{
				Config: testAccControlResourceConfig(rIdentifier2, "Binary provenance"),
				Check:  resource.TestCheckResourceAttr(resourceName, "identifier", rIdentifier2),
			},
		},
	})
}

func testAccControlResourceConfig(identifier, name string) string {
	return fmt.Sprintf(`
resource "kosli_control" "test" {
  identifier = %[1]q
  name       = %[2]q
}
`, identifier, name)
}

func testAccControlResourceConfigTags(identifier, tags string) string {
	return fmt.Sprintf(`
resource "kosli_control" "test" {
  identifier = %[1]q
  name       = "Binary provenance"
  tags       = %[2]s
}
`, identifier, tags)
}

func testAccControlResourceConfigFull(identifier, name, description, docsLink string) string {
	return fmt.Sprintf(`
resource "kosli_control" "test" {
  identifier  = %[1]q
  name        = %[2]q
  description = %[3]q
  links = {
    docs = %[4]q
  }
}
`, identifier, name, description, docsLink)
}
