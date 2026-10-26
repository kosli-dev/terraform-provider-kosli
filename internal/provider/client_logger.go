package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// tflogClientLogger forwards client diagnostics to tflog, so they honour
// TF_LOG and carry the calling operation's request fields.
type tflogClientLogger struct{}

func (tflogClientLogger) Debug(ctx context.Context, msg string, fields map[string]any) {
	tflog.Debug(ctx, msg, fields)
}

func (tflogClientLogger) Warn(ctx context.Context, msg string, fields map[string]any) {
	tflog.Warn(ctx, msg, fields)
}
