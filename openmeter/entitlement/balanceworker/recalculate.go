package balanceworker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker/filters"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/lrux"
	pkgmodels "github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

const (
	// DefaultIncludeDeletedDuration is the default duration for which deleted entitlements are included in recalculation.
	// This ensures that the recent deleted snapshot events are also resent.
	DefaultIncludeDeletedDuration = 24 * time.Hour

	defaultLRUCacheSize = 10_000
	defaultCacheTTL     = 15 * time.Second
	defaultPageSize     = 20_000

	metricNameRecalculationTime               = "balance_worker.entitlement_recalculation_time_ms"
	metricNameRecalculationJobCalculationTime = "balance_worker.entitlement_recalculation_job_calculation_time_ms"

	metricAttributeKeyEntitltementType = "entitlement_type"
)

type RecalculatorOptions struct {
	Entitlement *registry.Entitlement
	Subject     subject.Service
	Customer    customer.Service
	EventBus    eventbus.Publisher
	MetricMeter metric.Meter

	NotificationService notification.Service
	Logger              *slog.Logger

	HighWatermarkCacheSize int
}

func (o RecalculatorOptions) Validate() error {
	var errs []error

	if o.Entitlement == nil {
		errs = append(errs, errors.New("entitlements service is required"))
	}

	if o.Subject == nil {
		errs = append(errs, errors.New("subject service is required"))
	}

	if o.Customer == nil {
		errs = append(errs, errors.New("customer service is required"))
	}

	if o.EventBus == nil {
		errs = append(errs, errors.New("missing event bus"))
	}

	if o.MetricMeter == nil {
		errs = append(errs, errors.New("missing metric meter"))
	}

	if o.NotificationService == nil {
		errs = append(errs, errors.New("missing notification service"))
	}

	if o.Customer == nil {
		errs = append(errs, errors.New("missing customer service"))
	}

	if o.Subject == nil {
		errs = append(errs, errors.New("missing subject service"))
	}

	if o.HighWatermarkCacheSize <= 0 {
		errs = append(errs, fmt.Errorf("high watermark cache size must be positive"))
	}

	if o.Logger == nil {
		errs = append(errs, errors.New("missing logger"))
	}

	return errors.Join(errs...)
}

type Recalculator struct {
	opts RecalculatorOptions

	featureCache         *lrux.CacheWithItemTTL[pkgmodels.NamespacedID, feature.Feature]
	customerSubjectCache *lrux.CacheWithItemTTL[pkgmodels.NamespacedID, customerAndSubject]

	entitlementFilters *EntitlementFilters

	metricRecalculationTime                 metric.Int64Histogram
	metricRecalculationJobRecalculationTime metric.Int64Histogram
}

