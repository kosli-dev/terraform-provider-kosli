package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// recordingLogger captures client log events for assertions.
type recordingLogger struct {
	mu     sync.Mutex
	events []string
	fields []map[string]any
}

func (l *recordingLogger) record(level, msg string, fields map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, level+": "+msg)
	l.fields = append(l.fields, fields)
}

func (l *recordingLogger) Debug(_ context.Context, msg string, fields map[string]any) {
	l.record("debug", msg, fields)
}

func (l *recordingLogger) Warn(_ context.Context, msg string, fields map[string]any) {
	l.record("warn", msg, fields)
}

func TestWithLogger_RejectsNil(t *testing.T) {
	if _, err := NewClient("token", "org", WithLogger(nil)); err == nil {
		t.Fatal("expected error for nil logger")
	}
}

func TestClient_LogsAttemptsAndResponses(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	logger := &recordingLogger{}
	// WithLogger before the retry policy: hooks must still see the logger.
	c, err := NewClient("token", "org",
		WithBaseURL(server.URL),
		WithLogger(logger),
		WithRetryPolicy(2, time.Millisecond, 2*time.Millisecond),
	)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.Get(context.Background(), "/envs/x")
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	resp.Body.Close()

	want := []string{
		"debug: sending kosli api request",
		"debug: received kosli api response",
		"debug: retrying kosli api request",
		"debug: received kosli api response",
	}
	if strings.Join(logger.events, "\n") != strings.Join(want, "\n") {
		t.Fatalf("events = %q, want %q", logger.events, want)
	}
	if logger.fields[2]["attempt"] != 2 {
		t.Errorf("retry attempt field = %v, want 2", logger.fields[2]["attempt"])
	}
	if logger.fields[1]["status"] != http.StatusBadGateway || logger.fields[3]["status"] != http.StatusOK {
		t.Errorf("unexpected status fields: %v %v", logger.fields[1], logger.fields[3])
	}
	if url, _ := logger.fields[0]["url"].(string); !strings.HasSuffix(url, "/api/v2/envs/x") {
		t.Errorf("unexpected url field %q", url)
	}
}

func TestClient_LogsTransportFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close() // connection refused from here on

	logger := &recordingLogger{}
	c, err := NewClient("token", "org",
		WithBaseURL(server.URL),
		WithRetryPolicy(0, time.Millisecond, time.Millisecond),
		WithLogger(logger),
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := c.Get(context.Background(), "/x"); err == nil {
		t.Fatal("expected transport error")
	}
	last := logger.events[len(logger.events)-1]
	if last != "warn: kosli api request failed" {
		t.Fatalf("last event = %q, want warn", last)
	}
	if _, ok := logger.fields[len(logger.fields)-1]["error"]; !ok {
		t.Error("warn event missing error field")
	}
}

func TestSanitizeLogValue(t *testing.T) {
	got := sanitizeLogValue("GET /x\r\nforged: entry\n")
	if got != "GET /xforged: entry" {
		t.Errorf("sanitizeLogValue = %q", got)
	}
}
