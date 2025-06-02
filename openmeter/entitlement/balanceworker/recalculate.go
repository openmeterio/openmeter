package balanceworker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/registry"
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
	metricNameHighWatermarkCacheStats         = "balance_worker.high_watermark_cache_stats"

	metricAttributeKeyEntitltementType = "entitlement_type"
)

var (
	metricAttributeHighWatermarkCacheHit        = attribute.String("op", "hit")
	metricAttributeHighWatermarkCacheHitDeleted = attribute.String("op", "hit_deleted")
	metricAttributeHighWatermarkCacheMiss       = attribute.String("op", "miss")
	metricAttributeHighWatermarkCacheStale      = attribute.String("op", "stale")
)

type RecalculatorOptions struct {
	Entitlement     *registry.Entitlement
	SubjectResolver SubjectResolver
	EventBus        eventbus.Publisher
	MetricMeter     metric.Meter

	NotificationService notification.Service
}

func (o RecalculatorOptions) Validate() error {
	if o.Entitlement == nil {
		return errors.New("missing entitlement registry")
	}

	if o.EventBus == nil {
		return errors.New("missing event bus")
	}

	if o.MetricMeter == nil {
		return errors.New("missing metric meter")
	}

	if o.NotificationService == nil {
		return errors.New("missing notification service")
	}

	return nil
}

type Recalculator struct {
	opts RecalculatorOptions

	featureCache *lrux.CacheWithItemTTL[pkgmodels.NamespacedID, feature.Feature]
	subjectCache *lrux.CacheWithItemTTL[pkgmodels.NamespacedKey, models.Subject]

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
		NotificationService: opts.NotificationService,
		MetricMeter:         opts.MetricMeter,
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

	res.subjectCache, err = lrux.NewCacheWithItemTTL(defaultLRUCacheSize, res.getSubjectByKey, lrux.WithTTL(defaultCacheTTL))
	if err != nil {
		return nil, fmt.Errorf("failed to create subject ID cache: %w", err)
	}

	return res, nil
}

func (r *Recalculator) GetEntitlementFilters() *EntitlementFilters {
	return r.entitlementFilters
}

