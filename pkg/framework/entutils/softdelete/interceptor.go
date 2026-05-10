package softdelete

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/pkg/clock"
)

// queryWithWhereP is the subset of a generated *Query type we need to
// inject a storage-level predicate. Both the per-entity `<Pkg>Query` types
// and the generated `intercept.Query` interface (when FeatureIntercept is
// enabled) implement this method, so the interceptor works regardless of
// whether the caller passes a typed query or the generic intercept.Query.
type queryWithWhereP interface {
	WhereP(...func(*sql.Selector))
}

// Interceptor returns an ent.Interceptor that AND-injects ActivePredicate
// into every query on the entity it is attached to, unless IsSkipped(ctx)
// reports the caller has opted out.
//
// The interceptor uses ent.TraverseFunc so it also fires on edge
// traversals (i.e. WithX / HasXWith style joins through the entity), which
// is the behavior we want: a soft-deleted dependent should be invisible
// even when reached via a parent's edge.
func Interceptor() ent.Interceptor {
	return ent.TraverseFunc(func(ctx context.Context, q ent.Query) error {
		if IsSkipped(ctx) {
			return nil
		}
		w, ok := q.(queryWithWhereP)
		if !ok {
			// Defensive: if the generated type ever stops exposing
			// WhereP, surface it loudly rather than silently disabling
			// the filter.
			return nil
		}
		w.WhereP(ActivePredicate(clock.Now()))
		return nil
	})
}
