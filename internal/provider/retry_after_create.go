package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// ErrRenameRace is returned by retryReadAfterCreate when every retry attempt
// observes a 404. Callers use errors.Is to decide whether to append the
// rename-race hint to user diagnostics; transient errors (5xx, context
// cancellation, re-PUT failures) are returned as-is and don't trigger the
// hint. See issue #121.
var ErrRenameRace = errors.New("post-create read kept returning 404 (likely Terraform label rename racing destroy on the same resource)")

// retryAfterCreateBackoffs controls the sleep schedule between retry
// attempts in retryReadAfterCreate. Exposed as a package-level var so tests
// can override it to avoid real sleeps.
var retryAfterCreateBackoffs = []time.Duration{
	500 * time.Millisecond,
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
}

// retryReadAfterCreate fetches a resource immediately after Create. If the
// initial GET returns 404, a sibling resource with the same name is being
// destroyed in parallel (e.g. a Terraform label rename plans as parallel
// destroy + create against the same Kosli resource). The destroy may archive
// the resource between our PUT and our GET.
//
// On 404, the helper waits, optionally re-asserts desired state via rePut,
// and re-GETs. Pass rePut=nil to skip re-asserting (useful when the
// underlying create endpoint is non-idempotent and re-issuing it would have
// side effects, such as creating a new version). Bounded backoff; non-404
// errors are returned immediately as-is. If every attempt returns 404, the
// final error is wrapped with ErrRenameRace so callers can identify the
// rename-race scenario specifically.
func retryReadAfterCreate[T any](
	ctx context.Context,
	rePut func(context.Context) error,
	get func(context.Context) (*T, error),
) (*T, error) {
	v, err := get(ctx)
	if err == nil {
		return v, nil
	}

	for _, wait := range retryAfterCreateBackoffs {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if !client.IsNotFound(err) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		if rePut != nil {
			if rePutErr := rePut(ctx); rePutErr != nil {
				return nil, rePutErr
			}
		}
		v, err = get(ctx)
		if err == nil {
			return v, nil
		}
	}

	if client.IsNotFound(err) {
		return nil, fmt.Errorf("%w: %w", ErrRenameRace, err)
	}
	return nil, err
}

// renameRaceHint is appended to "Error Reading ... After Creation" diagnostics
// only when the underlying error is ErrRenameRace, to point users at the
// correct workaround for an intentional label rename.
const renameRaceHint = "\n\nThis can happen when a Terraform resource label is renamed while keeping the same `name`: " +
	"Terraform plans the rename as a parallel destroy + create that target the same Kosli resource, " +
	"and the destroy can remove or archive the resource before this read completes. To rename the " +
	"resource label without recreating, use `terraform state mv` instead."

// renameRaceDetail formats a "Could not read X after creation" diagnostic
// detail, appending renameRaceHint only when err is an ErrRenameRace.
func renameRaceDetail(kind, name string, err error) string {
	detail := fmt.Sprintf("Could not read %s %q after creation: %s", kind, name, err.Error())
	if errors.Is(err, ErrRenameRace) {
		detail += renameRaceHint
	}
	return detail
}

// afterCreateSummary returns the diagnostic summary that best describes a
// post-create failure. Tag-PATCH failures returned from the rePut closure
// (identified via ErrTagApplyFailed) are surfaced as a tag update error so
// users aren't misled by a "Reading ... After Creation" header. All other
// errors keep the read-after-creation summary. displayKind must already be
// in the casing the user should see (e.g. "Logical Environment") so this
// helper agrees with sibling diagnostics emitted on non-retry code paths.
func afterCreateSummary(displayKind string, err error) string {
	if errors.Is(err, ErrTagApplyFailed) {
		return fmt.Sprintf("Error Updating %s Tags", displayKind)
	}
	return fmt.Sprintf("Error Reading %s After Creation", displayKind)
}

// ErrTagApplyFailed wraps errors returned by applyTagsAsError so call sites
// using the helper inside a retry closure can distinguish tag-PATCH failures
// from create/read failures and surface a more accurate diagnostic summary
// (e.g. "Error Updating Environment Tags" instead of "Error Reading
// Environment After Creation").
var ErrTagApplyFailed = errors.New("applying tags failed")

// applyTagsAsError invokes applyTags and collapses any error-severity
// diagnostics into a single error using errors.Join so all reported errors
// are preserved. The combined error is wrapped with ErrTagApplyFailed so
// callers can identify it via errors.Is. Used inside retry closures where
// the surrounding code expects an `error` return rather than a
// *diag.Diagnostics.
func applyTagsAsError(ctx context.Context, c *client.Client, name, resourceType string, oldTags, newTags types.Map) error {
	var d diag.Diagnostics
	applyTags(ctx, c, name, resourceType, oldTags, newTags, &d)
	var errs []error
	for _, entry := range d {
		if entry.Severity() == diag.SeverityError {
			errs = append(errs, fmt.Errorf("%s: %s", entry.Summary(), entry.Detail()))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrTagApplyFailed, errors.Join(errs...))
}
