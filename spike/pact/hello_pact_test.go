package pact_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/pact-foundation/pact-go/v2/consumer"
	"github.com/pact-foundation/pact-go/v2/matchers"
)

func TestHelloPact(t *testing.T) {
	// Create a new Pact consumer test
	mockProvider, err := consumer.NewV2Pact(consumer.MockHTTPProviderConfig{
		Consumer: "TerraformProviderKosli",
		Provider: "KosliAPI",
		PactDir:  "./pacts",
	})
	if err != nil {
		t.Fatalf("failed to create pact: %v", err)
	}

	// Define the interaction: when the consumer GETs /hello, the provider responds with a greeting.
	// The pact-go v2 API is fluent — each method returns the builder, not an error.
	mockProvider.
		AddInteraction().
		Given("a greeting exists").
		UponReceiving("a request for a greeting").
		WithRequest("GET", "/hello").
		WillRespondWith(200, func(b *consumer.V2ResponseBuilder) {
			b.Header("Content-Type", matchers.String("application/json"))
			b.JSONBody(map[string]any{
				"greeting": matchers.Like("hello world"),
			})
		})

	// Execute the test — pact-go starts a mock server and gives us the URL
	err = mockProvider.ExecuteTest(t, func(config consumer.MockServerConfig) error {
		// Make a real HTTP call to the pact mock server
		url := fmt.Sprintf("http://%s:%d/hello", config.Host, config.Port)
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("expected 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read body: %w", err)
		}

		t.Logf("Response body: %s", string(body))
		return nil
	})
	if err != nil {
		t.Fatalf("pact test failed: %v", err)
	}
}
