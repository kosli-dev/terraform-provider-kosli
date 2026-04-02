package provider

import (
	"fmt"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccLogicalEnvironmentResource_basic tests minimal required configuration
func TestAccLogicalEnvironmentResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical")
	envName1 := acctest.RandomWithPrefix("tf-acc-test-env1")
	envName2 := acctest.RandomWithPrefix("tf-acc-test-env2")
	resourceName := "kosli_logical_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLogicalEnvironmentResourceConfigBasic(rName, envName1, envName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "type", "logical"),
					resource.TestCheckResourceAttr(resourceName, "included_environments.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included_environments.0", envName1),
					resource.TestCheckResourceAttr(resourceName, "included_environments.1", envName2),
				),
			},
		},
	})
}

// TestAccLogicalEnvironmentResource_full tests all attributes including optional fields
func TestAccLogicalEnvironmentResource_full(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical")
	envName1 := acctest.RandomWithPrefix("tf-acc-test-env1")
	envName2 := acctest.RandomWithPrefix("tf-acc-test-env2")
	resourceName := "kosli_logical_environment.test"
	description := "Test logical environment for acceptance testing"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLogicalEnvironmentResourceConfigFull(rName, envName1, envName2, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "type", "logical"),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "included_environments.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included_environments.0", envName1),
					resource.TestCheckResourceAttr(resourceName, "included_environments.1", envName2),
				),
			},
		},
	})
}

// TestAccLogicalEnvironmentResource_update tests updating included_environments list
func TestAccLogicalEnvironmentResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical")
	envName1 := acctest.RandomWithPrefix("tf-acc-test-env1")
	envName2 := acctest.RandomWithPrefix("tf-acc-test-env2")
	envName3 := acctest.RandomWithPrefix("tf-acc-test-env3")
	resourceName := "kosli_logical_environment.test"
	description1 := "Initial description"
	description2 := "Updated description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with two environments
			{
				Config: testAccLogicalEnvironmentResourceConfigFull(rName, envName1, envName2, description1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description1),
					resource.TestCheckResourceAttr(resourceName, "included_environments.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included_environments.0", envName1),
					resource.TestCheckResourceAttr(resourceName, "included_environments.1", envName2),
				),
			},
			// Step 2: Update description and add third environment
			{
				Config: testAccLogicalEnvironmentResourceConfigUpdate(rName, envName1, envName2, envName3, description2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "description", description2),
					resource.TestCheckResourceAttr(resourceName, "included_environments.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "included_environments.0", envName1),
					resource.TestCheckResourceAttr(resourceName, "included_environments.1", envName2),
					resource.TestCheckResourceAttr(resourceName, "included_environments.2", envName3),
				),
			},
			// Step 3: Remove third environment
			{
				Config: testAccLogicalEnvironmentResourceConfigFull(rName, envName1, envName2, description2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "included_environments.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included_environments.0", envName1),
					resource.TestCheckResourceAttr(resourceName, "included_environments.1", envName2),
				),
			},
		},
	})
}

// TestAccLogicalEnvironmentResource_import tests terraform import functionality
func TestAccLogicalEnvironmentResource_import(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical")
	envName1 := acctest.RandomWithPrefix("tf-acc-test-env1")
	envName2 := acctest.RandomWithPrefix("tf-acc-test-env2")
	resourceName := "kosli_logical_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create resource
			{
				Config: testAccLogicalEnvironmentResourceConfigBasic(rName, envName1, envName2),
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

// TestAccLogicalEnvironmentResource_forceRecreate tests ForceNew behavior when name changes
func TestAccLogicalEnvironmentResource_forceRecreate(t *testing.T) {
	rName1 := acctest.RandomWithPrefix("tf-acc-test-logical1")
	rName2 := acctest.RandomWithPrefix("tf-acc-test-logical2")
	envName1 := acctest.RandomWithPrefix("tf-acc-test-env1")
	envName2 := acctest.RandomWithPrefix("tf-acc-test-env2")
	resourceName := "kosli_logical_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial name
			{
				Config: testAccLogicalEnvironmentResourceConfigBasic(rName1, envName1, envName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName1),
					resource.TestCheckResourceAttr(resourceName, "type", "logical"),
				),
			},
			// Step 2: Change name (ForceNew should trigger recreation)
			{
				Config: testAccLogicalEnvironmentResourceConfigBasic(rName2, envName1, envName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName2),
					resource.TestCheckResourceAttr(resourceName, "type", "logical"),
				),
			},
		},
	})
}

// TestAccLogicalEnvironmentResource_emptyList tests creating logical environment with empty list
func TestAccLogicalEnvironmentResource_emptyList(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical")
	resourceName := "kosli_logical_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLogicalEnvironmentResourceConfigEmpty(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "type", "logical"),
					resource.TestCheckResourceAttr(resourceName, "included_environments.#", "0"),
				),
			},
		},
	})
}

// TestAccLogicalEnvironmentResource_optionalDescription tests creating resource without description
func TestAccLogicalEnvironmentResource_optionalDescription(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical")
	envName1 := acctest.RandomWithPrefix("tf-acc-test-env1")
	envName2 := acctest.RandomWithPrefix("tf-acc-test-env2")
	resourceName := "kosli_logical_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLogicalEnvironmentResourceConfigBasic(rName, envName1, envName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "type", "logical"),
					resource.TestCheckNoResourceAttr(resourceName, "description"),
				),
			},
		},
	})
}

