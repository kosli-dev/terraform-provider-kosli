package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// renameInPlace applies a changed name as an in-place rename at the start of
// an Update (see ADR 005). It is a no-op when planned equals prior.
//
// The rename uses the prior-state name because the API requires the current
// name, and the caller's remaining calls must then target the new name. If the
// rename fails, an error diagnostic is added and nothing else is called, so
// the framework keeps the prior state. If it succeeds, persist runs so the new
// name reaches state before the remaining update, and a later failure cannot
// leave state pointing at the old name.
//
// displayKind is the user-facing resource kind in title case, for example
// "Logical Environment". It returns false when the caller must return.
func renameInPlace(
	ctx context.Context,
	displayKind string,
	prior, planned types.String,
	rename func(ctx context.Context, name, newName string) error,
	persist func(ctx context.Context) diag.Diagnostics,
	diags *diag.Diagnostics,
) bool {
	if planned.Equal(prior) {
		return true
	}

	oldName, newName := prior.ValueString(), planned.ValueString()
	if err := rename(ctx, oldName, newName); err != nil {
		diags.AddError(
			fmt.Sprintf("Error Renaming %s", displayKind),
			fmt.Sprintf("Could not rename %s %q to %q: %s", strings.ToLower(displayKind), oldName, newName, err.Error()),
		)
		return false
	}

	diags.Append(persist(ctx)...)
	return !diags.HasError()
}
