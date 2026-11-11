package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type testKey string

func (k testKey) Validate() error {
	if k == "" {
		return models.NewGenericValidationError(errors.New("key is required"))
	}

	return nil
}

type calculatorFunc func(context.Context, rte.CalculateInput[testKey]) (int, error)

func (f calculatorFunc) Calculate(ctx context.Context, input rte.CalculateInput[testKey]) (int, error) {
	return f(ctx, input)
}

type storeStub struct {
	store func(context.Context, rte.StoreInput[testKey, int]) (rte.StoreResult[int], error)
	get   func(context.Context, testKey) (rte.State[int], error)
}

func (s storeStub) Store(ctx context.Context, input rte.StoreInput[testKey, int]) (rte.StoreResult[int], error) {
	return s.store(ctx, input)
}

func (s storeStub) Get(ctx context.Context, key testKey) (rte.State[int], error) {
	return s.get(ctx, key)
}

func TestServiceLifecycle(t *testing.T) {
	// Given a calculator that can fail and a store that records only explicit writes.
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	key := testKey("calculation")
	calls, writes, updates := 0, 0, 0
	sourceErr := errors.New("source unavailable")
	var stored *rte.State[int]
	metrics, err := newCalculationMetrics(noop.NewMeterProvider().Meter("test"), rte.CalculationKindCreditBalance)
	require.NoError(t, err)
	engine := newCalculation(Registration[testKey, int]{
		Calculator: calculatorFunc(func(_ context.Context, input rte.CalculateInput[testKey]) (int, error) {
			calls++
			require.Equal(t, key, input.Key)

			return 0, sourceErr
		}),
		Store: storeStub{
			store: func(_ context.Context, input rte.StoreInput[testKey, int]) (rte.StoreResult[int], error) {
				writes++
				require.Equal(t, key, input.Key)
				stored = &input.State

				return rte.StoreResult[int]{State: *stored, Updated: true}, nil
			},
			get: func(_ context.Context, got testKey) (rte.State[int], error) {
				require.Equal(t, key, got)
				if stored == nil {
					return rte.State[int]{}, rte.NewNotCalculatedError()
				}

				return *stored, nil
			},
		},
	}, metrics, publisherStub{publish: func(context.Context, marshaler.Event) error {
		updates++
		return nil
	}}, updatedEvent)

	// When reading before a successful calculation, no calculator or write runs.
	_, err = engine.Get(t.Context(), key)
	require.True(t, models.IsGenericNotFoundError(err))
	require.Zero(t, calls)
	require.Zero(t, writes)
	require.Zero(t, updates)
	_, err = engine.Calculate(t.Context(), key)
	require.ErrorIs(t, err, sourceErr)
	require.Zero(t, writes)
	require.Zero(t, updates)
	_, err = engine.Get(t.Context(), key)
	require.True(t, models.IsGenericNotFoundError(err))

	// Then a successful zero is stored, and a failed refresh leaves that success intact.
	sourceErr = nil
	calculated, err := engine.Calculate(t.Context(), key)
	require.NoError(t, err)
	require.Equal(t, rte.State[int]{CalculatedAt: now, Value: 0}, calculated)
	require.Equal(t, 1, writes)
	require.Equal(t, 1, updates)
	sourceErr = errors.New("refresh failed")
	_, err = engine.Calculate(t.Context(), key)
	require.ErrorIs(t, err, sourceErr)
	require.Equal(t, 1, writes)
	require.Equal(t, 1, updates)
	callsBeforeRead := calls
	result, err := engine.Get(t.Context(), key)
	require.NoError(t, err)
	require.Equal(t, calculated, result)
	require.Equal(t, callsBeforeRead, calls, "Get must not invoke the calculator")
}

func TestCalculatePreservesStartAndReturnsStoredWinner(t *testing.T) {
	// Given a calculator that finishes later and a store retaining another attempt's result.
	startedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(startedAt)
	defer clock.UnFreeze()

	ctx := t.Context()
	key := testKey("calculation")
	winner := rte.State[int]{CalculatedAt: startedAt.Add(time.Minute), Value: 100}
	metrics, err := newCalculationMetrics(noop.NewMeterProvider().Meter("test"), rte.CalculationKindCreditBalance)
	require.NoError(t, err)
	engine := newCalculation(Registration[testKey, int]{
		Calculator: calculatorFunc(func(gotCtx context.Context, input rte.CalculateInput[testKey]) (int, error) {
			require.Equal(t, ctx, gotCtx)
			require.Equal(t, rte.CalculateInput[testKey]{Key: key, At: startedAt}, input)
			clock.FreezeTime(startedAt.Add(time.Hour))
			defer clock.UnFreeze()

			return 42, nil
		}),
		Store: storeStub{store: func(gotCtx context.Context, input rte.StoreInput[testKey, int]) (rte.StoreResult[int], error) {
			require.Equal(t, ctx.Err(), gotCtx.Err())
			require.Equal(t, key, input.Key)
			require.Equal(t, rte.State[int]{CalculatedAt: startedAt, Value: 42}, input.State)

			return rte.StoreResult[int]{State: winner, Updated: false}, nil
		}},
	}, metrics, publisherStub{}, updatedEvent)

	// When calculation completes, RTE passes its original start to storage.
	result, err := engine.Calculate(ctx, key)
	require.NoError(t, err)

	// Then it returns the store's retained winner rather than its own stale output.
	require.Equal(t, winner, result)
}

func TestServiceRejectsInvalidKeys(t *testing.T) {
	// Given collaborators that record any accidental invocation.
	calls := 0
	metrics, err := newCalculationMetrics(noop.NewMeterProvider().Meter("test"), rte.CalculationKindCreditBalance)
	require.NoError(t, err)
	engine := newCalculation(Registration[testKey, int]{
		Calculator: calculatorFunc(func(context.Context, rte.CalculateInput[testKey]) (int, error) {
			calls++

			return 0, nil
		}),
		Store: storeStub{
			store: func(context.Context, rte.StoreInput[testKey, int]) (rte.StoreResult[int], error) {
				calls++

				return rte.StoreResult[int]{}, nil
			},
			get: func(context.Context, testKey) (rte.State[int], error) {
				calls++

				return rte.State[int]{}, nil
			},
		},
	}, metrics, publisherStub{}, updatedEvent)

	// When either operation receives an invalid identity.
	_, err = engine.Calculate(t.Context(), "")
	require.True(t, models.IsGenericValidationError(err))
	_, err = engine.Get(t.Context(), "")
	require.True(t, models.IsGenericValidationError(err))

	// Then validation stops execution before calculation or storage.
	require.Zero(t, calls)
}

type txStub struct{}

func (txStub) Commit() error    { return nil }
func (txStub) Rollback() error  { return nil }
func (txStub) SavePoint() error { return nil }
func (storeStub) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	return ctx, txStub{}, nil
}

type publisherStub struct {
	eventbus.Publisher
	publish func(context.Context, marshaler.Event) error
}

func (p publisherStub) Publish(ctx context.Context, event marshaler.Event) error {
	if p.publish != nil {
		return p.publish(ctx, event)
	}

	return nil
}

func updatedEvent(rte.StoreInput[testKey, int]) (rte.CalculationUpdatedEvent, error) {
	return rte.CalculationUpdatedEvent{}, nil
}
