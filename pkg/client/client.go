// Package client provides a Go client for the Kosli API.
//
// The client handles authentication, request/response processing, error handling,
// and retry logic for communicating with the Kosli API.
//
// Example usage:
//
//	client, err := client.NewClient("api-token", "org-name")
//	if err != nil {
//	    return err
//	}
//
//	resp, err := client.Get(context.Background(), "/custom-attestation-types/org-name")
//	if err != nil {
//	    return err
//	}
//	defer resp.Body.Close()
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	// DefaultBaseURL is the default Kosli API base URL (EU region).
	DefaultBaseURL = "https://app.kosli.com"

	// DefaultAPIPath is the default API path.
	DefaultAPIPath = "/api/v2"

	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 30 * time.Second

	// DefaultUserAgent is the default User-Agent header value.
	DefaultUserAgent = "terraform-provider-kosli/dev"

	// DefaultRetryMax is the default maximum number of retries.
	DefaultRetryMax = 3

	// DefaultRetryWaitMin is the default minimum wait time between retries.
	DefaultRetryWaitMin = 1 * time.Second

	// DefaultRetryWaitMax is the default maximum wait time between retries.
	DefaultRetryWaitMax = 30 * time.Second
)

// Client represents a Kosli API client.
type Client struct {
	// httpClient is the underlying HTTP client used for requests.
	httpClient *http.Client

	// baseURL is the base URL of the Kosli API (e.g., "https://app.kosli.com").
	baseURL string

	// apiPath is the API path (e.g., "/api/v2").
	apiPath string

	// apiURL is the full API URL (baseURL + apiPath).
	apiURL string

	// apiToken is the API token for authentication.
	apiToken string

	// organization is the Kosli organization name.
	organization string

	// userAgent is the User-Agent header value.
	userAgent string
}

// ClientOption is a function that configures a Client.
type ClientOption func(*Client) error

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) error {
		if httpClient == nil {
			return fmt.Errorf("http client cannot be nil")
		}
		c.httpClient = httpClient
		return nil
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be greater than 0")
		}
		if c.httpClient != nil {
			c.httpClient.Timeout = timeout
		}
		return nil
	}
}

// WithUserAgent sets a custom User-Agent header.
func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) error {
		if userAgent == "" {
			return fmt.Errorf("user agent cannot be empty")
		}
		c.userAgent = userAgent
		return nil
	}
}

// WithBaseURL sets a custom base URL for the API.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		if baseURL == "" {
			return fmt.Errorf("base URL cannot be empty")
		}
		// Trim trailing slash
		c.baseURL = strings.TrimRight(baseURL, "/")
		c.apiURL = c.baseURL + c.apiPath
		return nil
	}
}

// WithAPIPath sets a custom API path.
// An empty string is allowed and results in requests going directly to the base URL.
func WithAPIPath(apiPath string) ClientOption {
	return func(c *Client) error {
		// Allow empty API path for direct access to base URL
		if apiPath != "" {
			// Ensure leading slash
			if !strings.HasPrefix(apiPath, "/") {
				apiPath = "/" + apiPath
			}
			c.apiPath = strings.TrimRight(apiPath, "/")
		} else {
			c.apiPath = ""
		}
		c.apiURL = c.baseURL + c.apiPath
		return nil
	}
}

// WithRetryPolicy enables retry with exponential backoff.
func WithRetryPolicy(retryMax int, retryWaitMin, retryWaitMax time.Duration) ClientOption {
	return func(c *Client) error {
		if retryMax < 0 {
			return fmt.Errorf("retry max must be >= 0")
		}
		if retryWaitMin <= 0 {
			return fmt.Errorf("retry wait min must be > 0")
		}
		if retryWaitMax <= 0 {
			return fmt.Errorf("retry wait max must be > 0")
		}
		if retryWaitMin > retryWaitMax {
			return fmt.Errorf("retry wait min must be <= retry wait max")
		}

		// Create retryable HTTP client
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = retryMax
		retryClient.RetryWaitMin = retryWaitMin
		retryClient.RetryWaitMax = retryWaitMax
		retryClient.CheckRetry = retryablehttp.DefaultRetryPolicy
		retryClient.Backoff = retryablehttp.DefaultBackoff
		retryClient.HTTPClient = &http.Client{
			Timeout: c.httpClient.Timeout,
		}

		// Replace the standard client with the retryable client's standard client
		c.httpClient = retryClient.StandardClient()
		return nil
	}
}

// NewClient creates a new Kosli API client.
//
// Required parameters:
//   - apiToken: The Kosli API token for authentication
//   - organization: The Kosli organization name
//
// Optional parameters can be provided via ClientOption functions.
//
// Example:
//
//	client, err := NewClient(
//	    "api-token",
//	    "org-name",
//	    WithTimeout(60 * time.Second),
//	    WithRetryPolicy(5, 2*time.Second, 60*time.Second),
//	)
func NewClient(apiToken, organization string, opts ...ClientOption) (*Client, error) {
	// Validate required parameters
	if apiToken == "" {
		return nil, fmt.Errorf("API token is required")
	}
	if organization == "" {
		return nil, fmt.Errorf("organization is required")
	}

	// Create client with defaults
	client := &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		baseURL:      DefaultBaseURL,
		apiPath:      DefaultAPIPath,
		apiToken:     apiToken,
		organization: organization,
		userAgent:    DefaultUserAgent,
	}

	// Compute full API URL
	client.apiURL = client.baseURL + client.apiPath

	// Store the original http client to detect if it was replaced
	originalClient := client.httpClient

	// Apply user options first
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// If the user provided a custom HTTP client (via WithHTTPClient), don't apply retry
	// Otherwise, apply default retry policy
	if client.httpClient == originalClient {
		if err := WithRetryPolicy(DefaultRetryMax, DefaultRetryWaitMin, DefaultRetryWaitMax)(client); err != nil {
			return nil, fmt.Errorf("failed to apply default retry policy: %w", err)
		}
	}

	return client, nil
}

// Get performs a GET request to the specified path.
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request to the specified path with the given body.
func (c *Client) Post(ctx context.Context, path string, body any) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request to the specified path with the given body.
func (c *Client) Put(ctx context.Context, path string, body any) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request to the specified path.
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodDelete, path, nil)
}

// doRequest performs an HTTP request with authentication and error handling.
func (c *Client) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	// Build full URL
	url := c.apiURL + path

	// Marshal body to JSON if provided
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, parseErrorResponse(resp)
	}

	return resp, nil
}

// ParseResponse reads and unmarshals a JSON response body into the provided interface.
//
// The response body is closed after reading.
//
// Example:
//
//	var result []AttestationType
//	err := client.ParseResponse(resp, &result)
func ParseResponse(resp *http.Response, v any) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// Organization returns the organization name configured for this client.
func (c *Client) Organization() string {
	return c.organization
}
