package provider

import (
	"fmt"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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
					resource.TestCheckNoResourceAttr(resourceName, "description"),
					resource.TestCheckResourceAttrSet(resourceName, "template"),
				),
			},
		},
	})
}

// TestAccFlowResource_full tests all attributes including description, and template
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
				),
			},
			// Step 2: Update description
			{
				Config: testAccFlowResourceConfigUpdate(rName, description2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description2),
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

// TestAccFlowResource_rename tests that changing name renames the flow in
// place (an Update, not a replace), keeps the other attributes, leaves the old
// name resolving to the renamed flow, and can be combined with other changes.
func TestAccFlowResource_rename(t *testing.T) {
	rName1 := acctest.RandomWithPrefix("tf-acc-test")
	rName2 := acctest.RandomWithPrefix("tf-acc-test")
	rName3 := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_flow.test"
	description1 := "Flow renamed in place"
	description2 := "Flow renamed in place, description updated"
	tags := map[string]string{"managed-by": "terraform", "team": "platform"}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial name, description and tags
			{
				Config: testAccFlowResourceConfigRename(rName1, description1, tags),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName1),
					resource.TestCheckResourceAttr(resourceName, "description", description1),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
				),
			},
			// Step 2: Change only the name. The plan must be an in-place update,
			// the other attributes must survive, and the old name must still
			// resolve to the renamed flow (read via the data source after apply).
			{
				Config: testAccFlowResourceConfigRename(rName2, description1, tags) + testAccFlowDataSourceByOldName(rName1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName2),
					resource.TestCheckResourceAttr(resourceName, "description", description1),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.managed-by", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "tags.team", "platform"),
					resource.TestCheckResourceAttr("data.kosli_flow.old_name", "description", description1),
				),
			},
			// Step 3: Rename again together with a description change, still in place
			{
				Config: testAccFlowResourceConfigRename(rName3, description2, tags),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName3),
					resource.TestCheckResourceAttr(resourceName, "description", description2),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
				),
			},
			// Step 4: Import by the current name and verify state matches
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        rName3,
				ImportStateVerifyIdentifierAttribute: "name",
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

// testAccFlowResourceConfig returns basic configuration (name only)
func testAccFlowResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name = %[1]q
}
`, name)
}

// testAccFlowResourceConfigFull returns configuration with name and description
func testAccFlowResourceConfigFull(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name        = %[1]q
  description = %[2]q
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
}
`, name, description)
}

// testAccFlowResourceConfigUpdateWithTemplate returns updated configuration with template
func testAccFlowResourceConfigUpdateWithTemplate(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name        = %[1]q
  description = %[2]q
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

// TestAccFlowResource_tags tests the full tags CRUD lifecycle
func TestAccFlowResource_tags(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_flow.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with tags
			{
				Config: testAccFlowResourceConfigWithTags(rName, map[string]string{
					"env":  "test",
					"team": "platform",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "tags.env", "test"),
					resource.TestCheckResourceAttr(resourceName, "tags.team", "platform"),
				),
			},
			// Step 2: Update tags (modify value and add a key)
			{
				Config: testAccFlowResourceConfigWithTags(rName, map[string]string{
					"env":     "staging",
					"team":    "platform",
					"managed": "terraform",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.env", "staging"),
					resource.TestCheckResourceAttr(resourceName, "tags.team", "platform"),
					resource.TestCheckResourceAttr(resourceName, "tags.managed", "terraform"),
				),
			},
			// Step 3: Remove all tags by setting explicit empty map
			{
				Config: testAccFlowResourceConfigWithTags(rName, map[string]string{}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
				),
			},
		},
	})
}

// TestAccFlowResource_tagsImport tests that tags are preserved through import
func TestAccFlowResource_tagsImport(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_flow.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with tags
			{
				Config: testAccFlowResourceConfigWithTags(rName, map[string]string{
					"managed-by": "terraform",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.managed-by", "terraform"),
				),
			},
			// Step 2: Import and verify tags are preserved
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

// testAccFlowResourceConfigRename returns a configuration with name, description and tags,
// used to verify that everything but the name survives an in-place rename.
func testAccFlowResourceConfigRename(name, description string, tags map[string]string) string {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tagLines := ""
	for _, k := range keys {
		tagLines += fmt.Sprintf("    %s = %q\n", k, tags[k])
	}
	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name        = %[1]q
  description = %[2]q
  tags = {
%[3]s  }
}
`, name, description, tagLines)
}

// testAccFlowDataSourceByOldName reads a flow by a name it was renamed from.
// depends_on defers the read until after the rename is applied.
func testAccFlowDataSourceByOldName(oldName string) string {
	return fmt.Sprintf(`
data "kosli_flow" "old_name" {
  name       = %[1]q
  depends_on = [kosli_flow.test]
}
`, oldName)
}

// testAccFlowResourceConfigWithTags returns configuration with specified tags
func testAccFlowResourceConfigWithTags(name string, tags map[string]string) string {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tagsHCL := "  tags = {\n"
	for _, k := range keys {
		tagsHCL += fmt.Sprintf("    %s = %q\n", k, tags[k])
	}
	tagsHCL += "  }\n"

	return fmt.Sprintf(`
resource "kosli_flow" "test" {
  name = %[1]q
%[2]s}
`, name, tagsHCL)
}
