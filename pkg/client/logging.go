package client

import (
	"context"
	"fmt"
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
func WithLogger(logger Logger) ClientOption {
	return func(c *Client) error {
		if logger == nil {
			return fmt.Errorf("logger cannot be nil")
		}
		c.logger = logger
		return nil
	}
}
