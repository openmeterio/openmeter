package balanceworker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker/filters"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

type FilterScope string

const (
	FilterScopeNamespace   FilterScope = "namespace"
	FilterScopeEntitlement FilterScope = "entitlement"
)

const (
	metricNameEntitlementsFilterRequestsTotal = "balance_worker.filter.requests_total"
	metricNameEntitlementsFilterMatchesTotal  = "balance_worker.filter.matches_total"
	metricNameEntitlementsFilterFilteredTotal = "balance_worker.filter.filtered_total"
	metricNameEntitlementsFilterErrorsTotal   = "balance_worker.filter.errors_total"

	metricLabelScope = "filter.scope"

	metricLabelFilterName = "filter.name"
	metricLabelNamespace  = "namespace"
)

var _ models.Validator = (*EntitlementFiltersConfig)(nil)

type EntitlementFiltersConfig struct {
	NotificationService notification.Service
	MetricMeter         metric.Meter
	StateStorage        FilterStateStorage
	Logger              *slog.Logger
}

func (c EntitlementFiltersConfig) Validate() error {
	var errs []error

	if c.NotificationService == nil {
		errs = append(errs, fmt.Errorf("notification service is required"))
	}

	if c.MetricMeter == nil {
		errs = append(errs, fmt.Errorf("metric meter is required"))
	}

	if err := c.StateStorage.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state storage: %w", err))
	}

	if c.Logger == nil {
		errs = append(errs, fmt.Errorf("logger is required"))
	}

	return errors.Join(errs...)
}

var _ filters.Filter = (*EntitlementFilters)(nil)

type EntitlementFilters struct {
	filters []filters.NamedFilter

	meterEntitlementsFilterRequestsTotal metric.Int64Counter
	meterEntitlementsFilterMatchesTotal  metric.Int64Counter
	meterEntitlementsFilterFilteredTotal metric.Int64Counter
	meterEntitlementsFilterErrorsTotal   metric.Int64Counter
}

func NewEntitlementFilters(cfg EntitlementFiltersConfig) (*EntitlementFilters, error) {
	notificationFilter, err := filters.NewNotificationsFilter(filters.NotificationsFilterConfig{
		NotificationService: cfg.NotificationService,
		CacheTTL:            defaultCacheTTL,
		CacheSize:           defaultLRUCacheSize,
	})
	if err != nil {
		return nil, err
	}

	var highWatermarkCacheBackend filters.HighWatermarkBackend

	switch cfg.StateStorage.Driver() {
	case FilterStateStorageDriverRedis:
		redis, err := cfg.StateStorage.Redis()
		if err != nil {
			return nil, err
		}

		highWatermarkCacheBackend, err = filters.NewHighWatermarkRedisBackend(filters.HighWatermarkRedisBackendConfig{
			Redis:      redis.Client,
			Logger:     cfg.Logger,
			Expiration: redis.Expiration,
		})
		if err != nil {
			return nil, err
		}
	case FilterStateStorageDriverInMemory:
		highWatermarkCacheBackend, err = filters.NewHighWatermarkInMemoryBackend(defaultLRUCacheSize)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported state storage driver: %s", cfg.StateStorage.Driver())
	}

	highWatermarkCache, err := filters.NewHighWatermarkCache(highWatermarkCacheBackend)
	if err != nil {
		return nil, err
	}

	return EntitlementFilters{
		filters: []filters.NamedFilter{notificationFilter, highWatermarkCache},
	}.WithMetrics(cfg.MetricMeter)
}

func (f EntitlementFilters) WithMetrics(meter metric.Meter) (*EntitlementFilters, error) {
	var err error

	res := &f

	res.meterEntitlementsFilterRequestsTotal, err = meter.Int64Counter(metricNameEntitlementsFilterRequestsTotal)
	if err != nil {
		return nil, err
	}
	res.meterEntitlementsFilterMatchesTotal, err = meter.Int64Counter(metricNameEntitlementsFilterMatchesTotal)
	if err != nil {
		return nil, err
	}
	res.meterEntitlementsFilterFilteredTotal, err = meter.Int64Counter(metricNameEntitlementsFilterFilteredTotal)
	if err != nil {
		return nil, err
	}

	res.meterEntitlementsFilterErrorsTotal, err = meter.Int64Counter(metricNameEntitlementsFilterErrorsTotal)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (f EntitlementFilters) IsNamespaceInScope(ctx context.Context, namespace string) (bool, error) {
	f.meterEntitlementsFilterRequestsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String(metricLabelScope, string(FilterScopeNamespace)),
		),
	)

	return f.executeFilters(ctx, func(ctx context.Context, filter filters.Filter) (bool, error) {
		return filter.IsNamespaceInScope(ctx, namespace)
	}, FilterScopeNamespace)
}

func (f EntitlementFilters) IsEntitlementInScope(ctx context.Context, req filters.EntitlementFilterRequest) (bool, error) {
	f.meterEntitlementsFilterRequestsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String(metricLabelScope, string(FilterScopeEntitlement)),
			attribute.String(metricLabelNamespace, req.Entitlement.Namespace),
		),
	)

	return f.executeFilters(ctx,
		func(ctx context.Context, filter filters.Filter) (bool, error) {
			return filter.IsEntitlementInScope(ctx, req)
		},
		FilterScopeEntitlement,
		attribute.String(metricLabelNamespace, req.Entitlement.Namespace),
	)
}

func (f EntitlementFilters) executeFilters(ctx context.Context, check func(ctx context.Context, filter filters.Filter) (bool, error), scope FilterScope, additionalAttributes ...attribute.KeyValue) (bool, error) {
	for _, filter := range f.filters {
		attributes := []attribute.KeyValue{
			attribute.String(metricLabelFilterName, filter.Name()),
			attribute.String(metricLabelScope, string(scope)),
		}

		attributes = append(attributes, additionalAttributes...)

		isInScope, err := check(ctx, filter)
		if err != nil {
			f.meterEntitlementsFilterErrorsTotal.Add(ctx, 1, metric.WithAttributes(attributes...))
			return false, err
		}

		if !isInScope {
			f.meterEntitlementsFilterFilteredTotal.Add(ctx, 1, metric.WithAttributes(attributes...))
			return false, nil
		}
	}

	attributes := []attribute.KeyValue{
		attribute.String(metricLabelScope, string(scope)),
	}
	attributes = append(attributes, additionalAttributes...)

	f.meterEntitlementsFilterMatchesTotal.Add(ctx, 1, metric.WithAttributes(attributes...))

	return true, nil
}

func (f EntitlementFilters) RecordLastCalculation(ctx context.Context, req filters.RecordLastCalculationRequest) error {
	errs := []error{}

	for _, filter := range f.filters {
		if recorder, ok := filter.(filters.CalculationTimeRecorder); ok {
			if err := recorder.RecordLastCalculation(ctx, req); err != nil {
				errs = append(errs, fmt.Errorf("recording last calculation for filter %s: %w", filter.Name(), err))
			}
		}
	}

	return errors.Join(errs...)
}
