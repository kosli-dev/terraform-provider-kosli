package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// tflogClientLogger forwards Kosli API client diagnostics to
// terraform-plugin-log, so they honour TF_LOG and carry the request-scoped
// fields (RPC name, request ID) of the Terraform operation that made the call.
type tflogClientLogger struct{}

func (tflogClientLogger) Debug(ctx context.Context, msg string, fields map[string]any) {
	tflog.Debug(ctx, msg, fields)
}

func (tflogClientLogger) Warn(ctx context.Context, msg string, fields map[string]any) {
	tflog.Warn(ctx, msg, fields)
}
