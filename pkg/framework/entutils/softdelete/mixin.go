package softdelete

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/mixin"

	"github.com/openmeterio/openmeter/pkg/clock"
)

// SoftDeleteMixin opts an entity into centralized soft-delete handling.
//
// It contributes:
//   - Interceptors() — AND-injects the active-row predicate
//     `Or(DeletedAtIsNil(), DeletedAtGT(now))` into every query, unless
//     the caller has called Skip(ctx).
//   - Hooks() — on OpDelete and OpDeleteOne, rewrites the mutation to an
//     update that stamps `deleted_at = clock.Now()`, then runs the
//     registered cascade walker for the entity (if any).
//
// The mixin contributes no fields. Entities adopting it must already have
// a `deleted_at` column (typically via `entutils.TimeMixin`); for the
// hand-rolled case (`field.Time("deleted_at").Optional().Nillable()`) the
// mixin composes cleanly because it adds no fields of its own.
//
// Hard delete is intentionally not supported on entities using this mixin
// — there is no Skip-style escape hatch for the write path. Callers that
// want a row physically removed must operate without the mixin.
type SoftDeleteMixin struct {
	mixin.Schema
}

func (SoftDeleteMixin) Interceptors() []ent.Interceptor {
	return []ent.Interceptor{Interceptor()}
}

func (SoftDeleteMixin) Hooks() []ent.Hook {
	return []ent.Hook{deleteHook()}
}

// deleteHook returns a hook that intercepts OpDelete and OpDeleteOne and
// converts them into soft-delete updates, unless the caller has opted into
// hard delete via AllowHardDelete(ctx).
func deleteHook() ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			op := m.Op()
			if !op.Is(ent.OpDeleteOne) && !op.Is(ent.OpDelete) {
				return next.Mutate(ctx, m)
			}

			// Hard-delete escape hatch: let the original DELETE flow.
			// The cascade walker is not run here — DB-level FKs handle
			// dependent cleanup when the parent row is physically removed.
			if IsHardDeleteAllowed(ctx) {
				return next.Mutate(ctx, m)
			}

			mx, ok := m.(softDeleteMutation)
			if !ok {
				return nil, fmt.Errorf("softdelete: mutation %T does not satisfy SetOp/SetDeletedAt/WhereP", m)
			}

			ids, err := collectIDs(ctx, m)
			if err != nil {
				return nil, fmt.Errorf("softdelete: collecting ids for %s: %w", m.Type(), err)
			}

			// Constrain the upcoming UPDATE to currently-active rows so
			// we never re-stamp `deleted_at` on a tombstone, and switch
			// the operation into the update path so UpdateDefault hooks
			// (e.g. TimeMixin's `updated_at`) fire correctly.
			mx.WhereP(ActivePredicate(clock.Now()))
			if op.Is(ent.OpDeleteOne) {
				mx.SetOp(ent.OpUpdateOne)
			} else {
				mx.SetOp(ent.OpUpdate)
			}
			mx.SetDeletedAt(clock.Now())

			// Re-dispatch through the client so the rewritten op flows
			// through the regular OpUpdate*/OpUpdateOne hook pipeline.
			value, err := dispatchViaClient(ctx, m)
			if err != nil {
				return nil, err
			}

			if err := RunCascade(ctx, m, ids); err != nil {
				return nil, fmt.Errorf("softdelete: cascade for %s: %w", m.Type(), err)
			}
			return value, nil
		})
	}
}

// softDeleteMutation is the structural surface every generated mutation
// implements. We assert it lazily so the mixin does not need to import any
// generated package.
type softDeleteMutation interface {
	SetOp(ent.Op)
	SetDeletedAt(time.Time)
	WhereP(...func(*sql.Selector))
}

// collectIDs reads the ID set of the rows about to be soft-deleted.
// Generated mutations expose `IDs(ctx) ([]<idtype>, error)`. OpenMeter
// uses string ULIDs everywhere, so we type-assert to that.
func collectIDs(ctx context.Context, m ent.Mutation) ([]string, error) {
	type stringIDs interface {
		IDs(context.Context) ([]string, error)
	}
	if l, ok := m.(stringIDs); ok {
		return l.IDs(ctx)
	}
	// Some entities may use other ID types (e.g. CustomerSubjects has a
	// composite key). Fall back to reflection so the cascade gets an
	// empty list rather than a crash; cascade itself is a no-op when no
	// walker is registered.
	v := reflect.ValueOf(m)
	method := v.MethodByName("IDs")
	if !method.IsValid() {
		return nil, nil
	}
	out := method.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(out) != 2 {
		return nil, nil
	}
	if errVal := out[1].Interface(); errVal != nil {
		if e, ok := errVal.(error); ok && e != nil {
			return nil, e
		}
	}
	return nil, nil
}

// dispatchViaClient calls m.Client().Mutate(ctx, m). The generated
// `Client()` method returns the package-specific *Client, which we cannot
// reference from this generic mixin; reflection lets us reach it without
// importing the db package (and creating an import cycle).
func dispatchViaClient(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	client, err := clientFromMutation(m)
	if err != nil {
		return nil, err
	}
	mutator, ok := client.(ent.Mutator)
	if !ok {
		return nil, fmt.Errorf("softdelete: mutation %T Client() did not return ent.Mutator", m)
	}
	return mutator.Mutate(ctx, m)
}

// clientFromMutation extracts the package-specific *db.Client from a
// mutation via reflection. Returned as `any` so callers (the cascade
// registry, generated walkers) can use it without the softdelete package
// importing the generated db package.
func clientFromMutation(m ent.Mutation) (any, error) {
	v := reflect.ValueOf(m)
	method := v.MethodByName("Client")
	if !method.IsValid() {
		return nil, fmt.Errorf("softdelete: mutation %T has no Client() method", m)
	}
	out := method.Call(nil)
	if len(out) != 1 {
		return nil, fmt.Errorf("softdelete: mutation %T Client() returned %d values", m, len(out))
	}
	return out[0].Interface(), nil
}
