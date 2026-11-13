package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// testAccRepoNamespace is the namespace of every repo the acceptance tests
// report, so the namespace filter has something to match.
const testAccRepoNamespace = "tf-acc-test"

func testAccClient() (*client.Client, error) {
	var opts []client.ClientOption
	if apiURL := os.Getenv("KOSLI_API_URL"); apiURL != "" {
		opts = append(opts, client.WithBaseURL(apiURL))
	}
	return client.NewClient(os.Getenv("KOSLI_API_TOKEN"), os.Getenv("KOSLI_ORG"), opts...)
}

type testAccRepo struct {
	ID        string
	Name      string
	ShortName string
}

// beginTrailRequest is the multipart body of PUT /api/v2/trails/{org}/{flow}.
type beginTrailRequest struct {
	DataJSON map[string]any
}

func (req *beginTrailRequest) MarshalMultipart() (io.Reader, string, error) {
	if req == nil {
		return nil, "", fmt.Errorf("nil request")
	}
	payload, err := json.Marshal(req.DataJSON)
	if err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("data_json", string(payload)); err != nil {
		return nil, "", err
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return &buf, writer.FormDataContentType(), nil
}

// testAccReportRepo makes Kosli record a new repo, the way a pipeline does,
// by beginning a trail that references it. Repos cannot be deleted, so each
// run leaves one behind; a fresh repo per run keeps the authoritative tags
// of one run from fighting another's. The auto-created flow is archived on
// cleanup.
func testAccReportRepo(t *testing.T) testAccRepo {
	t.Helper()
	// Runs before resource.Test, so it must skip as resource.Test does.
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
	}
	testAccPreCheck(t)
	ctx := context.Background()

	c, err := testAccClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	flowName := acctest.RandomWithPrefix("tf-acc-test")
	shortName := acctest.RandomWithPrefix("tf-acc-repo")
	repoName := testAccRepoNamespace + "/" + shortName

	req := &beginTrailRequest{DataJSON: map[string]any{
		"name": "repo-tags",
		"repo_info": map[string]any{
			"name":     repoName,
			"id":       acctest.RandString(12),
			"url":      "https://github.com/" + repoName,
			"provider": "github",
		},
	}}
	resp, err := c.Put(ctx, "/trails/"+c.Organization()+"/"+flowName, req)
	if err != nil {
		t.Fatalf("failed to begin trail reporting repo %q: %v", repoName, err)
	}
	resp.Body.Close()
	t.Cleanup(func() {
		if err := c.ArchiveFlow(context.Background(), flowName); err != nil {
			t.Logf("failed to archive flow %q: %v", flowName, err)
		}
	})

	repos, err := c.ListRepos(ctx, &client.ListReposOptions{Name: repoName})
	if err != nil {
		t.Fatalf("failed to look up repo %q: %v", repoName, err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected Kosli to record exactly one repo named %q, got %d", repoName, len(repos))
	}
	return testAccRepo{ID: repos[0].ID, Name: repoName, ShortName: shortName}
}

func testAccSetRepoTag(t *testing.T, repoID, key, value string) {
	t.Helper()
	c, err := testAccClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	payload := &client.TagResourcePayload{SetTags: map[string]string{key: value}, RemoveTags: []string{}}
	if err := c.TagResource(context.Background(), "repo", repoID, payload); err != nil {
		t.Fatalf("failed to tag repo %q: %v", repoID, err)
	}
}

func testAccCheckRepoHasNoTags(repoID string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		c, err := testAccClient()
		if err != nil {
			return err
		}
		repo, err := c.GetRepo(context.Background(), repoID)
		if err != nil {
			return err
		}
		if len(repo.Tags) != 0 {
			return fmt.Errorf("expected repo %q to have no tags after destroy, got %v", repoID, repo.Tags)
		}
		return nil
	}
}

func TestAccRepoTagsResource_basic(t *testing.T) {
	repo := testAccReportRepo(t)
	resourceName := "kosli_repo_tags.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRepoHasNoTags(repo.ID),
		Steps: []resource.TestStep{
			{
				Config: testAccRepoTagsResourceConfig(repo.ID, map[string]string{"team": "platform", "tier": "1"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "repo_id", repo.ID),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.team", "platform"),
					resource.TestCheckResourceAttr(resourceName, "tags.tier", "1"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateId:                        repo.ID,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "repo_id",
			},
			{
				Config: testAccRepoTagsResourceConfig(repo.ID, map[string]string{"team": "payments"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.team", "payments"),
				),
			},
			{
				Config: testAccRepoTagsResourceConfig(repo.ID, map[string]string{}),
				Check:  resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
			},
		},
	})
}

// TestAccRepoTagsResource_removesUnmanagedTags verifies tags is authoritative:
// a tag set outside Terraform shows as drift and the next apply removes it.
func TestAccRepoTagsResource_removesUnmanagedTags(t *testing.T) {
	repo := testAccReportRepo(t)
	resourceName := "kosli_repo_tags.test"
	config := testAccRepoTagsResourceConfig(repo.ID, map[string]string{"team": "platform"})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRepoHasNoTags(repo.ID),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
			},
			{
				PreConfig:          func() { testAccSetRepoTag(t, repo.ID, "manual", "ui") },
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "tags.manual"),
				),
			},
		},
	})
}

func TestAccRepoTagsResource_unknownRepo(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRepoTagsResourceConfig("tf-acc-test-no-such-repo", map[string]string{"team": "platform"}),
				ExpectError: regexp.MustCompile(`Repo Not Found`),
			},
		},
	})
}

func testAccRepoTagsResourceConfig(repoID string, tags map[string]string) string {
	return fmt.Sprintf(`
resource "kosli_repo_tags" "test" {
  repo_id = %q
  tags    = %s
}
`, repoID, hclStringMap(tags))
}

// hclStringMap renders tags as an HCL map literal.
func hclStringMap(tags map[string]string) string {
	encoded, _ := json.Marshal(tags)
	return string(encoded)
}
