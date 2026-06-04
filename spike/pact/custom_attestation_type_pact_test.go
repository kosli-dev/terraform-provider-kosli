package pact_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
	"github.com/pact-foundation/pact-go/v2/consumer"
	"github.com/pact-foundation/pact-go/v2/matchers"
)

func TestCustomAttestationType_Create_Pact(t *testing.T) {
	mockProvider, err := consumer.NewV2Pact(consumer.MockHTTPProviderConfig{
		Consumer: "TerraformProviderKosli",
		Provider: "KosliAPI",
		PactDir:  "./pacts",
	})
	if err != nil {
		t.Fatalf("failed to create pact: %v", err)
	}

	// Create uses multipart/form-data, which Pact V2 can't match on the
	// request body. We only verify method, path, and response.
	// See spike notes: this is a known limitation, and the client code
	// bypasses doRequest() for this endpoint (should be refactored).
	mockProvider.
		AddInteraction().
		Given("no attestation type named test-coverage exists").
		UponReceiving("a request to create custom attestation type test-coverage").
		WithRequestPathMatcher("POST", matchers.Term(
			"/api/v2/custom-attestation-types/test-org",
			`/api/v2/custom-attestation-types/[^/]+`,
		)).
		WillRespondWith(201, func(b *consumer.V2ResponseBuilder) {
			b.Header("Content-Type", matchers.String("application/json"))
			// The API returns the JSON string "OK". The consumer doesn't
			// read the body, so we just check the status code and headers.
		})

	err = mockProvider.ExecuteTest(t, func(config consumer.MockServerConfig) error {
		baseURL := fmt.Sprintf("http://%s:%d", config.Host, config.Port)
		c, err := client.NewClient("test-token", "test-org",
			client.WithBaseURL(baseURL),
		)
		if err != nil {
			return err
		}

		err = c.CreateCustomAttestationType(context.Background(), &client.CreateCustomAttestationTypeRequest{
			Name:        "test-coverage",
			Description: "Code coverage attestation",
			Schema:      `{"type":"object","properties":{"coverage":{"type":"number"}}}`,
			JqRules:     []string{".coverage >= 80"},
		})
		if err != nil {
			return fmt.Errorf("CreateCustomAttestationType failed: %w", err)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("pact test failed: %v", err)
	}
}

func TestCustomAttestationType_Read_Pact(t *testing.T) {
	mockProvider, err := consumer.NewV2Pact(consumer.MockHTTPProviderConfig{
		Consumer: "TerraformProviderKosli",
		Provider: "KosliAPI",
		PactDir:  "./pacts",
	})
	if err != nil {
		t.Fatalf("failed to create pact: %v", err)
	}

	// The API returns a versions array; the client extracts the latest version's
	// type_schema and evaluator into flat Schema/JqRules fields.
	// The contract must reflect the API shape, not the client's transformation.
	mockProvider.
		AddInteraction().
		Given("attestation type test-coverage exists with version 1").
		UponReceiving("a request to get custom attestation type test-coverage").
		WithRequestPathMatcher("GET", matchers.Term(
			"/api/v2/custom-attestation-types/test-org/test-coverage",
			`/api/v2/custom-attestation-types/[^/]+/[^/]+`,
		)).
		WillRespondWith(200, func(b *consumer.V2ResponseBuilder) {
			b.Header("Content-Type", matchers.String("application/json"))
			// Only fields the consumer reads: name, description, archived,
			// versions (with type_schema and evaluator for the latest version)
			b.JSONBody(map[string]any{
				"name":        matchers.Like("test-coverage"),
				"description": matchers.Like("Code coverage attestation"),
				"archived":    matchers.Like(false),
				"versions": matchers.EachLike(map[string]any{
					"version":   matchers.Like(1),
					"timestamp": matchers.Like(1700000000.123456),
					"type_schema": matchers.Like(map[string]any{
						"type": "object",
						"properties": map[string]any{
							"coverage": map[string]any{"type": "number"},
						},
					}),
					"evaluator": matchers.Like(map[string]any{
						"content_type": "jq",
						"rules":        []string{".coverage >= 80"},
					}),
					"created_by": matchers.Like("Test User"),
				}, 1),
			})
		})

	err = mockProvider.ExecuteTest(t, func(config consumer.MockServerConfig) error {
		baseURL := fmt.Sprintf("http://%s:%d", config.Host, config.Port)
		c, err := client.NewClient("test-token", "test-org",
			client.WithBaseURL(baseURL),
		)
		if err != nil {
			return err
		}

		result, err := c.GetCustomAttestationType(context.Background(), "test-coverage", nil)
		if err != nil {
			return fmt.Errorf("GetCustomAttestationType failed: %w", err)
		}

		// Verify the client's transformation from API format
		if result.Name != "test-coverage" {
			t.Errorf("expected name 'test-coverage', got %s", result.Name)
		}
		if result.Schema == "" {
			t.Error("expected non-empty schema")
		}
		if len(result.JqRules) != 1 || result.JqRules[0] != ".coverage >= 80" {
			t.Errorf("expected jq_rules ['.coverage >= 80'], got %v", result.JqRules)
		}
		if result.Archived {
			t.Error("expected archived to be false")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("pact test failed: %v", err)
	}
}

func TestCustomAttestationType_Delete_Pact(t *testing.T) {
	mockProvider, err := consumer.NewV2Pact(consumer.MockHTTPProviderConfig{
		Consumer: "TerraformProviderKosli",
		Provider: "KosliAPI",
		PactDir:  "./pacts",
	})
	if err != nil {
		t.Fatalf("failed to create pact: %v", err)
	}

	// Delete (archive) is a PUT with no body
	mockProvider.
		AddInteraction().
		Given("attestation type test-coverage exists").
		UponReceiving("a request to archive custom attestation type test-coverage").
		WithRequestPathMatcher("PUT", matchers.Term(
			"/api/v2/custom-attestation-types/test-org/test-coverage/archive",
			`/api/v2/custom-attestation-types/[^/]+/[^/]+/archive`,
		)).
		WillRespondWith(200, func(b *consumer.V2ResponseBuilder) {
			b.Header("Content-Type", matchers.String("application/json"))
		})

	err = mockProvider.ExecuteTest(t, func(config consumer.MockServerConfig) error {
		baseURL := fmt.Sprintf("http://%s:%d", config.Host, config.Port)
		c, err := client.NewClient("test-token", "test-org",
			client.WithBaseURL(baseURL),
		)
		if err != nil {
			return err
		}

		err = c.ArchiveCustomAttestationType(context.Background(), "test-coverage")
		if err != nil {
			return fmt.Errorf("ArchiveCustomAttestationType failed: %w", err)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("pact test failed: %v", err)
	}
}
