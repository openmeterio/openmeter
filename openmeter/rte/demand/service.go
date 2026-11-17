package demand

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	rtemetrics "github.com/openmeterio/openmeter/openmeter/rte/metrics"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Repository  rte.DemandRepository
	Service     rte.Service
	Logger      *slog.Logger
	MetricMeter metric.Meter

	// Zero uses seven days. Replays beyond this window can trigger extra work.
	InvalidationRetention time.Duration
	CalculationTimeout    time.Duration
	RetryMinDelay         time.Duration
	RetryMaxDelay         time.Duration
}

func (c Config) Validate() error {
	var errs []error
	if c.Repository == nil {
		errs = append(errs, errors.New("demand repository is required"))
	}

	if c.Service == nil {
		errs = append(errs, errors.New("calculation service is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.MetricMeter == nil {
		errs = append(errs, errors.New("metric meter is required"))
	}

	if c.RetryMinDelay <= 0 || c.RetryMaxDelay < c.RetryMinDelay {
		errs = append(errs, errors.New("retry delays must be positive and max must be at least min"))
	}

	if c.InvalidationRetention < 0 {
		errs = append(errs, errors.New("invalidation retention must not be negative"))
	}

	if c.CalculationTimeout <= 0 {
		errs = append(errs, errors.New("calculation timeout must be positive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type service struct {
	config    Config
	metrics   *rtemetrics.Demand
	lastPrune atomic.Int64
}

func New(config Config) (rte.DemandService, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if config.InvalidationRetention == 0 {
		config.InvalidationRetention = 7 * 24 * time.Hour
	}

	metrics, err := rtemetrics.NewDemand(config.MetricMeter)
	if err != nil {
		return nil, err
	}

	return &service{config: config, metrics: metrics}, nil
}

func (s *service) Activate(ctx context.Context, target rte.CalculationTarget) error {
	if err := target.Validate(); err != nil {
		return err
	}

	return s.config.Repository.Activate(ctx, target)
}

func (s *service) EnsureFresh(ctx context.Context, input rte.EnsureFreshInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	start := time.Now()
	outcome := rtemetrics.DemandOutcomeError
	defer func() {
		s.metrics.EnsureFresh(ctx, input.Target.Kind, start, outcome)
	}()

	clean, err := s.config.Repository.EnsurePull(ctx, input)
	if err != nil {
		return err
	}

	if clean {
		outcome = rtemetrics.DemandOutcomeCacheHit
		return nil
	}

	processed, err := s.process(ctx, &input.Target)
	if err != nil {
		return err
	}

	outcome = rtemetrics.DemandOutcomeCacheHit
	if processed {
		outcome = rtemetrics.DemandOutcomeCalculated
	}

	return nil
}

func (s *service) Deactivate(ctx context.Context, target rte.CalculationTarget) error {
	if err := target.Validate(); err != nil {
		return err
	}

	return s.config.Repository.Deactivate(ctx, target)
}

func (s *service) Invalidate(ctx context.Context, input rte.InvalidateInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	result, err := s.config.Repository.Invalidate(ctx, input)
	outcome := rtemetrics.DemandOutcomeMarked
	if result.Duplicate {
		outcome = rtemetrics.DemandOutcomeDuplicate
	}

	if err != nil {
		outcome = rtemetrics.DemandOutcomeError
	}

	s.metrics.Operation(ctx, rtemetrics.DemandOperationInvalidate, outcome)
	s.metrics.Invalidated(ctx, result.Matched)

	return err
}

func (s *service) Process(ctx context.Context) (bool, error) {
	s.pruneReceipts(ctx)
	return s.process(ctx, nil)
}

func (s *service) pruneReceipts(ctx context.Context) {
	now := clock.Now()
	previous := s.lastPrune.Load()
	if previous != 0 && now.UnixNano()-previous < int64(time.Minute) {
		return
	}

	if !s.lastPrune.CompareAndSwap(previous, now.UnixNano()) {
		return
	}

	count, err := s.config.Repository.PruneInvalidations(ctx, rte.PruneInvalidationsInput{Before: now.Add(-s.config.InvalidationRetention), Limit: 1000})
	if err == nil && count == 1000 {
		// Drain an accumulated backlog without adding maintenance to every poll.
		s.lastPrune.CompareAndSwap(now.UnixNano(), now.Add(-time.Minute+time.Second).UnixNano())
	}

	outcome := rtemetrics.DemandOutcomeEmpty
	if count > 0 {
		outcome = rtemetrics.DemandOutcomePruned
	}

	if err != nil {
		outcome = rtemetrics.DemandOutcomeError
		s.config.Logger.WarnContext(ctx, "prune RTE invalidation receipts", "error", err)
	}

	s.metrics.Operation(ctx, rtemetrics.DemandOperationMaintenance, outcome)
}

func (s *service) process(ctx context.Context, target *rte.CalculationTarget) (bool, error) {
	result, err := s.config.Repository.Process(ctx, rte.ProcessDemandInput{
		Target: target, CalculationTimeout: s.config.CalculationTimeout, RetryMinDelay: s.config.RetryMinDelay, RetryMaxDelay: s.config.RetryMaxDelay,
	}, func(ctx context.Context, target rte.CalculationTarget) error {
		key, err := target.DecodeKey()
		if err != nil {
			return err
		}

		switch key := key.(type) {
		case creditbalance.Key:
			_, err = s.config.Service.CreditBalances().Calculate(ctx, key)
		case meteredentitlement.Key:
			_, err = s.config.Service.MeteredEntitlements().Calculate(ctx, key)
		default:
			return fmt.Errorf("unsupported calculation key %T", key)
		}

		return err
	})
	outcome := rtemetrics.DemandOutcomeEmpty
	switch {
	case err != nil:
		outcome = rtemetrics.DemandOutcomeError
	case result.Retired:
		outcome = rtemetrics.DemandOutcomeRetired
	case result.CalculationError != nil:
		outcome = rtemetrics.DemandOutcomeRetry
	case result.Processed:
		outcome = rtemetrics.DemandOutcomeCalculated
	}

	s.metrics.Operation(ctx, rtemetrics.DemandOperationProcess, outcome)

	return result.Processed && err == nil, errors.Join(err, result.CalculationError)
}
