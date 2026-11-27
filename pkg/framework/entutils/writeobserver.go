package entutils

import (
	"context"

	"entgo.io/ent"

	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// WriteObserverHook reports every Ent mutation to the write observer of the
// context (see transaction.WithWriteObserver). The generated clients install it
// themselves (see expose.tpl).
func WriteObserverHook(next ent.Mutator) ent.Mutator {
	return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
		err := transaction.ReportWrite(ctx, transaction.WriteAttempt{
			Kind:      transaction.WriteKindMutation,
			Target:    m.Type(),
			Operation: m.Op().String(),
		})
		if err != nil {
			return nil, err
		}

		return next.Mutate(ctx, m)
	})
}

// rowLockingQuery is implemented by the generated query builders (see expose.tpl).
type rowLockingQuery interface {
	LocksRows() bool
}

// WriteObserverInterceptor reports queries that take row locks (FOR UPDATE / FOR
// SHARE) to the write observer of the context. A read replica rejects those, so
// they need the primary just like mutations do.
var WriteObserverInterceptor ent.Interceptor = ent.InterceptFunc(func(next ent.Querier) ent.Querier {
	return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
		if locking, ok := q.(rowLockingQuery); ok && locking.LocksRows() {
			attempt := transaction.WriteAttempt{Kind: transaction.WriteKindRowLock}
			if qc := ent.QueryFromContext(ctx); qc != nil {
				attempt.Target = qc.Type
				attempt.Operation = qc.Op
			}

			if err := transaction.ReportWrite(ctx, attempt); err != nil {
				return nil, err
			}
		}

		return next.Query(ctx, q)
	})
})
