package client

import (
	"context"
	"fmt"
	"strings"
)

// Logger receives client diagnostics. ctx is the caller's context, so
// request-scoped loggers such as tflog work.
type Logger interface {
	Debug(ctx context.Context, msg string, fields map[string]any)
	Warn(ctx context.Context, msg string, fields map[string]any)
}

// WithLogger sets the diagnostics logger; without one the client is silent.
// Attempts and responses log at debug, transport failures at warn. Attempt
// events come from the retry layer, so WithHTTPClient clients only log
// transport failures.
func WithLogger(logger Logger) ClientOption {
	return func(c *Client) error {
		if logger == nil {
			return fmt.Errorf("logger cannot be nil")
		}
		c.logger = logger
		return nil
	}
}

// sanitizeLogValue strips CR and LF to prevent forged log lines.
func sanitizeLogValue(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	return strings.ReplaceAll(s, "\r", "")
}
