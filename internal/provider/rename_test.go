package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRenameInPlace_NoOpWhenUnchanged(t *testing.T) {
	var diags diag.Diagnostics
	called := false

	ok := renameInPlace(context.TODO(), "Flow", types.StringValue("same"), types.StringValue("same"),
		func(context.Context, string, string) error { called = true; return nil },
		func(context.Context) diag.Diagnostics { called = true; return nil },
		&diags)

	if !ok || diags.HasError() {
		t.Fatalf("expected success with no diagnostics, got ok=%v diags=%v", ok, diags)
	}
	if called {
		t.Error("expected neither rename nor persist to be called when the name is unchanged")
	}
}

func TestRenameInPlace_RenamesThenPersists(t *testing.T) {
	var diags diag.Diagnostics
	var order []string

	ok := renameInPlace(context.TODO(), "Flow", types.StringValue("old"), types.StringValue("new"),
		func(_ context.Context, name, newName string) error {
			order = append(order, "rename "+name+"->"+newName)
			return nil
		},
		func(context.Context) diag.Diagnostics { order = append(order, "persist"); return nil },
		&diags)

	if !ok || diags.HasError() {
		t.Fatalf("expected success, got ok=%v diags=%v", ok, diags)
	}
	if len(order) != 2 || order[0] != "rename old->new" || order[1] != "persist" {
		t.Errorf("unexpected call order: %v", order)
	}
}

func TestRenameInPlace_RenameError_SkipsPersist(t *testing.T) {
	var diags diag.Diagnostics
	persisted := false

	ok := renameInPlace(context.TODO(), "Logical Environment", types.StringValue("old"), types.StringValue("new"),
		func(context.Context, string, string) error { return errors.New("name taken") },
		func(context.Context) diag.Diagnostics { persisted = true; return nil },
		&diags)

	if ok {
		t.Error("expected false after a failed rename")
	}
	if persisted {
		t.Error("expected persist not to run after a failed rename")
	}
	if !diags.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	got := diags.Errors()[0]
	if got.Summary() != "Error Renaming Logical Environment" {
		t.Errorf("unexpected summary %q", got.Summary())
	}
	if want := `Could not rename logical environment "old" to "new": name taken`; got.Detail() != want {
		t.Errorf("unexpected detail %q, want %q", got.Detail(), want)
	}
}

func TestRenameInPlace_PersistError(t *testing.T) {
	var diags diag.Diagnostics

	ok := renameInPlace(context.TODO(), "Environment", types.StringValue("old"), types.StringValue("new"),
		func(context.Context, string, string) error { return nil },
		func(context.Context) diag.Diagnostics {
			var d diag.Diagnostics
			d.AddError("Persist Failed", "boom")
			return d
		},
		&diags)

	if ok {
		t.Error("expected false when persisting the new name fails")
	}
	if !diags.HasError() || diags.Errors()[0].Summary() != "Persist Failed" {
		t.Errorf("expected persist diagnostics to be surfaced, got %v", diags)
	}
}
