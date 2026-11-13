package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &repoDataSource{}
var _ datasource.DataSourceWithConfigValidators = &repoDataSource{}

// NewRepoDataSource creates a new repo data source.
func NewRepoDataSource() datasource.DataSource {
	return &repoDataSource{}
}

// repoDataSource defines the data source implementation.
type repoDataSource struct {
	client *client.Client
}

// repoDataSourceModel is an alias for repoElementModel: one repo exposes the
// same fields as each kosli_repos result, so repoToElement maps both.
type repoDataSourceModel = repoElementModel

// Metadata returns the data source type name.
func (d *repoDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repo"
}

// Schema defines the schema for the data source.
func (d *repoDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches one Kosli repo, by its internal ID or by its name. Kosli records a repo when an attestation or trail references it. " +
			"Use the returned `id` with `kosli_repo_tags` to tag the repo, or `kosli_repos` to find several repos at once.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Internal Kosli ID of the repo. Set either `id` or `name`.",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Full name of the repo, including its namespace, e.g. `acme/platform/api`. Set either `id` or `name`. " +
					"Names can repeat across VCS providers; set `vcs_provider` when more than one repo has this name.",
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"vcs_provider": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "VCS provider of the repo, e.g. `github` or `gitlab`. When set, narrows a lookup by `name`; conflicts with `id`. " +
					"When not set, Kosli's value is returned, or null when Kosli could not determine it.",
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL of the repo.",
			},
			"vcs_instance": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "VCS host of the repo. Null when Kosli could not determine it.",
			},
			"tags": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Key-value pairs tagging the repo.",
			},
		},
	}
}

// ConfigValidators requires exactly one of id and name. A lookup by id ignores
// vcs_provider, so the two conflict.
func (d *repoDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
		datasourcevalidator.Conflicting(path.MatchRoot("id"), path.MatchRoot("vcs_provider")),
	}
}

// Configure adds the provider configured client to the data source.
func (d *repoDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

// Read refreshes the Terraform state with the latest data.
func (d *repoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx = dataSourceOperation(ctx, d)
	var config repoDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var repo *client.Repo
	if !config.ID.IsNull() {
		repo = d.readByID(ctx, config.ID.ValueString(), resp)
	} else {
		repo = d.readByName(ctx, config.Name.ValueString(), config.VCSProvider.ValueString(), resp)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	data := repoToElement(ctx, *repo, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	// A configured value must survive into state, and the API may store a
	// finer provider (bitbucket_cloud) than the one filtered on (bitbucket).
	if !config.VCSProvider.IsNull() {
		data.VCSProvider = config.VCSProvider
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *repoDataSource) readByID(ctx context.Context, id string, resp *datasource.ReadResponse) *client.Repo {
	repo, err := d.client.GetRepo(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError(
				"Repo Not Found",
				fmt.Sprintf("No repo with ID %q exists in the organization.", id),
			)
			return nil
		}
		resp.Diagnostics.AddError(
			"Error Reading Repo",
			fmt.Sprintf("Could not read repo %q: %s", id, err.Error()),
		)
		return nil
	}
	return repo
}

// readByName uses the list endpoint's exact-name filter, which avoids putting
// a '/'-separated name into the request path.
func (d *repoDataSource) readByName(ctx context.Context, name, vcsProvider string, resp *datasource.ReadResponse) *client.Repo {
	repos, err := d.client.ListRepos(ctx, &client.ListReposOptions{Name: name, Provider: vcsProvider})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Repo",
			fmt.Sprintf("Could not look up repo %q: %s", name, err.Error()),
		)
		return nil
	}

	switch len(repos) {
	case 1:
		return &repos[0]
	case 0:
		detail := fmt.Sprintf("No repo named %q exists in the organization", name)
		if vcsProvider != "" {
			detail += fmt.Sprintf(" with VCS provider %q", vcsProvider)
		}
		resp.Diagnostics.AddError("Repo Not Found", detail+". Kosli records a repo when an attestation or trail references it.")
		return nil
	default:
		providers := make([]string, 0, len(repos))
		for _, r := range repos {
			p := r.Provider
			if p == "" {
				p = "unknown"
			}
			providers = append(providers, p)
		}
		sort.Strings(providers)
		resp.Diagnostics.AddError(
			"Ambiguous Repo Name",
			fmt.Sprintf("%d repos are named %q (VCS providers: %s). Set vcs_provider to choose one, or look the repo up by id.",
				len(repos), name, strings.Join(providers, ", ")),
		)
		return nil
	}
}
