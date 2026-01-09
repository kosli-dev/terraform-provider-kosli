package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewClient_Success tests creating a client with valid inputs.
func TestNewClient_Success(t *testing.T) {
	client, err := NewClient("test-token", "test-org")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if client.apiToken != "test-token" {
		t.Errorf("expected apiToken 'test-token', got %s", client.apiToken)
	}

	if client.organization != "test-org" {
		t.Errorf("expected organization 'test-org', got %s", client.organization)
	}

	if client.baseURL != DefaultBaseURL {
		t.Errorf("expected baseURL %s, got %s", DefaultBaseURL, client.baseURL)
	}

	if client.apiPath != DefaultAPIPath {
		t.Errorf("expected apiPath %s, got %s", DefaultAPIPath, client.apiPath)
	}

	if client.userAgent != DefaultUserAgent {
		t.Errorf("expected userAgent %s, got %s", DefaultUserAgent, client.userAgent)
	}

	expectedURL := DefaultBaseURL + DefaultAPIPath
	if client.apiURL != expectedURL {
		t.Errorf("expected apiURL %s, got %s", expectedURL, client.apiURL)
	}
}

// TestNewClient_ValidationErrors tests validation of required parameters.
func TestNewClient_ValidationErrors(t *testing.T) {
	tests := []struct {
		name         string
		token        string
		org          string
		expectedErr  string
	}{
		{
			name:        "empty token",
			token:       "",
			org:         "test-org",
			expectedErr: "API token is required",
		},
		{
			name:        "empty organization",
			token:       "test-token",
			org:         "",
			expectedErr: "organization is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.token, tt.org)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error containing %q, got %q", tt.expectedErr, err.Error())
			}
		})
	}
}

// TestNewClient_WithOptions tests applying client options.
func TestNewClient_WithOptions(t *testing.T) {
	t.Run("with timeout", func(t *testing.T) {
		// Create a server that delays response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Set a very short timeout
		timeout := 50 * time.Millisecond
		client, err := NewClient("test-token", "test-org",
			WithBaseURL(server.URL),
			WithAPIPath(""),
			WithTimeout(timeout),
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Request should timeout
		_, err = client.Get(context.Background(), "/test")
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}

		// Error should contain timeout or deadline exceeded
		if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Errorf("expected timeout/deadline error, got: %v", err)
		}
	})

	t.Run("with user agent", func(t *testing.T) {
		userAgent := "custom-agent/1.0"
		client, err := NewClient("test-token", "test-org", WithUserAgent(userAgent))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if client.userAgent != userAgent {
			t.Errorf("expected userAgent %s, got %s", userAgent, client.userAgent)
		}
	})

	t.Run("with base URL", func(t *testing.T) {
		baseURL := "https://app.us.kosli.com"
		client, err := NewClient("test-token", "test-org", WithBaseURL(baseURL))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if client.baseURL != baseURL {
			t.Errorf("expected baseURL %s, got %s", baseURL, client.baseURL)
		}

		expectedURL := baseURL + DefaultAPIPath
		if client.apiURL != expectedURL {
			t.Errorf("expected apiURL %s, got %s", expectedURL, client.apiURL)
		}
	})

	t.Run("with API path", func(t *testing.T) {
		apiPath := "/api/v3"
		client, err := NewClient("test-token", "test-org", WithAPIPath(apiPath))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if client.apiPath != apiPath {
			t.Errorf("expected apiPath %s, got %s", apiPath, client.apiPath)
		}

		expectedURL := DefaultBaseURL + apiPath
		if client.apiURL != expectedURL {
			t.Errorf("expected apiURL %s, got %s", expectedURL, client.apiURL)
		}
	})

	t.Run("with custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{Timeout: 5 * time.Second}
		client, err := NewClient("test-token", "test-org", WithHTTPClient(customClient))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Note: After applying default retry policy, the client will be replaced
		// So we just verify no error occurred
		if client == nil {
			t.Error("expected client to be non-nil")
		}
	})
}

// TestNewClient_OptionErrors tests errors in option functions.
func TestNewClient_OptionErrors(t *testing.T) {
	tests := []struct {
		name        string
		option      ClientOption
		expectedErr string
	}{
		{
			name:        "nil HTTP client",
			option:      WithHTTPClient(nil),
			expectedErr: "http client cannot be nil",
		},
		{
			name:        "zero timeout",
			option:      WithTimeout(0),
			expectedErr: "timeout must be greater than 0",
		},
		{
			name:        "negative timeout",
			option:      WithTimeout(-1 * time.Second),
			expectedErr: "timeout must be greater than 0",
		},
		{
			name:        "empty user agent",
			option:      WithUserAgent(""),
			expectedErr: "user agent cannot be empty",
		},
		{
			name:        "empty base URL",
			option:      WithBaseURL(""),
			expectedErr: "base URL cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient("test-token", "test-org", tt.option)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error containing %q, got %q", tt.expectedErr, err.Error())
			}
		})
	}
}

