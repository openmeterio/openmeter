package transaction

import (
	"context"
	"errors"
)

// WriteKind says how a write reached the database.
type WriteKind string

const (
	// WriteKindMutation is an Ent create, update or delete.
	WriteKindMutation WriteKind = "mutation"
	// WriteKindRowLock is a SELECT ... FOR UPDATE / FOR SHARE.
	WriteKindRowLock WriteKind = "row_lock"
	// WriteKindAdvisoryLock is a PostgreSQL advisory lock. A read replica grants
	// these without coordinating with the primary, so they never fail on their own.
	WriteKindAdvisoryLock WriteKind = "advisory_lock"
	// WriteKindRawSQL is a hand-written statement that Ent cannot classify.
	WriteKindRawSQL WriteKind = "raw_sql"
)

// WriteAttempt describes one database write that needs the primary.
type WriteAttempt struct {
	Kind WriteKind
	// Target is the Ent type for mutations and row locks, and the lock key for
	// advisory locks. Lock keys are unbounded, so do not use Target as a metric label.
	Target string
	// Operation is the Ent operation for mutations (for example "OpCreate").
	Operation string
}

// ReadOnlyObserver is told about every write attempted inside a read-only scope.
// Returning an error rejects the write.
type ReadOnlyObserver interface {
	ObserveWrite(ctx context.Context, attempt WriteAttempt) error
}

type readOnlyContextKey struct{}

// WithReadOnly marks ctx as a scope that is expected to only read, so it could
// be served by a read replica. Writes are still executed: the observer decides
// whether they are only recorded or rejected.
func WithReadOnly(ctx context.Context, observer ReadOnlyObserver) (context.Context, error) {
	if observer == nil {
		return ctx, errors.New("read-only observer is required")
	}

	return context.WithValue(ctx, readOnlyContextKey{}, observer), nil
}

func IsReadOnly(ctx context.Context) bool {
	_, ok := ctx.Value(readOnlyContextKey{}).(ReadOnlyObserver)

	return ok
}

// ReportWrite must be called before any database write that Ent hooks cannot
// see, such as raw SQL or advisory locks. It does nothing outside a read-only
// scope.
func ReportWrite(ctx context.Context, attempt WriteAttempt) error {
	observer, ok := ctx.Value(readOnlyContextKey{}).(ReadOnlyObserver)
	if !ok {
		return nil
	}

	return observer.ObserveWrite(ctx, attempt)
}
