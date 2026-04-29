package provider

import (
	"context"
	"time"

	"github.com/kosli-dev/terraform-provider-kosli/pkg/client"
)

// retryReadAfterCreate fetches a resource immediately after Create. If the
// initial GET returns 404, a sibling resource with the same name is being
// destroyed in parallel (e.g. a Terraform label rename plans as parallel
// destroy + create against the same Kosli resource). The destroy may archive
// the resource between our PUT and our GET.
//
// On 404, the helper waits, re-issues the create PUT to re-assert desired
// state, and re-GETs. Bounded backoff; non-404 errors are returned
// immediately. See issue #121.
func retryReadAfterCreate[T any](
	ctx context.Context,
	rePut func(context.Context) error,
	get func(context.Context) (*T, error),
) (*T, error) {
	backoffs := []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second, 4 * time.Second}

	v, err := get(ctx)
	if err == nil {
		return v, nil
	}

	for _, wait := range backoffs {
		if !client.IsNotFound(err) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		if rePutErr := rePut(ctx); rePutErr != nil {
			return nil, rePutErr
		}
		v, err = get(ctx)
		if err == nil {
			return v, nil
		}
	}

	return nil, err
}

// renameRaceHint is appended to "Error Reading ... After Creation" diagnostics
// to point users at the correct workaround for an intentional label rename.
const renameRaceHint = "\n\nThis can happen when a Terraform resource label is renamed while keeping the same `name`: " +
	"Terraform plans the rename as a parallel destroy + create that target the same Kosli resource, " +
	"and the destroy can archive the resource before this read. To rename the resource label without " +
	"recreating, use `terraform state mv` instead."
