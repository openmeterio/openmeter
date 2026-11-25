package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/rte"
	rtemetrics "github.com/openmeterio/openmeter/openmeter/rte/metrics"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
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
	metrics      rtemetrics.Calculation
	calculator   rte.Calculator[K, Result]
	store        rte.Store[K, Result]
	publisher    eventbus.Publisher
	updatedEvent func(rte.StoreInput[K, Result]) (rte.CalculationUpdatedEvent, error)
}

func newCalculation[K rte.Key, Result any](r Registration[K, Result], metrics rtemetrics.Calculation, publisher eventbus.Publisher, updatedEvent func(rte.StoreInput[K, Result]) (rte.CalculationUpdatedEvent, error)) *calculation[K, Result] {
	return &calculation[K, Result]{
		metrics:      metrics,
		calculator:   r.Calculator,
		store:        r.Store,
		publisher:    publisher,
		updatedEvent: updatedEvent,
	}
}

func (c *calculation[K, Result]) Calculate(ctx context.Context, key K) (rte.State[Result], error) {
	return c.calculate(ctx, key, false)
}

func (c *calculation[K, Result]) calculate(ctx context.Context, key K, useCheckpoints bool) (rte.State[Result], error) {
	if err := key.Validate(); err != nil {
		return rte.State[Result]{}, err
	}

	started := time.Now()
	outcome := rtemetrics.CalculationOutcomeFailed
	method := rtemetrics.CalculationMethodFull
	defer func() {
		c.metrics.Calculated(ctx, started, outcome, method)
	}()

	input := rte.CalculateInput[K]{Key: key, At: clock.Now()}
	write, method, err := c.compute(ctx, input, useCheckpoints)
	if err != nil {
		return rte.State[Result]{}, fmt.Errorf("calculate: %w", err)
	}

	updated := false
	state, err := transaction.Run(ctx, c.store, func(ctx context.Context) (rte.State[Result], error) {
		result, err := c.store.Store(ctx, write)
		if err != nil {
			return rte.State[Result]{}, err
		}

		updated = result.Updated
		if updated {
			event, err := c.updatedEvent(rte.StoreInput[K, Result]{Key: key, State: result.State})
			if err != nil {
				return rte.State[Result]{}, err
			}

			if err := c.publisher.Publish(ctx, event); err != nil {
				return rte.State[Result]{}, fmt.Errorf("publish calculation update: %w", err)
			}
		}

		return result.State, nil
	})
	if err == nil {
		outcome = rtemetrics.CalculationOutcomeSuperseded
		if updated {
			outcome = rtemetrics.CalculationOutcomeUpdated
		}
	}

	return state, err
}

func (c *calculation[K, Result]) compute(ctx context.Context, input rte.CalculateInput[K], useCheckpoints bool) (rte.StoreInput[K, Result], rtemetrics.CalculationMethod, error) {
	write := rte.StoreInput[K, Result]{Key: input.Key, State: rte.State[Result]{CalculatedAt: input.At}}
	calculator, ok := c.calculator.(rte.IncrementalCalculator[K, Result])
	if !useCheckpoints || !ok {
		value, err := c.calculator.Calculate(ctx, input)
		write.State.Value = value

		return write, rtemetrics.CalculationMethodFull, err
	}

	history, err := c.store.GetCheckpoints(ctx, input.Key)
	if err != nil {
		return write, rtemetrics.CalculationMethodMiss, fmt.Errorf("get checkpoints: %w", err)
	}

	candidates := history.Candidates(input.At)
	result, err := calculator.CalculateIncremental(ctx, input, candidates)
	method := rtemetrics.CalculationMethodMiss
	switch result.Use {
	case rte.CheckpointReused:
		method = rtemetrics.CalculationMethodHit
	case rte.CheckpointAdvanced:
		method = rtemetrics.CalculationMethodAdvanced
	}

	write.State.Value = result.Value
	if err != nil || (result.Checkpoint == nil && len(result.Discarded) == 0) {
		return write, method, err
	}

	next := slices.DeleteFunc(candidates, func(cp rte.Checkpoint) bool {
		return slices.ContainsFunc(result.Discarded, cp.At.Equal)
	})
	if result.Checkpoint != nil {
		// Prefer the fresh payload when the calculator rebuilt an existing boundary.
		next = append(rte.Checkpoints{*result.Checkpoint}, next...).Candidates(input.At)
	}

	write.Checkpoints = &next

	return write, method, nil
}

func (c *calculation[K, Result]) Get(ctx context.Context, key K) (rte.State[Result], error) {
	if err := key.Validate(); err != nil {
		return rte.State[Result]{}, err
	}

	start := time.Now()
	state, err := c.store.Get(ctx, key)
	switch {
	case err == nil:
		c.metrics.Read(ctx, start, rtemetrics.ReadOutcomeHit)
	case models.IsGenericNotFoundError(err):
		c.metrics.Read(ctx, start, rtemetrics.ReadOutcomeMissing)
	default:
		c.metrics.Read(ctx, start, rtemetrics.ReadOutcomeFailed)
	}

	return state, err
}
