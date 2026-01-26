package provider

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

const (
	// DefaultAPIURL is the default Kosli API URL (EU region).
	DefaultAPIURL = "https://app.kosli.com"

	// DefaultTimeout is the default HTTP timeout in seconds.
	DefaultTimeout = 30
)

// Ensure KosliProvider satisfies various provider interfaces.
var _ provider.Provider = &KosliProvider{}

// KosliProvider defines the provider implementation.
type KosliProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance tests.
	version string
}

// KosliProviderModel describes the provider data model.
type KosliProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
	Org      types.String `tfsdk:"org"`
	APIURL   types.String `tfsdk:"api_url"`
	Timeout  types.Int64  `tfsdk:"timeout"`
}

// Metadata returns the provider type name.
func (p *KosliProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kosli"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *KosliProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Kosli resources using Terraform. The Kosli provider allows you to define and manage Kosli custom attestation types as Infrastructure-as-Code.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Description: "Kosli API token for authentication. Can also be set via KOSLI_API_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"org": schema.StringAttribute{
				Description: "Kosli organization name. Can also be set via KOSLI_ORG environment variable.",
				Optional:    true,
			},
			"api_url": schema.StringAttribute{
				Description: "Kosli API endpoint URL. Defaults to https://app.kosli.com (EU region). Use https://app.us.kosli.com for US region. Can also be set via KOSLI_API_URL environment variable.",
				Optional:    true,
			},
			"timeout": schema.Int64Attribute{
				Description: "HTTP client timeout in seconds. Defaults to 30 seconds.",
				Optional:    true,
			},
		},
	}
}

// Configure prepares a Kosli API client for data sources and resources.
func (p *KosliProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config KosliProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve configuration values with environment variable fallbacks
	apiToken := getConfigValue(config.APIToken, "KOSLI_API_TOKEN")
	org := getConfigValue(config.Org, "KOSLI_ORG")
	apiURL := getConfigValue(config.APIURL, "KOSLI_API_URL")

	// Set default API URL if not provided
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}

	// Validate required fields
	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API Token",
			"The provider requires an API token. Set the api_token attribute in the provider configuration or the KOSLI_API_TOKEN environment variable.",
		)
	}

	if org == "" {
		resp.Diagnostics.AddError(
			"Missing Organization",
			"The provider requires an organization name. Set the org attribute in the provider configuration or the KOSLI_ORG environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Determine timeout
	timeout := DefaultTimeout * time.Second
	if !config.Timeout.IsNull() {
		timeout = time.Duration(config.Timeout.ValueInt64()) * time.Second
	}

	// Create API client with options
	var opts []client.ClientOption

	// Set base URL if not default
	if apiURL != DefaultAPIURL {
		opts = append(opts, client.WithBaseURL(apiURL))
	}

	// Set timeout
	opts = append(opts, client.WithTimeout(timeout))

	// Set user agent with provider version
	userAgent := fmt.Sprintf("terraform-provider-kosli/%s", p.version)
	opts = append(opts, client.WithUserAgent(userAgent))

	kosliClient, err := client.NewClient(apiToken, org, opts...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Kosli API Client",
			fmt.Sprintf("An unexpected error occurred when creating the Kosli API client: %s", err.Error()),
		)
		return
	}

	// Make the client available to resources and data sources
	resp.DataSourceData = kosliClient
	resp.ResourceData = kosliClient
}

// Resources defines the resources implemented in the provider.
func (p *KosliProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCustomAttestationTypeResource,
		NewEnvironmentResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *KosliProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCustomAttestationTypeDataSource,
		NewEnvironmentDataSource,
	}
}

// New returns a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &KosliProvider{
			version: version,
		}
	}
}

// getConfigValue returns the value from the config if set, otherwise falls back to environment variable.
func getConfigValue(configValue types.String, envVar string) string {
	if !configValue.IsNull() && configValue.ValueString() != "" {
		return configValue.ValueString()
	}
	return os.Getenv(envVar)
}
