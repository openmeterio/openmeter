// Package softdelete centralizes OpenMeter's time-windowed soft-delete
// semantics so adapters do not have to repeat
// `Or(DeletedAtIsNil(), DeletedAtGT(now))` predicates by hand.
//
// The package exposes three pieces:
//
//   - SoftDeleteMixin: an ent mixin that opts an entity into automatic
//     read-side filtering and rewrites OpDelete*/OpDeleteOne mutations into
//     OpUpdate*/OpUpdateOne mutations that stamp `deleted_at = clock.Now()`.
//   - Skip(ctx)/IsSkipped(ctx): a context-scoped opt-out for callers that
//     legitimately need to see (or operate on) tombstoned rows.
//   - Register/RunCascade: a registry for per-entity cascade walkers that
//     propagate `deleted_at` to dependents whose schemas also carry the
//     field. The registry lives in this package so the runtime mixin can
//     call into it without introducing a cycle with the generated `ent/db`
//     package.
//
// Project policy: hard delete is forbidden on entities using this mixin. The
// hook unconditionally rewrites OpDelete*; there is no opt-out for hard
// delete because we never want a SQL DELETE statement issued for a
// soft-delete entity. Edges relying on `entsql.OnDelete(entsql.Cascade)`
// between two soft-delete schemas become inert (the DB never sees a
// DELETE) and should be dropped as adoption progresses.
package softdelete

import "context"

type skipKey struct{}

// Skip returns a context that disables the soft-delete read filter for any
// query executed with it. Use this in code paths that intentionally want to
// see or operate on tombstoned rows (e.g. `IncludeDeleted=true` list
// parameters, time-windowed queries that filter `deleted_at` against a
// non-`now` reference, or admin tooling).
func Skip(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipKey{}, struct{}{})
}

// IsSkipped reports whether the soft-delete read filter has been disabled
// on this context via Skip.
func IsSkipped(ctx context.Context) bool {
	_, ok := ctx.Value(skipKey{}).(struct{})
	return ok
}

type allowHardDeleteKey struct{}

// AllowHardDelete returns a context that opts the soft-delete delete-rewrite
// hook out for the duration of the call. When the flag is set, OpDelete and
// OpDeleteOne mutations flow through unchanged and issue real SQL DELETEs;
// the in-process cascade walker does NOT run, so cleanup of dependents falls
// to the database FKs (`entsql.OnDelete(entsql.Cascade)`).
//
// Use this only in adapters that intentionally remove soft-delete-bearing
// rows physically — typically in transactional flows where the row is being
// replaced or where its physical absence is required by a unique constraint
// or downstream consumer (e.g. gathering invoice line lifecycle, plan-edit
// phase replacement).
//
// Default behavior (without this flag) is the project policy: never hard
// delete a soft-delete-bearing row.
func AllowHardDelete(ctx context.Context) context.Context {
	return context.WithValue(ctx, allowHardDeleteKey{}, struct{}{})
}

// IsHardDeleteAllowed reports whether the caller has opted into hard-delete
// behavior via AllowHardDelete.
func IsHardDeleteAllowed(ctx context.Context) bool {
	_, ok := ctx.Value(allowHardDeleteKey{}).(struct{})
	return ok
}