func (r *Recalculator) Recalculate(ctx context.Context, ns string) error {
	if ns == "" {
		return errors.New("namespace is required")
	}

	inScope, err := r.entitlementFilters.IsNamespaceInScope(ctx, ns)
	if err != nil {
		return fmt.Errorf("failed to check if namespace is in scope: %w", err)
	}
	if !inScope {
		return nil
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
			return err
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

	return r.processEntitlements(ctx, affectedEntitlements)
}

func (r *Recalculator) processEntitlements(ctx context.Context, entitlements []entitlement.Entitlement) error {
	var errs error
	for _, ent := range entitlements {
		start := time.Now()

		inScope, err := r.entitlementFilters.IsEntitlementInScope(ctx, ent)
		if err != nil {
			return fmt.Errorf("failed to check if entitlement is in scope: %w", err)
		}
		if !inScope {
			continue
		}

		if err := r.sendEntitlementEvent(ctx, ent); err != nil {
			errs = errors.Join(errs, fmt.Errorf("error sending event for entitlement [id=%s]: %w", ent.ID, err))
		}

		r.metricRecalculationJobRecalculationTime.Record(ctx,
			time.Since(start).Milliseconds(),
			metric.WithAttributes(
				attribute.String(metricAttributeKeyEntitltementType, string(ent.EntitlementType)),
			))
	}

	return errs
}

func (r *Recalculator) sendEntitlementEvent(ctx context.Context, ent entitlement.Entitlement) error {
	if ent.DeletedAt != nil || (ent.ActiveTo != nil && time.Now().After(*ent.ActiveTo)) {
		return r.sendEntitlementDeletedEvent(ctx, ent)
	}

	return r.sendEntitlementUpdatedEvent(ctx, ent)
}

func (r *Recalculator) sendEntitlementDeletedEvent(ctx context.Context, ent entitlement.Entitlement) error {
	subject, err := r.subjectCache.Get(ctx, pkgmodels.NamespacedKey{
		Namespace: ent.Namespace,
		Key:       ent.SubjectKey,
	})
	if err != nil {
		return err
	}

	feature, err := r.featureCache.Get(ctx, pkgmodels.NamespacedID{
		Namespace: ent.Namespace,
		ID:        ent.FeatureID,
	})
	if err != nil {
		return err
	}

	event := marshaler.WithSource(
		metadata.ComposeResourcePath(ent.Namespace, metadata.EntityEntitlement, ent.ID),
		snapshot.SnapshotEvent{
			Entitlement: ent,
			Namespace: models.NamespaceID{
				ID: ent.Namespace,
			},
			Subject:   subject,
			Feature:   feature,
			Operation: snapshot.ValueOperationDelete,

			CalculatedAt: convert.ToPointer(time.Now().Add(-defaultClockDrift)),

			CurrentUsagePeriod: ent.CurrentUsagePeriod,
		},
	)

	return r.opts.EventBus.Publish(ctx, event)
}

func (r *Recalculator) sendEntitlementUpdatedEvent(ctx context.Context, ent entitlement.Entitlement) error {
	subject, err := r.subjectCache.Get(ctx, pkgmodels.NamespacedKey{
		Namespace: ent.Namespace,
		Key:       ent.SubjectKey,
	})
	if err != nil {
		return err
	}

	feature, err := r.featureCache.Get(ctx, pkgmodels.NamespacedID{
		Namespace: ent.Namespace,
		ID:        ent.FeatureID,
	})
	if err != nil {
		return err
	}

	calculatedAt := time.Now()

	value, err := r.opts.Entitlement.Entitlement.GetEntitlementValue(ctx, ent.Namespace, ent.SubjectKey, ent.ID, calculatedAt)
	if err != nil {
		return fmt.Errorf("failed to get entitlement value: %w", err)
	}

	r.metricRecalculationTime.Record(ctx,
		time.Since(calculatedAt).Milliseconds(),
		metric.WithAttributes(
			attribute.String(metricAttributeKeyEntitltementType, string(ent.EntitlementType)),
		))

	mappedValues, err := entitlementdriver.MapEntitlementValueToAPI(value)
	if err != nil {
		return fmt.Errorf("failed to map entitlement value: %w", err)
	}

	event := marshaler.WithSource(
		metadata.ComposeResourcePath(ent.Namespace, metadata.EntityEntitlement, ent.ID),
		snapshot.SnapshotEvent{
			Entitlement: ent,
			Namespace: models.NamespaceID{
				ID: ent.Namespace,
			},
			Subject:   subject,
			Feature:   feature,
			Operation: snapshot.ValueOperationUpdate,

			CalculatedAt: &calculatedAt,

			Value:              convert.ToPointer((snapshot.EntitlementValue)(mappedValues)),
			CurrentUsagePeriod: ent.CurrentUsagePeriod,
		},
	)

	return r.opts.EventBus.Publish(ctx, event)
}

func (r *Recalculator) getSubjectByKey(ctx context.Context, namespacedKey pkgmodels.NamespacedKey) (models.Subject, error) {
	if r.opts.SubjectResolver == nil {
		return models.Subject{
			Key: namespacedKey.Key,
		}, nil
	}

	subject, err := r.opts.SubjectResolver.GetSubjectByKey(ctx, namespacedKey.Namespace, namespacedKey.Key)
	if err != nil {
		return models.Subject{
			Key: namespacedKey.Key,
		}, err
	}

	return subject, nil
}

func (r *Recalculator) getFeature(ctx context.Context, featureID pkgmodels.NamespacedID) (feature.Feature, error) {
	feat, err := r.opts.Entitlement.Feature.GetFeature(ctx, featureID.Namespace, featureID.ID, feature.IncludeArchivedFeatureTrue)
	if err != nil {
		return feature.Feature{}, err
	}

	return *feat, nil
}
