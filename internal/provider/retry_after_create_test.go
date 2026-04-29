package provider

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

func TestRetryReadAfterCreate_SuccessNoRetry(t *testing.T) {
	var puts, gets int
	v, err := retryReadAfterCreate(context.Background(),
		func(context.Context) error { puts++; return nil },
		func(context.Context) (*string, error) {
			gets++
			s := "ok"
			return &s, nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v == nil || *v != "ok" {
		t.Fatalf("unexpected value: %v", v)
	}
	if puts != 0 {
		t.Errorf("expected 0 re-PUTs, got %d", puts)
	}
	if gets != 1 {
		t.Errorf("expected 1 GET, got %d", gets)
	}
}

func TestRetryReadAfterCreate_RetriesOn404(t *testing.T) {
	notFound := &client.APIError{StatusCode: http.StatusNotFound, Message: "archived"}
	var puts, gets int
	v, err := retryReadAfterCreate(context.Background(),
		func(context.Context) error { puts++; return nil },
		func(context.Context) (*string, error) {
			gets++
			if gets == 1 {
				return nil, notFound
			}
			s := "ok"
			return &s, nil
		},
	)
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if v == nil || *v != "ok" {
		t.Fatalf("unexpected value: %v", v)
	}
	if puts != 1 {
		t.Errorf("expected 1 re-PUT after initial 404, got %d", puts)
	}
	if gets != 2 {
		t.Errorf("expected 2 GETs, got %d", gets)
	}
}

func TestRetryReadAfterCreate_NonNotFoundReturnsImmediately(t *testing.T) {
	boom := errors.New("boom")
	var puts, gets int
	_, err := retryReadAfterCreate(context.Background(),
		func(context.Context) error { puts++; return nil },
		func(context.Context) (*string, error) {
			gets++
			return nil, boom
		},
	)
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}
	if puts != 0 {
		t.Errorf("expected 0 re-PUTs on non-404, got %d", puts)
	}
	if gets != 1 {
		t.Errorf("expected 1 GET on non-404, got %d", gets)
	}
}

func TestRetryReadAfterCreate_PutErrorPropagates(t *testing.T) {
	notFound := &client.APIError{StatusCode: http.StatusNotFound}
	putErr := errors.New("put failed")
	_, err := retryReadAfterCreate(context.Background(),
		func(context.Context) error { return putErr },
		func(context.Context) (*string, error) { return nil, notFound },
	)
	if !errors.Is(err, putErr) {
		t.Fatalf("expected put error to propagate, got %v", err)
	}
}

func TestRetryReadAfterCreate_ContextCancelled(t *testing.T) {
	notFound := &client.APIError{StatusCode: http.StatusNotFound}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := retryReadAfterCreate(ctx,
		func(context.Context) error { return nil },
		func(context.Context) (*string, error) { return nil, notFound },
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
