package provider

import (
	"fmt"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccEnvironmentResource_basic tests minimal required configuration
func TestAccEnvironmentResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "type", "K8S"),
					resource.TestCheckResourceAttr(resourceName, "include_scaling", "false"),
				),
			},
		},
	})
}

// TestAccEnvironmentResource_full tests all attributes including optional fields
func TestAccEnvironmentResource_full(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_environment.test"
	description := "Test environment for acceptance testing"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfigFull(rName, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "type", "ECS"),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "include_scaling", "true"),
				),
			},
		},
	})
}

// TestAccEnvironmentResource_update tests resource updates
func TestAccEnvironmentResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_environment.test"
	description1 := "Initial description"
	description2 := "Updated description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial configuration
			{
				Config: testAccEnvironmentResourceConfigFull(rName, description1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description1),
					resource.TestCheckResourceAttr(resourceName, "include_scaling", "true"),
				),
			},
			// Step 2: Update description and include_scaling
			{
				Config: testAccEnvironmentResourceConfigUpdate(rName, description2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description2),
					resource.TestCheckResourceAttr(resourceName, "include_scaling", "false"),
				),
			},
		},
	})
}

// TestAccEnvironmentResource_import tests terraform import functionality
func TestAccEnvironmentResource_import(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create resource
			{
				Config: testAccEnvironmentResourceConfig(rName),
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

// TestAccEnvironmentResource_forceRecreate tests ForceNew behavior when name or type changes
func TestAccEnvironmentResource_forceRecreate(t *testing.T) {
	rName1 := acctest.RandomWithPrefix("tf-acc-test")
	rName2 := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial name and type
			{
				Config: testAccEnvironmentResourceConfig(rName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName1),
					resource.TestCheckResourceAttr(resourceName, "type", "K8S"),
				),
			},
			// Step 2: Change name (ForceNew should trigger recreation)
			{
				Config: testAccEnvironmentResourceConfig(rName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName2),
					resource.TestCheckResourceAttr(resourceName, "type", "K8S"),
				),
			},
			// Step 3: Change type (ForceNew should trigger recreation)
			{
				Config: testAccEnvironmentResourceConfigType(rName2, "docker"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName2),
					resource.TestCheckResourceAttr(resourceName, "type", "docker"),
				),
			},
		},
	})
}

// TestAccEnvironmentResource_types tests different environment types
func TestAccEnvironmentResource_types(t *testing.T) {
	types := []string{"K8S", "ECS", "S3", "docker", "server", "lambda"}

	for _, envType := range types {
		t.Run(envType, func(t *testing.T) {
			rName := acctest.RandomWithPrefix(fmt.Sprintf("tf-acc-test-%s", envType))
			resourceName := "kosli_environment.test"

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccEnvironmentResourceConfigType(rName, envType),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(resourceName, "name", rName),
							resource.TestCheckResourceAttr(resourceName, "type", envType),
						),
					},
				},
			})
		})
	}
}

// TestAccEnvironmentResource_optionalDescription tests creating resource without description
func TestAccEnvironmentResource_optionalDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "type", "K8S"),
					resource.TestCheckNoResourceAttr(resourceName, "description"),
				),
			},
		},
	})
}

// testAccEnvironmentResourceConfig returns basic configuration (name and type only)
func testAccEnvironmentResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name = %[1]q
  type = "K8S"
}
`, name)
}

// testAccEnvironmentResourceConfigFull returns full configuration with all attributes
func testAccEnvironmentResourceConfigFull(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name            = %[1]q
  type            = "ECS"
  description     = %[2]q
  include_scaling = true
}
`, name, description)
}

// testAccEnvironmentResourceConfigUpdate returns updated configuration
func testAccEnvironmentResourceConfigUpdate(name, description string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name            = %[1]q
  type            = "ECS"
  description     = %[2]q
  include_scaling = false
}
`, name, description)
}

// testAccEnvironmentResourceConfigType returns configuration with specified type
func testAccEnvironmentResourceConfigType(name, envType string) string {
	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name = %[1]q
  type = %[2]q
}
`, name, envType)
}

// TestAccEnvironmentResource_tags tests tags CRUD lifecycle
func TestAccEnvironmentResource_tags(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with tags
			{
				Config: testAccEnvironmentResourceConfigWithTags(rName, map[string]string{
					"env":  "test",
					"team": "platform",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "type", "K8S"),
					resource.TestCheckResourceAttr(resourceName, "tags.env", "test"),
					resource.TestCheckResourceAttr(resourceName, "tags.team", "platform"),
				),
			},
			// Step 2: Update tags (modify value and add a key)
			{
				Config: testAccEnvironmentResourceConfigWithTags(rName, map[string]string{
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
				Config: testAccEnvironmentResourceConfigWithTags(rName, map[string]string{}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
				),
			},
		},
	})
}

// TestAccEnvironmentResource_tagsImport tests that tags are preserved through import
func TestAccEnvironmentResource_tagsImport(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kosli_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with tags
			{
				Config: testAccEnvironmentResourceConfigWithTags(rName, map[string]string{
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

// testAccEnvironmentResourceConfigWithTags returns configuration with specified tags
func testAccEnvironmentResourceConfigWithTags(name string, tags map[string]string) string {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tagsHCL := "  tags = {\n"
	for _, k := range keys {
		tagsHCL += fmt.Sprintf("    %q = %q\n", k, tags[k])
	}
	tagsHCL += "  }\n"

	return fmt.Sprintf(`
resource "kosli_environment" "test" {
  name = %[1]q
  type = "K8S"
%[2]s}
`, name, tagsHCL)
}
