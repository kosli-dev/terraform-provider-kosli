package provider

import (
	"bytes"
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

var _ client.Logger = tflogClientLogger{}

func TestTflogClientLogger(t *testing.T) {
	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)

	logger := tflogClientLogger{}
	logger.Debug(ctx, "sending kosli api request", map[string]any{"method": "GET", "url": "https://app.kosli.com/api/v2/x"})
	logger.Warn(ctx, "kosli api request failed", map[string]any{"error": "boom"})

	entries, err := tflogtest.MultilineJSONDecode(&buf)
	if err != nil {
		t.Fatalf("decode log output: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d log entries, want 2: %v", len(entries), entries)
	}

	if entries[0]["@level"] != "debug" || entries[0]["@message"] != "sending kosli api request" || entries[0]["method"] != "GET" {
		t.Errorf("unexpected debug entry: %v", entries[0])
	}
	if entries[1]["@level"] != "warn" || entries[1]["@message"] != "kosli api request failed" || entries[1]["error"] != "boom" {
		t.Errorf("unexpected warn entry: %v", entries[1])
	}
}
