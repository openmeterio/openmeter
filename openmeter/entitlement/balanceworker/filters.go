package balanceworker

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
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
}

func (c EntitlementFiltersConfig) Validate() error {
	if c.NotificationService == nil {
		return fmt.Errorf("notification service is required")
	}

	if c.MetricMeter == nil {
		return fmt.Errorf("metric meter is required")
	}

	return nil
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

	return EntitlementFilters{
		filters: []filters.NamedFilter{notificationFilter},
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

func (f EntitlementFilters) IsEntitlementInScope(ctx context.Context, entitlement entitlement.Entitlement) (bool, error) {
	f.meterEntitlementsFilterRequestsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String(metricLabelScope, string(FilterScopeEntitlement)),
			attribute.String(metricLabelNamespace, entitlement.Namespace),
		),
	)

	return f.executeFilters(ctx,
		func(ctx context.Context, filter filters.Filter) (bool, error) {
			return filter.IsEntitlementInScope(ctx, entitlement)
		},
		FilterScopeEntitlement,
		attribute.String(metricLabelNamespace, entitlement.Namespace),
	)
}

func (f EntitlementFilters) executeFilters(ctx context.Context, check func(ctx context.Context, filter filters.Filter) (bool, error), scope FilterScope, additionalAttributes ...attribute.KeyValue) (bool, error) {
	for _, filter := range f.filters {
		attributes := []attribute.KeyValue{
			attribute.String(metricLabelFilterName, filter.Name()),
			attribute.String(metricLabelScope, string(scope)),
		}

		attributes = append(attributes, additionalAttributes...)

		nsInScope, err := check(ctx, filter)
		if err != nil {
			f.meterEntitlementsFilterErrorsTotal.Add(ctx, 1, metric.WithAttributes(attributes...))
			return false, err
		}

		if nsInScope {
			f.meterEntitlementsFilterMatchesTotal.Add(ctx, 1, metric.WithAttributes(attributes...))
			return true, nil
		}
	}

	attributes := []attribute.KeyValue{
		attribute.String(metricLabelScope, string(scope)),
	}
	attributes = append(attributes, additionalAttributes...)

	f.meterEntitlementsFilterFilteredTotal.Add(ctx, 1, metric.WithAttributes(attributes...))

	return false, nil
}
