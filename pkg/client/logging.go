package client

import (
	"context"
	"fmt"
	"strings"
)

// Logger receives diagnostic events from the client. The context is the one
// passed to the originating client call, so implementations can use
// request-scoped loggers such as terraform-plugin-log.
type Logger interface {
	Debug(ctx context.Context, msg string, fields map[string]any)
	Warn(ctx context.Context, msg string, fields map[string]any)
}

// WithLogger sets the logger that receives request diagnostics. Each request
// attempt and response is logged at debug level and transport failures at
// warn level. Without a logger the client is silent; the retry library's own
// stderr logger is never used.
//
// Attempt and response events come from the retry layer, so a client built
// with WithHTTPClient (which bypasses retries) only reports transport
// failures.
func WithLogger(logger Logger) ClientOption {
	return func(c *Client) error {
		if logger == nil {
			return fmt.Errorf("logger cannot be nil")
		}
		c.logger = logger
		return nil
	}
}

// sanitizeLogValue strips CR and LF so a logged value cannot forge extra log
// lines when the logger writes plain text.
func sanitizeLogValue(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	return strings.ReplaceAll(s, "\r", "")
}
