package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// withLockedNamespace runs fn in a transaction that holds the namespace's
// periodic realization lock. The lock is released when the transaction ends, so
// when the caller is already in a transaction it is held until that one ends.
func (s *service) withLockedNamespace[T any](ctx context.Context, namespace string, fn func(ctx context.Context) (T, error)) (T, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (T, error) {
		var empty T

		lockKey, err := lockr.NewKey("namespace", namespace, "periodic_realization_runs")
		if err != nil {
			return empty, fmt.Errorf("periodic realization runs lock key: %w", err)
		}

		if err := s.locker.LockForTX(ctx, lockKey); err != nil {
			return empty, fmt.Errorf("lock periodic realization runs: %w", err)
		}

		return fn(ctx)
	})
}
