package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// headerServer records the headers of the last request it received.
func headerServer(t *testing.T, body string) (*httptest.Server, *http.Header) {
	t.Helper()
	got := &http.Header{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*got = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)
	return server, got
}

func TestOperationNames(t *testing.T) {
	server, got := headerServer(t, `{}`)
	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	ctx := context.Background()
	tests := []struct {
		ctx  context.Context
		want string
	}{
		{resourceOperation(ctx, &environmentResource{}, "create"), "kosli_environment create"},
		{resourceOperation(ctx, &environmentResource{}, "read"), "kosli_environment read"},
		{resourceOperation(ctx, &flowResource{}, "update"), "kosli_flow update"},
		{resourceOperation(ctx, &customAttestationTypeResource{}, "delete"), "kosli_custom_attestation_type delete"},
		{resourceOperation(ctx, &actionResource{}, "import"), "kosli_action import"},
		{dataSourceOperation(ctx, &flowDataSource{}), "data.kosli_flow read"},
		{dataSourceOperation(ctx, &serviceAccountAPIKeysDataSource{}), "data.kosli_service_account_api_keys read"},
		{listResourceOperation(ctx, &controlListResource{}), "list.kosli_control list"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			resp, err := c.Get(tt.ctx, "/anything")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			resp.Body.Close()
			if got.Get(client.HeaderOperation) != tt.want {
				t.Errorf("%s = %q, want %q", client.HeaderOperation, got.Get(client.HeaderOperation), tt.want)
			}
		})
	}
}

func TestControlListResource_List_SendsOperation(t *testing.T) {
	ctx := context.Background()
	server, got := headerServer(t, `{"page": 1, "per_page": 100, "total_pages": 1, "total_count": 0, "controls": []}`)
	c, err := client.NewClient("test-token", "test-org", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	l := &controlListResource{client: c}
	req := newControlListRequest(t, ctx,
		tftypes.NewValue(tftypes.String, "secret-search-term"),
		tftypes.NewValue(tftypes.Bool, nil),
		false, 0)
	stream := &list.ListResultsStream{}
	l.List(ctx, req, stream)
	collectListResults(stream)

	if got.Get(client.HeaderOperation) != "list.kosli_control list" {
		t.Errorf("%s = %q, want %q", client.HeaderOperation, got.Get(client.HeaderOperation), "list.kosli_control list")
	}
	for name, values := range *got {
		for _, v := range values {
			if strings.Contains(v, "secret-search-term") {
				t.Errorf("header %s leaks a configuration value: %q", name, v)
			}
		}
	}
}

func TestKosliProvider_Configure_SetsAnalyticsHeaders(t *testing.T) {
	server, got := headerServer(t, `{}`)
	resp := configureWithEnv(t, "test-token", "test-org", server.URL)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	c := resp.ResourceData.(*client.Client)

	r, err := c.Get(context.Background(), "/anything")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r.Body.Close()

	if got.Get(client.HeaderClient) != "terraform/test" {
		t.Errorf("%s = %q, want %q", client.HeaderClient, got.Get(client.HeaderClient), "terraform/test")
	}
	if got.Get("User-Agent") != "terraform-provider-kosli/test" {
		t.Errorf("User-Agent = %q, want %q", got.Get("User-Agent"), "terraform-provider-kosli/test")
	}
	id, err := uuid.Parse(got.Get(client.HeaderInvocationID))
	if err != nil || id.Version() != 4 {
		t.Errorf("%s = %q, want a UUID v4", client.HeaderInvocationID, got.Get(client.HeaderInvocationID))
	}
	if id.String() != invocationID() {
		t.Errorf("%s = %q, want the process invocation id %q", client.HeaderInvocationID, id, invocationID())
	}
	if _, ok := (*got)[client.HeaderOperation]; ok {
		t.Errorf("%s should be omitted outside an operation", client.HeaderOperation)
	}
}
