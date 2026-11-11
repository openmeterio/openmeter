package demand

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
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
	config      Config
	operations  metric.Int64Counter
	invalidated metric.Int64Counter
	lastPrune   atomic.Int64
}

func New(config Config) (rte.DemandService, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if config.InvalidationRetention == 0 {
		config.InvalidationRetention = 7 * 24 * time.Hour
	}

	operations, err := config.MetricMeter.Int64Counter("rte.demand.operations", metric.WithDescription("Demand operations by operation and outcome"))
	if err != nil {
		return nil, err
	}

	invalidated, err := config.MetricMeter.Int64Counter("rte.demand.invalidated", metric.WithDescription("Active calculation keys marked dirty"))
	if err != nil {
		return nil, err
	}

	return &service{config: config, operations: operations, invalidated: invalidated}, nil
}

func (s *service) Activate(ctx context.Context, input rte.ActivateDemandInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return s.config.Repository.Activate(ctx, input)
}

func (s *service) EnsureFresh(ctx context.Context, input rte.EnsureFreshInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if err := s.config.Repository.EnsurePull(ctx, input); err != nil {
		return err
	}

	processed, err := s.process(ctx, &input.Target)
	outcome := "cache_hit"
	if processed {
		outcome = "calculated"
	}

	if err != nil {
		outcome = "error"
	}

	s.record(ctx, "pull", outcome)

	return err
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
	outcome := "marked"
	if result.Duplicate {
		outcome = "duplicate"
	}

	if err != nil {
		outcome = "error"
	}

	s.record(ctx, "invalidate", outcome)
	if result.Matched > 0 {
		s.invalidated.Add(ctx, int64(result.Matched))
	}

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

	outcome := "empty"
	if count > 0 {
		outcome = "pruned"
	}

	if err != nil {
		outcome = "error"
		s.config.Logger.WarnContext(ctx, "prune RTE invalidation receipts", "error", err)
	}

	s.record(ctx, "maintenance", outcome)
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
	outcome := "empty"
	if result.Processed {
		outcome = "calculated"
	}

	if result.CalculationError != nil {
		outcome = "retry"
	}

	if err != nil {
		outcome = "error"
	}

	s.record(ctx, "process", outcome)

	return result.Processed, errors.Join(err, result.CalculationError)
}

func (s *service) record(ctx context.Context, operation, outcome string) {
	s.operations.Add(ctx, 1, metric.WithAttributes(attribute.String("operation", operation), attribute.String("outcome", outcome)))
}
