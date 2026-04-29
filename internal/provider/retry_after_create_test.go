package provider

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// withFastBackoffs replaces the package-level backoff schedule with zero-duration
// waits for the duration of a test, restoring it on cleanup. Keeps the actual
// retry count intact so tests can assert how many attempts were made.
func withFastBackoffs(t *testing.T) {
	t.Helper()
	prev := retryAfterCreateBackoffs
	retryAfterCreateBackoffs = make([]time.Duration, len(prev))
	t.Cleanup(func() { retryAfterCreateBackoffs = prev })
}

func TestRetryReadAfterCreate_SuccessNoRetry(t *testing.T) {
	withFastBackoffs(t)
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
	withFastBackoffs(t)
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
	withFastBackoffs(t)
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
	if errors.Is(err, ErrRenameRace) {
		t.Fatalf("non-404 errors must not be wrapped with ErrRenameRace, got %v", err)
	}
	if puts != 0 {
		t.Errorf("expected 0 re-PUTs on non-404, got %d", puts)
	}
	if gets != 1 {
		t.Errorf("expected 1 GET on non-404, got %d", gets)
	}
}

func TestRetryReadAfterCreate_PutErrorPropagates(t *testing.T) {
	withFastBackoffs(t)
	notFound := &client.APIError{StatusCode: http.StatusNotFound}
	putErr := errors.New("put failed")
	_, err := retryReadAfterCreate(context.Background(),
		func(context.Context) error { return putErr },
		func(context.Context) (*string, error) { return nil, notFound },
	)
	if !errors.Is(err, putErr) {
		t.Fatalf("expected put error to propagate, got %v", err)
	}
	if errors.Is(err, ErrRenameRace) {
		t.Fatalf("re-PUT errors must not be wrapped with ErrRenameRace, got %v", err)
	}
}

func TestRetryReadAfterCreate_ContextCancelled(t *testing.T) {
	withFastBackoffs(t)
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
	if errors.Is(err, ErrRenameRace) {
		t.Fatalf("context.Canceled must not be wrapped with ErrRenameRace, got %v", err)
	}
}

// TestRetryReadAfterCreate_ExhaustedRetriesWrapsRenameRace asserts that when
// every GET attempt returns 404 the helper wraps the final error with
// ErrRenameRace. It also pins the number of attempts to len(backoffs)+1
// (initial + one per backoff entry) so accidental changes to the schedule
// surface as test failures.
func TestRetryReadAfterCreate_ExhaustedRetriesWrapsRenameRace(t *testing.T) {
	withFastBackoffs(t)
	notFound := &client.APIError{StatusCode: http.StatusNotFound, Message: "archived"}
	var puts, gets int
	_, err := retryReadAfterCreate(context.Background(),
		func(context.Context) error { puts++; return nil },
		func(context.Context) (*string, error) { gets++; return nil, notFound },
	)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !errors.Is(err, ErrRenameRace) {
		t.Errorf("expected ErrRenameRace wrap, got %v", err)
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected wrapped error to still satisfy IsNotFound, got %v", err)
	}
	wantGets := len(retryAfterCreateBackoffs) + 1
	if gets != wantGets {
		t.Errorf("expected %d GETs (initial + 1 per backoff), got %d", wantGets, gets)
	}
	if puts != len(retryAfterCreateBackoffs) {
		t.Errorf("expected %d re-PUTs (one per backoff), got %d", len(retryAfterCreateBackoffs), puts)
	}
}

// TestRetryReadAfterCreate_NilRePutSkipsReassert covers the path used by
// kosli_custom_attestation_type, where the create endpoint is a POST that
// allocates a new version on every call and so must not be re-issued.
func TestRetryReadAfterCreate_NilRePutSkipsReassert(t *testing.T) {
	withFastBackoffs(t)
	notFound := &client.APIError{StatusCode: http.StatusNotFound}
	var gets int
	_, err := retryReadAfterCreate[string](context.Background(),
		nil,
		func(context.Context) (*string, error) { gets++; return nil, notFound },
	)
	if !errors.Is(err, ErrRenameRace) {
		t.Fatalf("expected ErrRenameRace, got %v", err)
	}
	if gets != len(retryAfterCreateBackoffs)+1 {
		t.Errorf("expected GETs to be retried even without rePut, got %d", gets)
	}
}

func TestRenameRaceDetail_AppendsHintOnlyForRenameRace(t *testing.T) {
	notFound := &client.APIError{StatusCode: http.StatusNotFound, Message: "archived"}
	raceErr := errors.Join(ErrRenameRace, notFound)
	if got := renameRaceDetail("environment", "test", raceErr); !contains(got, "terraform state mv") {
		t.Errorf("expected rename hint for ErrRenameRace, got: %s", got)
	}

	transient := errors.New("upstream 503")
	if got := renameRaceDetail("environment", "test", transient); contains(got, "terraform state mv") {
		t.Errorf("did not expect rename hint for non-rename error, got: %s", got)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
