package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccFlowResource_basic tests minimal required configuration (name only)
func TestAccFlowResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_flow.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "visibility", "private"),
					resource.TestCheckNoResourceAttr(resourceName, "description"),
					resource.TestCheckResourceAttrSet(resourceName, "template"),
				),
			},
		},
	})
}

// TestAccFlowResource_full tests all attributes including description, visibility, and template
func TestAccFlowResource_full(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_flow.test"
	description := "CD pipeline for acceptance testing"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowResourceConfigFull(rName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "visibility", "public"),
				),
			},
		},
	})
}

// TestAccFlowResource_withTemplate tests creating a flow with a YAML template
func TestAccFlowResource_withTemplate(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_flow.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowResourceConfigWithTemplate(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "visibility", "private"),
					resource.TestCheckResourceAttrSet(resourceName, "template"),
				),
			},
		},
	})
}

// TestAccFlowResource_update tests resource updates
func TestAccFlowResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_flow.test"
	description1 := "Initial description"
	description2 := "Updated description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial configuration
			{
				Config: testAccFlowResourceConfigFull(rName, description1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description1),
					resource.TestCheckResourceAttr(resourceName, "visibility", "public"),
				),
			},
			// Step 2: Update description and change visibility to private
			{
				Config: testAccFlowResourceConfigUpdate(rName, description2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description2),
					resource.TestCheckResourceAttr(resourceName, "visibility", "private"),
				),
			},
			// Step 3: Add a template
			{
				Config: testAccFlowResourceConfigUpdateWithTemplate(rName, description2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description2),
					resource.TestCheckResourceAttrSet(resourceName, "template"),
				),
			},
		},
	})
}

// TestAccFlowResource_import tests terraform import functionality
func TestAccFlowResource_import(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_flow.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create resource
			{
				Config: testAccFlowResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
				),
			},
			// Step 2: Import by name and verify state matches
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        rName,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

// TestAccFlowResource_forceRecreate tests ForceNew behavior when name changes
func TestAccFlowResource_forceRecreate(t *testing.T) {
	rName1 := acctest.RandomWithPrefix("tf-acc-test")
	rName2 := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_flow.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial name
			{
				Config: testAccFlowResourceConfig(rName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName1),
				),
			},
			// Step 2: Change name (ForceNew should trigger recreation)
			{
				Config: testAccFlowResourceConfig(rName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName2),
				),
			},
		},
	})
}

// TestAccFlowResource_optionalDescription tests creating a flow without description
func TestAccFlowResource_optionalDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_flow.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckNoResourceAttr(resourceName, "description"),
				),
			},
		},
	})
}

// TestAccFlowResource_publicVisibility tests creating a flow with public visibility
func TestAccFlowResource_publicVisibility(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_flow.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowResourceConfigPublic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "visibility", "public"),
				),
			},
		},
	})
}

// testAccFlowResourceConfig returns basic configuration (name only)
func testAccFlowResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name = %[1]q
}
`, name)
}

// testAccFlowResourceConfigFull returns full configuration with all attributes and template
func testAccFlowResourceConfigFull(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name        = %[1]q
  description = %[2]q
  visibility  = "public"
}
`, name, description)
}

// testAccFlowResourceConfigWithTemplate returns configuration with a YAML template
func testAccFlowResourceConfigWithTemplate(name string) string {
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name     = %[1]q
  template = <<-YAML
version: 1
trail:
  attestations:
    - name: unit-tests
      type: generic
  artifacts:
    - name: binary
      attestations:
        - name: sbom
          type: generic
YAML
}
`, name)
}

// testAccFlowResourceConfigUpdate returns updated configuration
func testAccFlowResourceConfigUpdate(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name        = %[1]q
  description = %[2]q
  visibility  = "private"
}
`, name, description)
}

// testAccFlowResourceConfigUpdateWithTemplate returns updated configuration with template
func testAccFlowResourceConfigUpdateWithTemplate(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name        = %[1]q
  description = %[2]q
  visibility  = "private"
  template    = <<-YAML
version: 1
trail:
  attestations:
    - name: pull-request
      type: pull_request
    - name: unit-tests
      type: generic
YAML
}
`, name, description)
}

// testAccFlowResourceConfigPublic returns configuration with public visibility
func testAccFlowResourceConfigPublic(name string) string {
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name       = %[1]q
  visibility = "public"
}
`, name)
}
