package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Registration pairs a calculator with the store for its key and result types.
type Registration[K rte.Key, Result any] struct {
	Calculator rte.Calculator[K, Result]
	Store      rte.Store[K, Result]
}

func (r Registration[K, Result]) Validate() error {
	var errs []error

	if r.Calculator == nil {
		errs = append(errs, errors.New("calculator is required"))
	}

	if r.Store == nil {
		errs = append(errs, errors.New("store is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type calculation[K rte.Key, Result any] struct {
	calculator rte.Calculator[K, Result]
	store      rte.Store[K, Result]
}

func newCalculation[K rte.Key, Result any](r Registration[K, Result]) rte.Calculation[K, Result] {
	return &calculation[K, Result]{calculator: r.Calculator, store: r.Store}
}

func (c *calculation[K, Result]) Calculate(ctx context.Context, key K) (rte.State[Result], error) {
	if err := key.Validate(); err != nil {
		return rte.State[Result]{}, err
	}

	calculatedAt := clock.Now()
	value, err := c.calculator.Calculate(ctx, rte.CalculateInput[K]{Key: key, At: calculatedAt})
	if err != nil {
		return rte.State[Result]{}, fmt.Errorf("calculate: %w", err)
	}

	return c.store.Store(ctx, rte.StoreInput[K, Result]{
		Key:   key,
		State: rte.State[Result]{CalculatedAt: calculatedAt, Value: value},
	})
}

func (c *calculation[K, Result]) Get(ctx context.Context, key K) (rte.State[Result], error) {
	if err := key.Validate(); err != nil {
		return rte.State[Result]{}, err
	}

	return c.store.Get(ctx, key)
}
