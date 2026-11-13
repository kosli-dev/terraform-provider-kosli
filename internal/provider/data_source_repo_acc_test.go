package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRepoDataSource_byNameAndID(t *testing.T) {
	repo := testAccReportRepo(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "kosli_repo" "by_name" {
  name = %[1]q
}

data "kosli_repo" "by_name_and_provider" {
  name         = %[1]q
  vcs_provider = "github"
}

data "kosli_repo" "by_id" {
  id = %[2]q
}
`, repo.Name, repo.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kosli_repo.by_name", "id", repo.ID),
					resource.TestCheckResourceAttr("data.kosli_repo.by_name", "url", "https://github.com/"+repo.Name),
					resource.TestCheckResourceAttr("data.kosli_repo.by_name", "vcs_provider", "github"),
					resource.TestCheckResourceAttr("data.kosli_repo.by_name", "tags.%", "0"),
					resource.TestCheckResourceAttr("data.kosli_repo.by_name_and_provider", "id", repo.ID),
					resource.TestCheckResourceAttr("data.kosli_repo.by_id", "name", repo.Name),
				),
			},
		},
	})
}

func TestAccRepoDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "kosli_repo" "test" {
  name = "tf-acc-test/no-such-repo"
}
`,
				ExpectError: regexp.MustCompile(`Repo Not Found`),
			},
		},
	})
}

func TestAccRepoDataSource_requiresIDOrName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "kosli_repo" "test" {
  vcs_provider = "github"
}
`,
				ExpectError: regexp.MustCompile(`Missing Attribute Configuration`),
			},
		},
	})
}