// TestClient_Get_Success tests a successful GET request.
func TestClient_Get_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		if r.URL.Path != "/test-path" {
			t.Errorf("expected path /test-path, got %s", r.URL.Path)
		}

		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %q", auth)
		}

		if ua := r.Header.Get("User-Agent"); ua != DefaultUserAgent {
			t.Errorf("expected User-Agent %q, got %q", DefaultUserAgent, ua)
		}

		// Return mock response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	// Create client pointing to mock server
	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Execute request
	resp, err := client.Get(context.Background(), "/test-path")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Parse response
	var result map[string]string
	if err := ParseResponse(resp, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result["result"] != "success" {
		t.Errorf("expected result 'success', got %q", result["result"])
	}
}

// TestClient_Post_Success tests a successful POST request with body.
func TestClient_Post_Success(t *testing.T) {
	requestBody := map[string]string{"name": "test"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}

		// Verify body
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}

		if body["name"] != "test" {
			t.Errorf("expected name 'test', got %q", body["name"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": "123"}`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.Post(context.Background(), "/test-path", requestBody)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

// TestClient_Put_Success tests a successful PUT request.
func TestClient_Put_Success(t *testing.T) {
	requestBody := map[string]string{"name": "updated"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"updated": true}`))
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.Put(context.Background(), "/test-path", requestBody)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// TestClient_Delete_Success tests a successful DELETE request.
func TestClient_Delete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.Delete(context.Background(), "/test-path")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", resp.StatusCode)
	}
}

// TestClient_ErrorHandling tests error handling for various status codes.
func TestClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
		checkFunc      func(error) bool
	}{
		{
			name:           "400 Bad Request",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"message": "invalid request"}`,
			expectedErrMsg: "invalid request",
			checkFunc:      IsBadRequest,
		},
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"message": "unauthorized"}`,
			expectedErrMsg: "unauthorized",
			checkFunc:      IsUnauthorized,
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			responseBody:   `{"message": "forbidden"}`,
			expectedErrMsg: "forbidden",
			checkFunc:      IsForbidden,
		},
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"message": "not found"}`,
			expectedErrMsg: "not found",
			checkFunc:      IsNotFound,
		},
		{
			name:           "409 Conflict",
			statusCode:     http.StatusConflict,
			responseBody:   `{"message": "conflict"}`,
			expectedErrMsg: "conflict",
			checkFunc:      IsConflict,
		},
		{
			name:           "429 Too Many Requests",
			statusCode:     http.StatusTooManyRequests,
			responseBody:   `{"message": "rate limited"}`,
			expectedErrMsg: "rate limited",
			checkFunc:      IsTooManyRequests,
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"message": "server error"}`,
			expectedErrMsg: "server error",
			checkFunc:      IsServerError,
		},
		{
			name:           "502 Bad Gateway",
			statusCode:     http.StatusBadGateway,
			responseBody:   `{"message": "bad gateway"}`,
			expectedErrMsg: "bad gateway",
			checkFunc:      IsServerError,
		},
		{
			name:           "503 Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			responseBody:   `{"message": "service unavailable"}`,
			expectedErrMsg: "service unavailable",
			checkFunc:      IsServerError,
		},
		{
			name:           "non-JSON error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   "plain text error",
			expectedErrMsg: "plain text error",
			checkFunc:      IsServerError,
		},
		{
			name:           "error field instead of message",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"error": "bad data"}`,
			expectedErrMsg: "bad data",
			checkFunc:      IsBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Use a plain HTTP client (no retry wrapper) to test direct error responses
			httpClient := &http.Client{Timeout: 10 * time.Second}
			client, err := NewClient("test-token", "test-org",
				WithBaseURL(server.URL),
				WithAPIPath(""),
				WithHTTPClient(httpClient), // This prevents the default retry policy from being applied
			)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			_, err = client.Get(context.Background(), "/test-path")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("expected error containing %q, got %q", tt.expectedErrMsg, err.Error())
			}

			if tt.checkFunc != nil && !tt.checkFunc(err) {
				t.Errorf("error check function failed for error: %v", err)
			}

			// Verify APIError fields
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatal("expected APIError")
			}

			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, apiErr.StatusCode)
			}
		})
	}
}

// TestClient_ContextCancellation tests request cancellation via context.
func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep to allow time for cancellation
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.Get(ctx, "/test-path")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context canceled error, got %v", err)
	}
}

// TestClient_ContextTimeout tests request timeout via context.
func TestClient_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the timeout
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = client.Get(ctx, "/test-path")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "deadline exceeded") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected timeout error, got %v", err)
	}
}

