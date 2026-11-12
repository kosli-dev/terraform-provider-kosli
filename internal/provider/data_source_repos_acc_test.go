package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccReposDataSource_filters(t *testing.T) {
	repo := testAccReportRepo(t)
	dataSourceName := "data.kosli_repos.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "kosli_repos" "test" {
  name = %q
}
`, repo.Name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "repos.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "repos.0.id", repo.ID),
					resource.TestCheckResourceAttr(dataSourceName, "repos.0.name", repo.Name),
					resource.TestCheckResourceAttr(dataSourceName, "repos.0.url", "https://github.com/"+repo.Name),
					resource.TestCheckResourceAttr(dataSourceName, "repos.0.provider", "github"),
					resource.TestCheckResourceAttr(dataSourceName, "repos.0.tags.%", "0"),
				),
			},
			{
				Config: fmt.Sprintf(`
data "kosli_repos" "test" {
  namespace    = %q
  search       = %q
  vcs_provider = "github"
}
`, testAccRepoNamespace, repo.ShortName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "repos.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "repos.0.id", repo.ID),
				),
			},
			{
				Config: `
data "kosli_repos" "test" {
  name = "tf-acc-test/no-such-repo"
}
`,
				Check: resource.TestCheckResourceAttr(dataSourceName, "repos.#", "0"),
			},
		},
	})
}

// TestAccReposDataSource_tagFilter tags a repo with kosli_repo_tags and finds
// it again by that tag.
func TestAccReposDataSource_tagFilter(t *testing.T) {
	repo := testAccReportRepo(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRepoHasNoTags(repo.ID),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "kosli_repo_tags" "test" {
  repo_id = %[1]q
  tags    = { "tf-acc-run" = %[2]q }
}

data "kosli_repos" "test" {
  tag        = ["tf-acc-run:%[2]s"]
  depends_on = [kosli_repo_tags.test]
}
`, repo.ID, repo.ShortName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kosli_repos.test", "repos.#", "1"),
					resource.TestCheckResourceAttr("data.kosli_repos.test", "repos.0.id", repo.ID),
					resource.TestCheckResourceAttr("data.kosli_repos.test", "repos.0.tags.tf-acc-run", repo.ShortName),
				),
			},
		},
	})
}

func TestAccReposDataSource_nameConflictsWithSearch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "kosli_repos" "test" {
  name   = "acme/api"
  search = "api"
}
`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}
