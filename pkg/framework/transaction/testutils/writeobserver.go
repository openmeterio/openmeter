package testutils

import (
	"context"
	"slices"
	"sync"

	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ transaction.WriteObserver = (*WriteRecorder)(nil)

// WriteRecorder collects the writes attempted with the context it is attached to.
type WriteRecorder struct {
	// Reject, when set, is returned for every write so the caller sees it fail.
	Reject error

	mu       sync.Mutex
	attempts []transaction.WriteAttempt
}

func (r *WriteRecorder) ObserveWrite(_ context.Context, attempt transaction.WriteAttempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.attempts = append(r.attempts, attempt)

	return r.Reject
}

func (r *WriteRecorder) RejectsWrites() bool {
	return r.Reject != nil
}

func (r *WriteRecorder) Attempts() []transaction.WriteAttempt {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.attempts)
}
