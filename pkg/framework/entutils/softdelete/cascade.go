package softdelete

import (
	"context"
	"sync"

	"entgo.io/ent"
)

// CascadeFunc soft-deletes every dependent reachable from the parent
// identified by parentIDs.
//
// `client` is the parent mutation's `*db.Client` — passed as `any` so the
// runtime registry does not need to import the generated db package (which
// would create an import cycle). Generated walkers type-assert it to the
// concrete `*db.Client` of their package.
//
// `parentIDs` is `[]any` to accommodate the heterogeneous ID types in the
// schema (most are string ULIDs, a handful of association tables use
// `int`). Walkers type-assert each element to the parent's actual ID type
// before issuing the FK predicate.
//
// The walker is responsible for using `client` (which is bound to the
// caller's transaction when one is active) so cascade writes participate in
// the same atomic unit as the parent.
type CascadeFunc func(ctx context.Context, client any, parentIDs []any) error

var (
	cascadeMu       sync.RWMutex
	cascadeRegistry = map[string]CascadeFunc{}
)

// Register installs a cascade walker for the given mutation type name (as
// returned by ent.Mutation.Type()). It is intended to be called from
// generated `init()` blocks, one per soft-delete-bearing schema.
//
// Calling Register twice for the same type panics: cascade behavior must
// be unambiguous, and a duplicate registration is a programmer error
// (e.g. an extension running twice over the same node).
func Register(typeName string, fn CascadeFunc) {
	if fn == nil {
		return
	}
	cascadeMu.Lock()
	defer cascadeMu.Unlock()
	if _, exists := cascadeRegistry[typeName]; exists {
		panic("softdelete: cascade walker already registered for " + typeName)
	}
	cascadeRegistry[typeName] = fn
}

// RunCascade invokes the walker registered for the parent mutation's type.
// Used by the soft-delete delete-rewrite hook in mixin.go: the hook has the
// `ent.Mutation`, so it pulls `Type()` and a typed `Client()` off it via
// reflection and dispatches.
//
// `parentIDs` is `[]string` because every soft-delete-bearing top-level
// entity in OpenMeter uses string ULID IDs (the int-ID entities are all
// junction tables that are reached only as descendants, never as the
// initial soft-delete target). The hook collects IDs as `[]string` from
// `mutation.IDs()`; this entry point preserves that shape and converts to
// `[]any` for the registry signature.
//
// When no walker is registered, RunCascade is a no-op — this is the
// expected state for entities that have no outgoing soft-delete edges.
func RunCascade(ctx context.Context, m ent.Mutation, parentIDs []string) error {
	if len(parentIDs) == 0 {
		return nil
	}
	cascadeMu.RLock()
	fn, ok := cascadeRegistry[m.Type()]
	cascadeMu.RUnlock()
	if !ok {
		return nil
	}
	client, err := clientFromMutation(m)
	if err != nil {
		return err
	}
	anyIDs := make([]any, len(parentIDs))
	for i, id := range parentIDs {
		anyIDs[i] = id
	}
	return fn(ctx, client, anyIDs)
}

// RunCascadeFor invokes the walker registered for the given type. Used by
// generated walkers to recurse into a child's walker without having to
// construct a synthetic `ent.Mutation`. The caller passes the same `client`
// they hold, since cascades stay within one transaction.
//
// When no walker is registered for `typeName`, RunCascadeFor is a no-op.
func RunCascadeFor(ctx context.Context, typeName string, client any, parentIDs []any) error {
	if len(parentIDs) == 0 {
		return nil
	}
	cascadeMu.RLock()
	fn, ok := cascadeRegistry[typeName]
	cascadeMu.RUnlock()
	if !ok {
		return nil
	}
	return fn(ctx, client, parentIDs)
}
