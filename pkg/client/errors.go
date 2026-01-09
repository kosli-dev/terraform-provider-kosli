package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// APIError represents an error returned by the Kosli API.
type APIError struct {
	// StatusCode is the HTTP status code.
	StatusCode int

	// Message is the error message from the API.
	Message string

	// RequestID is the request ID for debugging (if provided by API).
	RequestID string

	// Method is the HTTP method that was used.
	Method string

	// URL is the URL that was requested.
	URL string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("kosli api error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("kosli api error (status %d): %s", e.StatusCode, http.StatusText(e.StatusCode))
}

// IsNotFound returns true if the error is a 404 Not Found error.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// IsUnauthorized returns true if the error is a 401 Unauthorized error.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}

// IsConflict returns true if the error is a 409 Conflict error.
func IsConflict(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict
}

// IsTooManyRequests returns true if the error is a 429 Too Many Requests error.
func IsTooManyRequests(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusTooManyRequests
}

// IsBadRequest returns true if the error is a 400 Bad Request error.
func IsBadRequest(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusBadRequest
}

// IsForbidden returns true if the error is a 403 Forbidden error.
func IsForbidden(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden
}

// IsServerError returns true if the error is a 5xx server error.
func IsServerError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode >= 500 && apiErr.StatusCode < 600
}

// apiErrorResponse represents the error response format from the Kosli API.
type apiErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Error   string `json:"error"` // Some APIs use "error" instead of "message"
}

// parseErrorResponse extracts error details from an API response.
func parseErrorResponse(resp *http.Response) error {
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Method:     resp.Request.Method,
		URL:        resp.Request.URL.String(),
		RequestID:  resp.Header.Get("X-Request-ID"),
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// If we can't read the body, just use the status text
		apiErr.Message = http.StatusText(resp.StatusCode)
		return apiErr
	}

	// Try to parse as JSON error response
	var errorResp apiErrorResponse
	if err := json.Unmarshal(body, &errorResp); err == nil {
		// Successfully parsed JSON error
		if errorResp.Message != "" {
			apiErr.Message = errorResp.Message
		} else if errorResp.Error != "" {
			apiErr.Message = errorResp.Error
		} else {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
	} else {
		// Not a JSON error, use the body as the message if it's not too long
		if len(body) > 0 && len(body) < 500 {
			apiErr.Message = string(body)
		} else {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
	}

	return apiErr
}
