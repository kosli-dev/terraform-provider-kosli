package pact_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/pact-foundation/pact-go/v2/models"
	"github.com/pact-foundation/pact-go/v2/provider"
)

// stubKosliAPI returns an httptest server that mimics the Kosli API
// for the interactions in the pact file.
func stubKosliAPI() *httptest.Server {
	mux := http.NewServeMux()

	// GET /api/v2/environments/{org}/{name}
	mux.HandleFunc("/api/v2/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			// Fields the consumer contract requires:
			"name":             "production-k8s",
			"type":             "K8S",
			"description":      "Production Kubernetes cluster",
			"last_modified_at": 1700000000.123456,
			"last_reported_at": 1700000001.654321,
			"include_scaling":  true,
			"tags":             map[string]string{"env": "prod"},
			// Extra fields the real API returns (should be ignored by Pact):
			"org":               "test-org",
			"state":             nil,
			"require_provenance": false,
			"policies":          []any{},
		})
	})

	// Hello world endpoint from Step 1
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"greeting": "hi there",
		})
	})

	return httptest.NewServer(mux)
}

func TestVerifyProvider(t *testing.T) {
	// Start the stub server
	server := stubKosliAPI()
	defer server.Close()

	// Find the pact file
	pactDir, err := filepath.Abs("./pacts")
	if err != nil {
		t.Fatalf("failed to resolve pact dir: %v", err)
	}

	pactFile := filepath.Join(pactDir, "TerraformProviderKosli-KosliAPI.json")
	if _, err := os.Stat(pactFile); os.IsNotExist(err) {
		t.Fatalf("pact file not found: %s (run consumer tests first)", pactFile)
	}

	// Run provider verification
	verifier := provider.NewVerifier()

	err = verifier.VerifyProvider(t, provider.VerifyRequest{
		Provider:        "KosliAPI",
		ProviderBaseURL: server.URL,
		PactFiles:       []string{pactFile},
		StateHandlers: models.StateHandlers{
			"a greeting exists": func(setup bool, s models.ProviderState) (models.ProviderStateResponse, error) {
				// No setup needed for the hello world stub
				return nil, nil
			},
			"environment production-k8s exists": func(setup bool, s models.ProviderState) (models.ProviderStateResponse, error) {
				// Stub server always returns this environment — no setup needed
				fmt.Printf("  State handler called: %q (setup=%v)\n", s.Name, setup)
				return nil, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("provider verification failed: %v", err)
	}
}
