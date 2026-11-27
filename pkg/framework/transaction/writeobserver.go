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

// WriteObserver is told about every database write attempted with the context it
// is attached to.
type WriteObserver interface {
	// ObserveWrite is called before the write. Returning an error rejects it.
	ObserveWrite(ctx context.Context, attempt WriteAttempt) error
	// RejectsWrites says whether ObserveWrite rejects every write, so it can be
	// known before one is attempted (see RejectsWrites).
	RejectsWrites() bool
}

type writeObserverContextKey struct{}

// WithWriteObserver attaches observer to ctx, so it is told about every database
// write attempted with that context. Use it for work that is expected to only
// read, such as a request that could be served by a read replica. Writes are
// still executed: the observer decides whether they are only recorded or rejected.
func WithWriteObserver(ctx context.Context, observer WriteObserver) (context.Context, error) {
	if observer == nil {
		return ctx, errors.New("write observer is required")
	}

	return context.WithValue(ctx, writeObserverContextKey{}, observer), nil
}

// RejectsWrites reports whether ctx has a write observer that rejects every
// write. Transactions started with such a context are opened READ ONLY, so
// inside them the database also refuses the writes that nothing reports.
func RejectsWrites(ctx context.Context) bool {
	observer, ok := ctx.Value(writeObserverContextKey{}).(WriteObserver)

	return ok && observer.RejectsWrites()
}

// ReportWrite must be called before any database write that Ent hooks cannot
// see, such as raw SQL or advisory locks. It does nothing when ctx has no write
// observer.
func ReportWrite(ctx context.Context, attempt WriteAttempt) error {
	observer, ok := ctx.Value(writeObserverContextKey{}).(WriteObserver)
	if !ok {
		return nil
	}

	return observer.ObserveWrite(ctx, attempt)
}
