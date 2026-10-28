package client

import (
	"context"
	"fmt"
)

// Headers that tell the Kosli server which client, operation and run made a
// request. They carry no configuration values, only static names.
const (
	HeaderClient       = "X-Kosli-Client"
	HeaderOperation    = "X-Kosli-Operation"
	HeaderInvocationID = "X-Kosli-Invocation-Id"
)

type operationKey struct{}

// WithOperation returns a context whose requests send operation in the
// X-Kosli-Operation header, e.g. "kosli_environment create". The value is
// sent as given, so it must not contain user-supplied data.
func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, operationKey{}, operation)
}

func operationFromContext(ctx context.Context) string {
	operation, _ := ctx.Value(operationKey{}).(string)
	return operation
}

// WithClientName sets the X-Kosli-Client header, e.g. "terraform/v0.10.1".
func WithClientName(name string) ClientOption {
	return func(c *Client) error {
		if name == "" {
			return fmt.Errorf("client name cannot be empty")
		}
		c.clientName = name
		return nil
	}
}

// WithInvocationID sets the X-Kosli-Invocation-Id header, which groups the
// requests made by one run of the calling program.
func WithInvocationID(id string) ClientOption {
	return func(c *Client) error {
		if id == "" {
			return fmt.Errorf("invocation id cannot be empty")
		}
		c.invocationID = id
		return nil
	}
}