func NewRecalculator(opts RecalculatorOptions) (*Recalculator, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	metricRecalculationTime, err := opts.MetricMeter.Int64Histogram(
		metricNameRecalculationTime,
		metric.WithDescription("Entitlement recalculation time"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create recalculation time histogram: %w", err)
	}

	metricRecalculationJobRecalculationTime, err := opts.MetricMeter.Int64Histogram(
		metricNameRecalculationJobCalculationTime,
		metric.WithDescription("Time takes to recalculate the entitlements including the necessary data fetches"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create recalculation time histogram: %w", err)
	}

	entitlementFilters, err := NewEntitlementFilters(EntitlementFiltersConfig{
		NotificationService:    opts.NotificationService,
		MetricMeter:            opts.MetricMeter,
		HighWatermarkCacheSize: opts.HighWatermarkCacheSize,
		Logger:                 opts.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create entitlement filters: %w", err)
	}

	res := &Recalculator{
		opts:               opts,
		entitlementFilters: entitlementFilters,

		metricRecalculationTime:                 metricRecalculationTime,
		metricRecalculationJobRecalculationTime: metricRecalculationJobRecalculationTime,
	}

	res.featureCache, err = lrux.NewCacheWithItemTTL(defaultLRUCacheSize, res.getFeature, lrux.WithTTL(defaultCacheTTL))
	if err != nil {
		return nil, fmt.Errorf("failed to create feature cache: %w", err)
	}

	res.customerSubjectCache, err = lrux.NewCacheWithItemTTL(defaultLRUCacheSize, res.getCustomerAndSubject, lrux.WithTTL(defaultCacheTTL))
	if err != nil {
		return nil, fmt.Errorf("failed to create subject ID cache: %w", err)
	}

	return res, nil
}

func (r *Recalculator) GetEntitlementFilters() *EntitlementFilters {
	return r.entitlementFilters
}

func (r *Recalculator) Recalculate(ctx context.Context, ns string, recalculationStartedAt time.Time) error {
	affectedEntitlements, err := r.ListInScopeEntitlements(ctx, ns)
	if err != nil {
		return fmt.Errorf("failed to list in scope entitlements: %w", err)
	}

	return r.ProcessEntitlements(ctx, affectedEntitlements, recalculationStartedAt)
}

func (r *Recalculator) ListInScopeEntitlements(ctx context.Context, ns string) ([]entitlement.Entitlement, error) {
	if ns == "" {
		return nil, errors.New("namespace is required")
	}

	inScope, err := r.entitlementFilters.IsNamespaceInScope(ctx, ns)
	if err != nil {
		return nil, fmt.Errorf("failed to check if namespace is in scope: %w", err)
	}
	if !inScope {
		return nil, nil
	}

	// Note: this is to support namesapces with more than 64k entitlements, as the subqueries
	// to expand the edges uses IN statements in ent. We should rather fix ent to actually chunk
	// the subqueries.
	affectedEntitlements := []entitlement.Entitlement{}

	page := 1

	for {
		affectedEntitlementsPage, err := r.opts.Entitlement.EntitlementRepo.ListEntitlements(
			ctx,
			entitlement.ListEntitlementsParams{
				Namespaces:          []string{ns},
				IncludeDeleted:      true,
				IncludeDeletedAfter: time.Now().Add(-DefaultIncludeDeletedDuration),
				Page: pagination.Page{
					PageNumber: page,
					PageSize:   defaultPageSize,
				},
			})
		if err != nil {
			return nil, err
		}

		if len(affectedEntitlementsPage.Items) == 0 {
			break
		}

		affectedEntitlements = append(affectedEntitlements, affectedEntitlementsPage.Items...)

		if len(affectedEntitlements) >= affectedEntitlementsPage.TotalCount {
			break
		}

		page++
	}

	return affectedEntitlements, nil
}

func (r *Recalculator) ProcessEntitlements(ctx context.Context, entitlements []entitlement.Entitlement, recalculationStartedAt time.Time) error {
	var errs error
	for _, ent := range entitlements {
		start := time.Now()

		inScope, err := r.entitlementFilters.IsEntitlementInScope(ctx, filters.EntitlementFilterRequest{
			Entitlement: ent,
			EventAt:     recalculationStartedAt,
			Operation:   snapshot.ValueOperationUpdate,
		})
		if err != nil {
			return fmt.Errorf("failed to check if entitlement is in scope: %w", err)
		}
		if !inScope {
			continue
		}

		res, err := r.sendEntitlementEvent(ctx, ent)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("error sending event for entitlement [id=%s]: %w", ent.ID, err))
			continue
		}

		r.metricRecalculationJobRecalculationTime.Record(ctx,
			time.Since(start).Milliseconds(),
			metric.WithAttributes(
				attribute.String(metricAttributeKeyEntitltementType, string(ent.EntitlementType)),
			))

		err = r.entitlementFilters.RecordLastCalculation(ctx, filters.RecordLastCalculationRequest{
			Entitlement:  ent,
			CalculatedAt: res.CalculatedAt,
		})
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to record last calculation for entitlement [id=%s]: %w", ent.ID, err))
			continue
		}
	}

	return errs
}

type sendEntitlementEventResult struct {
	CalculatedAt time.Time
}

func (r *Recalculator) sendEntitlementEvent(ctx context.Context, ent entitlement.Entitlement) (sendEntitlementEventResult, error) {
	if ent.DeletedAt != nil || (ent.ActiveTo != nil && time.Now().After(*ent.ActiveTo)) {
		return r.sendEntitlementDeletedEvent(ctx, ent)
	}

	return r.sendEntitlementUpdatedEvent(ctx, ent)
}

