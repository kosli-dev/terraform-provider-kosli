package provider

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &reposDataSource{}
var _ datasource.DataSourceWithConfigValidators = &reposDataSource{}

// NewReposDataSource creates a new repos data source.
func NewReposDataSource() datasource.DataSource {
	return &reposDataSource{}
}

// reposDataSource defines the data source implementation.
type reposDataSource struct {
	client *client.Client
}

// reposDataSourceModel describes the data source data model.
type reposDataSourceModel struct {
	Name        types.String       `tfsdk:"name"`
	Search      types.String       `tfsdk:"search"`
	VCSProvider types.String       `tfsdk:"vcs_provider"`
	VCSInstance types.String       `tfsdk:"vcs_instance"`
	Namespace   types.String       `tfsdk:"namespace"`
	TagFilters  []tagFilterModel   `tfsdk:"tag_filter"`
	Repos       []repoElementModel `tfsdk:"repos"`
}

// tagFilterModel is one tag_filter block.
type tagFilterModel struct {
	Key    types.String `tfsdk:"key"`
	Values types.List   `tfsdk:"values"`
}

// repoElementModel is a single repo entry in the repos list.
type repoElementModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	URL         types.String `tfsdk:"url"`
	VCSProvider types.String `tfsdk:"vcs_provider"`
	VCSInstance types.String `tfsdk:"vcs_instance"`
	Tags        types.Map    `tfsdk:"tags"`
}

// Metadata returns the data source type name.
func (d *reposDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repos"
}

// Schema defines the schema for the data source.
func (d *reposDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Kosli repos matching the given filters. Kosli records a repo when an attestation or trail references it. " +
			"Use the returned `id` values with `kosli_repo_tags` to tag repos.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Return only repos with exactly this name, e.g. `acme/platform/api`. Conflicts with `search`.",
			},
			"search": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Return only repos whose name contains this string (case-insensitive). Conflicts with `name`.",
			},
			"vcs_provider": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Return only repos hosted by this VCS provider, e.g. `github`, `gitlab`, `bitbucket` or `azure-devops`.",
			},
			"vcs_instance": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Return only repos on this VCS host, e.g. `gitlab.company.com`.",
			},
			"namespace": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Return only repos nested anywhere under this `/`-separated namespace, e.g. `acme/platform`.",
			},
			"repos": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The matching repos, sorted by name and then by `id`. Empty when no repo matches.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Internal Kosli ID of the repo. Use it as `repo_id` on `kosli_repo_tags`.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Full name of the repo, including its namespace. Names can repeat across VCS providers.",
						},
						"url": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "URL of the repo.",
						},
						"vcs_provider": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "VCS provider of the repo. Null when Kosli could not determine it.",
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
				},
			},
		},

		Blocks: map[string]schema.Block{
			"tag_filter": schema.ListNestedBlock{
				MarkdownDescription: "Return only repos with a matching tag. Repos must match every `tag_filter` block.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Tag key the repo must have.",
							Validators: []validator.String{
								// The API splits each tag filter on its first colon.
								stringvalidator.RegexMatches(regexp.MustCompile(`^[^:]+$`), "must be non-empty and must not contain a colon"),
							},
						},
						"values": schema.ListAttribute{
							Optional:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Values the tag may have; a repo matches if its tag has any of them. Omit to match the key with any value.",
							Validators: []validator.List{
								listvalidator.NoNullValues(),
								listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
							},
						},
					},
				},
			},
		},
	}
}

// ConfigValidators rejects name together with search, which the API refuses.
func (d *reposDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.Conflicting(path.MatchRoot("name"), path.MatchRoot("search")),
	}
}

// Configure adds the provider configured client to the data source.
func (d *reposDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *reposDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx = dataSourceOperation(ctx, d)
	var data reposDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := &client.ListReposOptions{
		Name:        data.Name.ValueString(),
		Search:      data.Search.ValueString(),
		Provider:    data.VCSProvider.ValueString(),
		VCSInstance: data.VCSInstance.ValueString(),
		Namespace:   data.Namespace.ValueString(),
	}
	opts.Tags = tagFilterParams(ctx, data.TagFilters, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	repos, err := d.client.ListRepos(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Repos",
			fmt.Sprintf("Could not list repos: %s", err.Error()),
		)
		return
	}

	// The API sorts by name only, and names repeat across VCS providers.
	sort.SliceStable(repos, func(i, j int) bool {
		if repos[i].Name != repos[j].Name {
			return repos[i].Name < repos[j].Name
		}
		return repos[i].ID < repos[j].ID
	})

	data.Repos = make([]repoElementModel, 0, len(repos))
	for _, repo := range repos {
		data.Repos = append(data.Repos, repoToElement(ctx, repo, &resp.Diagnostics))
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// tagFilterParams converts tag_filter blocks to the API's tag query values:
// "key" matches any value, "key:value" one value. Values of one key are ORed
// by the API and different keys ANDed, as the blocks document.
func tagFilterParams(ctx context.Context, filters []tagFilterModel, diags *diag.Diagnostics) []string {
	var params []string
	for _, f := range filters {
		var values []string
		if !f.Values.IsNull() && !f.Values.IsUnknown() {
			diags.Append(f.Values.ElementsAs(ctx, &values, false)...)
			if diags.HasError() {
				return nil
			}
		}
		if len(values) == 0 {
			params = append(params, f.Key.ValueString())
			continue
		}
		for _, v := range values {
			params = append(params, f.Key.ValueString()+":"+v)
		}
	}
	return params
}

func repoToElement(ctx context.Context, repo client.Repo, diags *diag.Diagnostics) repoElementModel {
	return repoElementModel{
		ID:          types.StringValue(repo.ID),
		Name:        types.StringValue(repo.Name),
		URL:         types.StringValue(repo.URL),
		VCSProvider: stringOrNull(repo.Provider),
		VCSInstance: stringOrNull(repo.VCSInstance),
		Tags:        repoTagsValue(ctx, repo.Tags, diags),
	}
}

func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// repoTagsValue maps nil tags to an empty map so `tags = {}` in config does
// not drift against an untagged repo.
func repoTagsValue(ctx context.Context, tags map[string]string, diags *diag.Diagnostics) types.Map {
	if tags == nil {
		tags = map[string]string{}
	}
	value, d := types.MapValueFrom(ctx, types.StringType, tags)
	diags.Append(d...)
	return value
}