// testAccLogicalEnvironmentResourceConfigBasic returns basic configuration
func testAccLogicalEnvironmentResourceConfigBasic(name, env1, env2 string) string {
	return fmt.Sprintf(`
# Create prerequisite physical environments
resource "kosli_environment" "env1" {
  name = %[2]q
  type = "K8S"
}

resource "kosli_environment" "env2" {
  name = %[3]q
  type = "ECS"
}

# Create logical environment aggregating the physical ones
resource "kosli_logical_environment" "test" {
  name = %[1]q
  included_environments = [
    kosli_environment.env1.name,
    kosli_environment.env2.name,
  ]
}
`, name, env1, env2)
}

// testAccLogicalEnvironmentResourceConfigFull returns full configuration with all attributes
func testAccLogicalEnvironmentResourceConfigFull(name, env1, env2, description string) string {
	return fmt.Sprintf(`
# Create prerequisite physical environments
resource "kosli_environment" "env1" {
  name = %[2]q
  type = "K8S"
}

resource "kosli_environment" "env2" {
  name = %[3]q
  type = "ECS"
}

# Create logical environment with description
resource "kosli_logical_environment" "test" {
  name        = %[1]q
  description = %[4]q
  included_environments = [
    kosli_environment.env1.name,
    kosli_environment.env2.name,
  ]
}
`, name, env1, env2, description)
}

// testAccLogicalEnvironmentResourceConfigUpdate returns configuration with three environments
func testAccLogicalEnvironmentResourceConfigUpdate(name, env1, env2, env3, description string) string {
	return fmt.Sprintf(`
# Create prerequisite physical environments
resource "kosli_environment" "env1" {
  name = %[2]q
  type = "K8S"
}

resource "kosli_environment" "env2" {
  name = %[3]q
  type = "ECS"
}

resource "kosli_environment" "env3" {
  name = %[4]q
  type = "docker"
}

# Create logical environment with three included environments
resource "kosli_logical_environment" "test" {
  name        = %[1]q
  description = %[5]q
  included_environments = [
    kosli_environment.env1.name,
    kosli_environment.env2.name,
    kosli_environment.env3.name,
  ]
}
`, name, env1, env2, env3, description)
}

// testAccLogicalEnvironmentResourceConfigEmpty returns configuration with empty included_environments
func testAccLogicalEnvironmentResourceConfigEmpty(name string) string {
	return fmt.Sprintf(`
resource "kosli_logical_environment" "test" {
  name                  = %[1]q
  included_environments = []
}
`, name)
}

// TestAccLogicalEnvironmentResource_tags tests tags CRUD lifecycle
func TestAccLogicalEnvironmentResource_tags(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical")
	envName1 := acctest.RandomWithPrefix("tf-acc-test-env1")
	envName2 := acctest.RandomWithPrefix("tf-acc-test-env2")
	resourceName := "kosli_logical_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with tags
			{
				Config: testAccLogicalEnvironmentResourceConfigWithTags(rName, envName1, envName2, &map[string]string{
					"env":        "test",
					"managed-by": "terraform",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "type", "logical"),
					resource.TestCheckResourceAttr(resourceName, "tags.env", "test"),
					resource.TestCheckResourceAttr(resourceName, "tags.managed-by", "terraform"),
				),
			},
			// Step 2: Update tags (modify value, add key, remove key)
			{
				Config: testAccLogicalEnvironmentResourceConfigWithTags(rName, envName1, envName2, &map[string]string{
					"env":  "staging",
					"team": "platform",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.env", "staging"),
					resource.TestCheckResourceAttr(resourceName, "tags.team", "platform"),
					resource.TestCheckNoResourceAttr(resourceName, "tags.managed-by"),
				),
			},
			// Step 3: Remove all tags by setting explicit empty map
			{
				Config: testAccLogicalEnvironmentResourceConfigWithTags(rName, envName1, envName2, &map[string]string{}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
				),
			},
			// Step 4: Omit tags block entirely — Computed keeps previous value (empty map)
			{
				Config: testAccLogicalEnvironmentResourceConfigWithTags(rName, envName1, envName2, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
				),
			},
		},
	})
}

// TestAccLogicalEnvironmentResource_tagsImport tests that tags are preserved through import
func TestAccLogicalEnvironmentResource_tagsImport(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-logical")
	envName1 := acctest.RandomWithPrefix("tf-acc-test-env1")
	envName2 := acctest.RandomWithPrefix("tf-acc-test-env2")
	resourceName := "kosli_logical_environment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with tags
			{
				Config: testAccLogicalEnvironmentResourceConfigWithTags(rName, envName1, envName2, &map[string]string{
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

// testAccLogicalEnvironmentResourceConfigWithTags returns configuration with specified tags.
// Pass nil to omit the tags attribute entirely (exercises the Optional+Computed path);
// pass a non-nil empty map to emit tags = {} (exercises explicit removal).
func testAccLogicalEnvironmentResourceConfigWithTags(name, env1, env2 string, tags *map[string]string) string {
	tagsHCL := ""
	if tags != nil {
		keys := make([]string, 0, len(*tags))
		for k := range *tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		tagsHCL = "  tags = {\n"
		for _, k := range keys {
			tagsHCL += fmt.Sprintf("    %q = %q\n", k, (*tags)[k])
		}
		tagsHCL += "  }\n"
	}

	return fmt.Sprintf(`
resource "kosli_environment" "env1" {
  name = %[2]q
  type = "K8S"
}

resource "kosli_environment" "env2" {
  name = %[3]q
  type = "ECS"
}

resource "kosli_logical_environment" "test" {
  name = %[1]q
  included_environments = [
    kosli_environment.env1.name,
    kosli_environment.env2.name,
  ]
%[4]s}
`, name, env1, env2, tagsHCL)
}
