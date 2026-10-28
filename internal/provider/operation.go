package provider

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

const providerTypeName = "kosli"

// invocationID is shared by every client in the process, and Terraform
// starts a provider process per plan or apply.
var invocationID = sync.OnceValue(uuid.NewString)

// resourceOperation names the operation from the resource's type name only,
// so no configuration value reaches the X-Kosli-Operation header.
func resourceOperation(ctx context.Context, r resource.Resource, verb string) context.Context {
	resp := &resource.MetadataResponse{}
	r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerTypeName}, resp)
	return client.WithOperation(ctx, resp.TypeName+" "+verb)
}

func dataSourceOperation(ctx context.Context, d datasource.DataSource) context.Context {
	resp := &datasource.MetadataResponse{}
	d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: providerTypeName}, resp)
	return client.WithOperation(ctx, "data."+resp.TypeName+" read")
}

func listResourceOperation(ctx context.Context, l list.ListResource) context.Context {
	resp := &resource.MetadataResponse{}
	l.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerTypeName}, resp)
	return client.WithOperation(ctx, "list."+resp.TypeName+" list")
}