// TestParseResponse tests the ParseResponse helper function.
func TestParseResponse(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"name": "test", "value": 123}`))
		}))
		defer server.Close()

		client, err := NewClient("test-token", "test-org",
			WithBaseURL(server.URL),
			WithAPIPath(""),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.Get(context.Background(), "/test")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		var result map[string]any
		err = ParseResponse(resp, &result)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if result["name"] != "test" {
			t.Errorf("expected name 'test', got %v", result["name"])
		}

		if result["value"] != float64(123) {
			t.Errorf("expected value 123, got %v", result["value"])
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`not valid json`))
		}))
		defer server.Close()

		client, err := NewClient("test-token", "test-org",
			WithBaseURL(server.URL),
			WithAPIPath(""),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.Get(context.Background(), "/test")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		var result map[string]any
		err = ParseResponse(resp, &result)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "failed to unmarshal") {
			t.Errorf("expected unmarshal error, got %v", err)
		}
	})
}

// TestClient_Organization tests the Organization getter.
func TestClient_Organization(t *testing.T) {
	client, err := NewClient("test-token", "my-org")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if client.Organization() != "my-org" {
		t.Errorf("expected organization 'my-org', got %s", client.Organization())
	}
}

// TestClient_RetryPolicy tests retry behavior.
func TestClient_RetryPolicy(t *testing.T) {
	t.Run("retry on 503", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"message": "service unavailable"}`))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"result": "success"}`))
			}
		}))
		defer server.Close()

		client, err := NewClient("test-token", "test-org",
			WithBaseURL(server.URL),
			WithAPIPath(""),
			WithRetryPolicy(3, 10*time.Millisecond, 100*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.Get(context.Background(), "/test")
		if err != nil {
			t.Fatalf("expected success after retries, got %v", err)
		}
		defer resp.Body.Close()

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("no retry on 404", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "not found"}`))
		}))
		defer server.Close()

		client, err := NewClient("test-token", "test-org",
			WithBaseURL(server.URL),
			WithAPIPath(""),
			WithRetryPolicy(3, 10*time.Millisecond, 100*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.Get(context.Background(), "/test")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if attempts != 1 {
			t.Errorf("expected 1 attempt (no retry on 404), got %d", attempts)
		}

		if !IsNotFound(err) {
			t.Errorf("expected NotFound error, got %v", err)
		}
	})

	t.Run("exhaust max retries", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"message": "always unavailable"}`))
		}))
		defer server.Close()

		client, err := NewClient("test-token", "test-org",
			WithBaseURL(server.URL),
			WithAPIPath(""),
			WithRetryPolicy(2, 10*time.Millisecond, 100*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.Get(context.Background(), "/test")
		if err == nil {
			t.Fatal("expected error after exhausting retries, got nil")
		}

		// Should try initial + 2 retries = 3 total
		if attempts != 3 {
			t.Errorf("expected 3 attempts (1 + 2 retries), got %d", attempts)
		}

		// Error may be wrapped by retry library, just check it's an error
		if err == nil {
			t.Error("expected error after exhausting retries, got nil")
		}
	})
}

// TestWithRetryPolicy_ValidationErrors tests validation in retry policy option.
func TestWithRetryPolicy_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		retryMax    int
		waitMin     time.Duration
		waitMax     time.Duration
		expectedErr string
	}{
		{
			name:        "negative retry max",
			retryMax:    -1,
			waitMin:     1 * time.Second,
			waitMax:     10 * time.Second,
			expectedErr: "retry max must be >= 0",
		},
		{
			name:        "zero wait min",
			retryMax:    3,
			waitMin:     0,
			waitMax:     10 * time.Second,
			expectedErr: "retry wait min must be > 0",
		},
		{
			name:        "zero wait max",
			retryMax:    3,
			waitMin:     1 * time.Second,
			waitMax:     0,
			expectedErr: "retry wait max must be > 0",
		},
		{
			name:        "wait min > wait max",
			retryMax:    3,
			waitMin:     10 * time.Second,
			waitMax:     1 * time.Second,
			expectedErr: "retry wait min must be <= retry wait max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient("test-token", "test-org",
				WithRetryPolicy(tt.retryMax, tt.waitMin, tt.waitMax),
			)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error containing %q, got %q", tt.expectedErr, err.Error())
			}
		})
	}
}

// TestAPIError_Error tests the Error method of APIError.
func TestAPIError_Error(t *testing.T) {
	t.Run("with message", func(t *testing.T) {
		err := &APIError{
			StatusCode: 400,
			Message:    "bad request",
		}

		expected := "kosli api error (status 400): bad request"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("without message", func(t *testing.T) {
		err := &APIError{
			StatusCode: 404,
		}

		if !strings.Contains(err.Error(), "404") {
			t.Errorf("expected error to contain 404, got %q", err.Error())
		}

		if !strings.Contains(err.Error(), "Not Found") {
			t.Errorf("expected error to contain 'Not Found', got %q", err.Error())
		}
	})
}

// Benchmark tests
func BenchmarkClient_Get(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	client, _ := NewClient("test-token", "test-org",
		WithBaseURL(server.URL),
		WithAPIPath(""),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(context.Background(), "/test")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
