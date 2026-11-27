package testutils

import (
	"context"
	"slices"
	"sync"

	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ transaction.ReadOnlyObserver = (*ReadOnlyRecorder)(nil)

// ReadOnlyRecorder collects the writes attempted inside a read-only scope.
type ReadOnlyRecorder struct {
	// Reject, when set, is returned for every write so the caller sees it fail.
	Reject error

	mu       sync.Mutex
	attempts []transaction.WriteAttempt
}

func (r *ReadOnlyRecorder) ObserveWrite(_ context.Context, attempt transaction.WriteAttempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.attempts = append(r.attempts, attempt)

	return r.Reject
}

func (r *ReadOnlyRecorder) Attempts() []transaction.WriteAttempt {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.attempts)
}
