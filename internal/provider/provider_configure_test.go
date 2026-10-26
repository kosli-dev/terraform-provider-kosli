package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// configureWithEnv runs Configure with an all-null config so every value is
// resolved from the environment variables set on t.
func configureWithEnv(t *testing.T, token, org, apiURL string) *provider.ConfigureResponse {
	t.Helper()
	t.Setenv("KOSLI_API_TOKEN", token)
	t.Setenv("KOSLI_ORG", org)
	t.Setenv("KOSLI_API_URL", apiURL)

	ctx := context.Background()
	p := &KosliProvider{version: "test"}

	schemaResp := &provider.SchemaResponse{}
	p.Schema(ctx, provider.SchemaRequest{}, schemaResp)

	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	raw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"api_token": tftypes.NewValue(tftypes.String, nil),
		"org":       tftypes.NewValue(tftypes.String, nil),
		"api_url":   tftypes.NewValue(tftypes.String, nil),
		"timeout":   tftypes.NewValue(tftypes.Number, nil),
	})

	resp := &provider.ConfigureResponse{}
	p.Configure(ctx, provider.ConfigureRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw},
	}, resp)
	return resp
}

func TestKosliProvider_Configure_RejectsPlaintextAPIURL(t *testing.T) {
	resp := configureWithEnv(t, "test-token", "test-org", "http://app.kosli.com")

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic for an http api_url")
	}
	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "Invalid API URL" && strings.Contains(d.Detail(), "must use https") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Invalid API URL diagnostic, got %v", resp.Diagnostics)
	}
	if resp.ResourceData != nil {
		t.Error("no client should be created for an invalid api_url")
	}
}

func TestKosliProvider_Configure_AcceptsHTTPSAPIURL(t *testing.T) {
	resp := configureWithEnv(t, "test-token", "test-org", "https://app.us.kosli.com")

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.ResourceData.(*client.Client); !ok {
		t.Fatalf("ResourceData = %T, want *client.Client", resp.ResourceData)
	}
}