func (r *Recalculator) sendEntitlementDeletedEvent(ctx context.Context, ent entitlement.Entitlement) (sendEntitlementEventResult, error) {
	empty := sendEntitlementEventResult{}

	custSubj, err := r.customerSubjectCache.Get(ctx, pkgmodels.NamespacedID{
		Namespace: ent.Namespace,
		ID:        ent.CustomerID,
	})
	if err != nil {
		return empty, err
	}

	feat, err := r.featureCache.Get(ctx, pkgmodels.NamespacedID{
		Namespace: ent.Namespace,
		ID:        ent.FeatureID,
	})
	if err != nil {
		return empty, err
	}

	calculatedAt := time.Now()

	event := marshaler.WithSource(
		metadata.ComposeResourcePath(ent.Namespace, metadata.EntityEntitlement, ent.ID),
		snapshot.SnapshotEvent{
			Entitlement: ent,
			Namespace: models.NamespaceID{
				ID: ent.Namespace,
			},
			Subject:   lo.FromPtr(custSubj.Subject), // Subject is deprecated, use empty if nil
			Customer:  custSubj.Customer,
			Value:     nil, // explicit nil for deleted events
			Feature:   feat,
			Operation: snapshot.ValueOperationDelete,

			CalculatedAt: convert.ToPointer(calculatedAt),

			CurrentUsagePeriod: ent.CurrentUsagePeriod,
		},
	)

	return sendEntitlementEventResult{
		CalculatedAt: calculatedAt,
	}, r.opts.EventBus.Publish(ctx, event)
}

func (r *Recalculator) sendEntitlementUpdatedEvent(ctx context.Context, ent entitlement.Entitlement) (sendEntitlementEventResult, error) {
	empty := sendEntitlementEventResult{}

	feat, err := r.featureCache.Get(ctx, pkgmodels.NamespacedID{
		Namespace: ent.Namespace,
		ID:        ent.FeatureID,
	})
	if err != nil {
		return empty, err
	}

	custSubj, err := r.customerSubjectCache.Get(ctx, pkgmodels.NamespacedID{
		Namespace: ent.Namespace,
		ID:        ent.CustomerID,
	})
	if err != nil {
		return empty, err
	}

	calculatedAt := time.Now()

	value, err := r.opts.Entitlement.Entitlement.GetEntitlementValue(ctx, ent.Namespace, ent.CustomerID, ent.ID, calculatedAt)
	if err != nil {
		return empty, fmt.Errorf("failed to get entitlement value: %w", err)
	}

	r.metricRecalculationTime.Record(ctx,
		time.Since(calculatedAt).Milliseconds(),
		metric.WithAttributes(
			attribute.String(metricAttributeKeyEntitltementType, string(ent.EntitlementType)),
		))

	mappedValues, err := entitlementdriver.MapEntitlementValueToAPI(value)
	if err != nil {
		return empty, fmt.Errorf("failed to map entitlement value: %w", err)
	}

	event := marshaler.WithSource(
		metadata.ComposeResourcePath(ent.Namespace, metadata.EntityEntitlement, ent.ID),
		snapshot.SnapshotEvent{
			Entitlement: ent,
			Namespace: models.NamespaceID{
				ID: ent.Namespace,
			},
			Subject:   lo.FromPtr(custSubj.Subject), // Subject is deprecated, use empty if nil
			Customer:  custSubj.Customer,
			Feature:   feat,
			Operation: snapshot.ValueOperationUpdate,

			CalculatedAt: &calculatedAt,

			Value:              convert.ToPointer((snapshot.EntitlementValue)(mappedValues)),
			CurrentUsagePeriod: ent.CurrentUsagePeriod,
		},
	)

	return sendEntitlementEventResult{
		CalculatedAt: calculatedAt,
	}, r.opts.EventBus.Publish(ctx, event)
}

func (r *Recalculator) getCustomerAndSubject(ctx context.Context, namespacedId pkgmodels.NamespacedID) (customerAndSubject, error) {
	cust, subj, err := resolveCustomerAndSubject(ctx, r.opts.Customer, r.opts.Subject, namespacedId.Namespace, namespacedId.ID)
	if err != nil {
		return customerAndSubject{}, err
	}

	return customerAndSubject{
		Customer: cust,
		Subject:  subj,
	}, nil
}

func (r *Recalculator) getFeature(ctx context.Context, featureID pkgmodels.NamespacedID) (feature.Feature, error) {
	feat, err := r.opts.Entitlement.Feature.GetFeature(ctx, featureID.Namespace, featureID.ID, feature.IncludeArchivedFeatureTrue)
	if err != nil {
		return feature.Feature{}, err
	}

	return *feat, nil
}

// customerAndSubject is a helper struct to be used in the cache
type customerAndSubject struct {
	Customer customer.Customer
	// Subject is optional - may be nil if customer has no usage attribution with subject keys
	Subject *subject.Subject
}
