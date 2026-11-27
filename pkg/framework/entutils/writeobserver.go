package entutils

import (
	"context"
	"database/sql"

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

// WriteObserverInterceptor reports queries on entType that take row locks (FOR
// UPDATE / FOR SHARE) to the write observer of the context. A read replica
// rejects those, so they need the primary just like mutations do.
//
// It reports while the query is prepared, not when it is executed: a locked query
// that is only the starting point of a traversal (Query().ForUpdate().QueryEdge())
// is never executed on its own, but its lock is still part of the statement.
//
// The generated clients install one per entity, because the query context cannot
// tell which rows are locked: an eager-loaded or traversed query inherits it from
// the query that runs it, so it would name the wrong entity.
func WriteObserverInterceptor(entType string) ent.Interceptor {
	return ent.TraverseFunc(func(ctx context.Context, q ent.Query) error {
		locking, ok := q.(rowLockingQuery)
		if !ok || !locking.LocksRows() {
			return nil
		}

		attempt := transaction.WriteAttempt{Kind: transaction.WriteKindRowLock, Target: entType}

		// For an eager-loaded or traversed query this is the operation of the query that runs it.
		if qc := ent.QueryFromContext(ctx); qc != nil {
			attempt.Operation = qc.Op
		}

		return transaction.ReportWrite(ctx, attempt)
	})
}

// TxOptionsForContext makes the transaction READ ONLY when the write observer of
// ctx rejects every write. The hook and the interceptor only see what goes
// through Ent, so this is what stops hand-written SQL that nobody reported.
func TxOptionsForContext(ctx context.Context, opts *sql.TxOptions) *sql.TxOptions {
	if !transaction.RejectsWrites(ctx) {
		return opts
	}

	readOnly := sql.TxOptions{ReadOnly: true}
	if opts != nil {
		readOnly.Isolation = opts.Isolation
	}

	return &readOnly
}
