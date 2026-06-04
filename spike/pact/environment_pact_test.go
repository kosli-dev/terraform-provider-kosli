package pact_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
	"github.com/pact-foundation/pact-go/v2/consumer"
	"github.com/pact-foundation/pact-go/v2/matchers"
)

func TestGetEnvironment_Pact(t *testing.T) {
	mockProvider, err := consumer.NewV2Pact(consumer.MockHTTPProviderConfig{
		Consumer: "TerraformProviderKosli",
		Provider: "KosliAPI",
		PactDir:  "./pacts",
	})
	if err != nil {
		t.Fatalf("failed to create pact: %v", err)
	}

	// Define the interaction: GET /environments/{org}/{name} returns an environment
	mockProvider.
		AddInteraction().
		Given("environment production-k8s exists").
		UponReceiving("a request to get environment production-k8s").
		WithRequestPathMatcher("GET", matchers.Term("/api/v2/environments/test-org/production-k8s", `/api/v2/environments/[^/]+/[^/]+`)).
		WillRespondWith(200, func(b *consumer.V2ResponseBuilder) {
			b.Header("Content-Type", matchers.String("application/json"))
			// Only include fields the consumer (data source) actually reads.
			// Pact's response matching is liberal — extra fields from the
			// provider are allowed and ignored.
			b.JSONBody(map[string]any{
				"name":             matchers.Like("production-k8s"),
				"type":             matchers.Like("K8S"),
				"description":      matchers.Like("Production Kubernetes cluster"),
				"last_modified_at": matchers.Like(1234567890.123456),
				"last_reported_at": matchers.Like(1234567891.123456),
				"include_scaling":  matchers.Like(true),
				"tags":             matchers.Like(map[string]string{"env": "prod"}),
			})
		})

	// Execute the test using the real pkg/client code
	err = mockProvider.ExecuteTest(t, func(config consumer.MockServerConfig) error {
		// Create a real client pointing at Pact's mock server
		baseURL := fmt.Sprintf("http://%s:%d", config.Host, config.Port)
		c, err := client.NewClient("test-token", "test-org",
			client.WithBaseURL(baseURL),
		)
		if err != nil {
			return err
		}

		env, err := c.GetEnvironment(context.Background(), "production-k8s")
		if err != nil {
			return err
		}

		// Verify the client parsed the response correctly
		if env.Name != "production-k8s" {
			t.Errorf("expected name 'production-k8s', got %s", env.Name)
		}
		if env.Type != "K8S" {
			t.Errorf("expected type 'K8S', got %s", env.Type)
		}
		if env.Description != "Production Kubernetes cluster" {
			t.Errorf("expected description, got %s", env.Description)
		}
		if !env.IncludeScaling {
			t.Error("expected IncludeScaling to be true")
		}
		if env.LastReportedAt == nil {
			t.Error("expected non-nil LastReportedAt")
		}
		if env.Tags["env"] != "prod" {
			t.Errorf("expected tag env=prod, got %v", env.Tags)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("pact test failed: %v", err)
	}
}
