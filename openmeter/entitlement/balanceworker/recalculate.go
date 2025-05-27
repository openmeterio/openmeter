package balanceworker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	// DefaultIncludeDeletedDuration is the default duration for which deleted entitlements are included in recalculation.
	// This ensures that the recent deleted snapshot events are also resent.
	DefaultIncludeDeletedDuration = 24 * time.Hour

	defaultLRUCacheSize = 10_000
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
	Entitlement        *registry.Entitlement
	SubjectResolver    SubjectResolver
	EventBus           eventbus.Publisher
	MetricMeter        metric.Meter
	StreamingConnector streaming.Connector

	OnlyRecalculateEntitlementsWithChanges bool
	Logger                                 *slog.Logger
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

	if o.StreamingConnector == nil {
		return errors.New("missing streaming connector")
	}

	if o.Logger == nil {
		return errors.New("missing logger")
	}

	return nil
}

type Recalculator struct {
	opts RecalculatorOptions

	featureCache *lru.Cache[string, feature.Feature]
	subjectCache *lru.Cache[string, models.Subject]

	metricRecalculationTime                 metric.Int64Histogram
	metricRecalculationJobRecalculationTime metric.Int64Histogram
}

func NewRecalculator(opts RecalculatorOptions) (*Recalculator, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	featureCache, err := lru.New[string, feature.Feature](defaultLRUCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create feature cache: %w", err)
	}

	subjectCache, err := lru.New[string, models.Subject](defaultLRUCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create subject ID cache: %w", err)
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

	return &Recalculator{
		opts:                                    opts,
		featureCache:                            featureCache,
		subjectCache:                            subjectCache,
		metricRecalculationTime:                 metricRecalculationTime,
		metricRecalculationJobRecalculationTime: metricRecalculationJobRecalculationTime,
	}, nil
}

func (r *Recalculator) Recalculate(ctx context.Context, ns string) error {
	if ns == "" {
		return errors.New("namespace is required")
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

	if r.opts.OnlyRecalculateEntitlementsWithChanges {
		var err error
		// Recalculation avoidance
		affectedEntitlements, err = r.entitlementsWithChanges(ctx, ns, affectedEntitlements)
		if err != nil {
			return err
		}
	}

	return r.processEntitlements(ctx, affectedEntitlements)
}

func (r *Recalculator) entitlementsWithChanges(ctx context.Context, namespace string, entitlements []entitlement.Entitlement) ([]entitlement.Entitlement, error) {
	now := time.Now()
	eventsByNamespace, err := r.opts.StreamingConnector.CountEvents(ctx, namespace, streaming.CountEventsParams{
		From: now.Add(-DefaultIncludeDeletedDuration),
		To:   &now,
	})
	if err != nil {
		return nil, err
	}

	subjectsWithIngestedEvents, _ := slicesx.UniqueGroupBy(eventsByNamespace, func(e streaming.CountEventRow) string {
		return e.Subject
	})

	nrSubjectsWithIngestedEvents := 0
	nrDeleted := 0
	nrActivatingDeactivating := 0
	nrGrantActive := 0
	nrReseting := 0

	out := make([]entitlement.Entitlement, 0, len(entitlements))
	for _, ent := range entitlements {
		// If he subject has any events, let's recalculate
		if _, ok := subjectsWithIngestedEvents[ent.SubjectKey]; ok {
			out = append(out, ent)
			nrSubjectsWithIngestedEvents++
			break
		}

		// If the entitlement is deleted, let's recalculate (cheap, and ensures consistency)
		if ent.DeletedAt != nil {
			out = append(out, ent)
			nrDeleted++
			break
		}

		checkPeriod := timeutil.ClosedPeriod{
			From: now.Add(-DefaultIncludeDeletedDuration),
			To:   now,
		}

		/// If the entilement has been actived/deactivated recently, let's recalculate
		if checkPeriod.Contains(ent.ActiveFromTime()) {
			out = append(out, ent)
			nrActivatingDeactivating++
			break
		}

		if ent.ActiveToTime() != nil && checkPeriod.Contains(*ent.ActiveToTime()) {
			out = append(out, ent)
			nrActivatingDeactivating++
			break
		}

		// If the entitlement is not active, let's skip it
		if !ent.IsActive(now) {
			continue
		}

		// If the entitlement has been reset recently and active, let's recalculate
		if ent.CurrentUsagePeriod != nil &&
			(checkPeriod.Contains(ent.CurrentUsagePeriod.From) || checkPeriod.Contains(ent.CurrentUsagePeriod.To)) {
			out = append(out, ent)
			nrReseting++
			break
		}

		// If the entitlement has recent grants, let's recalculate
		canChange, err := r.hasEntitlementGrantInducedChanges(ctx, now, ent)
		if err != nil {
			return nil, err
		}

		if canChange {
			out = append(out, ent)
			nrActivatingDeactivating++
			break
		}
	}

	r.opts.Logger.Info("recalculation build avoidance stats for namespace",
		slog.String("namespace", namespace),
		slog.Int("stat.subjectsWithIngestedEvents", nrSubjectsWithIngestedEvents),
		slog.Int("stat.deleted", nrDeleted),
		slog.Int("stat.activatingDeactivating", nrActivatingDeactivating),
		slog.Int("stat.grantActive", nrGrantActive),
		slog.Int("stat.reseting", nrReseting),
		slog.Int("stat.totalEntitlements", len(entitlements)),
	)

	return out, nil
}

func (r *Recalculator) hasEntitlementGrantInducedChanges(ctx context.Context, at time.Time, ent entitlement.Entitlement) (bool, error) {
	grants, err := r.opts.Entitlement.MeteredEntitlement.ListEntitlementGrants(ctx, ent.Namespace, ent.SubjectKey, ent.ID)
	if err != nil {
		return false, err
	}

	// Let's check if we have any grants that might affect the balance
	checkPeriod := timeutil.ClosedPeriod{
		From: at.Add(-DefaultIncludeDeletedDuration),
		To:   at,
	}

	for _, grant := range grants {
		effectivePeriod := grant.GetEffectivePeriod()

		if effectivePeriod.Overlaps(checkPeriod) {
			continue
		}

		// non-recurring grant => let's check if it got activted or expired
		if grant.Recurrence == nil {
			if checkPeriod.Contains(effectivePeriod.From) || checkPeriod.Contains(effectivePeriod.To) {
				return true, nil
			}
		} else {
			next, err := grant.Recurrence.GetPeriodAt(at)
			if err != nil {
				return false, err
			}

			if next.Overlaps(checkPeriod) {
				return true, nil
			}
		}
	}

	return false, nil
}

func (r *Recalculator) processEntitlements(ctx context.Context, entitlements []entitlement.Entitlement) error {
	var errs error
	for _, ent := range entitlements {
		start := time.Now()

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
	subject, err := r.getSubjectByKey(ctx, ent.Namespace, ent.SubjectKey)
	if err != nil {
		return err
	}

	feature, err := r.getFeature(ctx, ent.Namespace, ent.FeatureID)
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
	subject, err := r.getSubjectByKey(ctx, ent.Namespace, ent.SubjectKey)
	if err != nil {
		return err
	}

	feature, err := r.getFeature(ctx, ent.Namespace, ent.FeatureID)
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

func (r *Recalculator) getSubjectByKey(ctx context.Context, ns, key string) (models.Subject, error) {
	if r.opts.SubjectResolver == nil {
		return models.Subject{
			Key: key,
		}, nil
	}

	if id, ok := r.subjectCache.Get(key); ok {
		return id, nil
	}

	id, err := r.opts.SubjectResolver.GetSubjectByKey(ctx, ns, key)
	if err != nil {
		return models.Subject{
			Key: key,
		}, err
	}

	r.subjectCache.Add(key, id)
	return id, nil
}

func (r *Recalculator) getFeature(ctx context.Context, ns, id string) (feature.Feature, error) {
	if feat, ok := r.featureCache.Get(id); ok {
		return feat, nil
	}

	feat, err := r.opts.Entitlement.Feature.GetFeature(ctx, ns, id, feature.IncludeArchivedFeatureTrue)
	if err != nil {
		return feature.Feature{}, err
	}

	r.featureCache.Add(id, *feat)
	return *feat, nil
}
