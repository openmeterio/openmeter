package balanceworker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker/negcache"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/samber/lo"
)

type handleEntitlementEventOptions struct {
	// Source is the source of the event, e.g. the "subject" field from the upstream cloudevents event causing the change
	source string

	// EventAt is the time of the event, e.g. the "time" field from the upstream cloudevents event causing the change
	eventAt time.Time

	// UseNegCache is true if the entitlement handler should use the negative cache (should be only enabled for events that are
	// coming from ingested events)
	useNegCache bool

	rawIngestedEvents []serializer.CloudEventsKafkaPayload
}

type handleOption func(*handleEntitlementEventOptions)

func WithSource(source string) handleOption {
	return func(o *handleEntitlementEventOptions) {
		o.source = source
	}
}

func WithEventAt(eventAt time.Time) handleOption {
	return func(o *handleEntitlementEventOptions) {
		o.eventAt = eventAt
	}
}

func WithNegCache(useNegCache bool) handleOption {
	return func(o *handleEntitlementEventOptions) {
		o.useNegCache = useNegCache
	}
}

func WithRawIngestedEvents(rawIngestedEvents []serializer.CloudEventsKafkaPayload) handleOption {
	return func(o *handleEntitlementEventOptions) {
		o.rawIngestedEvents = rawIngestedEvents
	}
}

func getOptions(opts ...handleOption) handleEntitlementEventOptions {
	options := handleEntitlementEventOptions{}

	for _, opt := range opts {
		opt(&options)
	}

	return options
}

func (w *Worker) handleEntitlementEvent(ctx context.Context, entitlementID NamespacedID, options ...handleOption) (marshaler.Event, error) {
	calculatedAt := time.Now()

	opts := getOptions(options...)

	if opts.eventAt.IsZero() {
		// TODO: set to error when the queue has been flushed
		w.opts.Logger.Warn("eventAt is zero, ignoring event", "entitlementID", entitlementID, "source", opts.source)
		return nil, nil
	}

	if entry, ok := w.highWatermarkCache.Get(entitlementID.ID); ok {
		if entry.HighWatermark.After(opts.eventAt) || entry.IsDeleted {
			if entry.IsDeleted {
				w.metricHighWatermarkCacheStats.Add(ctx, 1, metric.WithAttributes(metricAttributeHighWatermarkCacheHitDeleted))
			} else {
				w.metricHighWatermarkCacheStats.Add(ctx, 1, metric.WithAttributes(metricAttributeHighWatermarkCacheHit))
			}

			return nil, nil
		}

		w.metricHighWatermarkCacheStats.Add(ctx, 1, metric.WithAttributes(metricAttributeHighWatermarkCacheStale))
	} else {
		w.metricHighWatermarkCacheStats.Add(ctx, 1, metric.WithAttributes(metricAttributeHighWatermarkCacheMiss))
	}

	entitlements, err := w.entitlement.Entitlement.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
		Namespaces:     []string{entitlementID.Namespace},
		IDs:            []string{entitlementID.ID},
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement: %w", err)
	}

	if len(entitlements.Items) == 0 {
		return nil, fmt.Errorf("entitlement not found: %s", entitlementID.ID)
	}

	if len(entitlements.Items) > 1 {
		return nil, fmt.Errorf("multiple entitlements found: %s", entitlementID.ID)
	}

	entitlementEntity := entitlements.Items[0]
	return w.processEntitlementEntity(ctx, &entitlementEntity, calculatedAt, options...)
}

func (w *Worker) processEntitlementEntity(ctx context.Context, entitlementEntity *entitlement.Entitlement, calculatedAt time.Time, options ...handleOption) (marshaler.Event, error) {
	opts := getOptions(options...)

	if entitlementEntity == nil {
		return nil, fmt.Errorf("entitlement entity is nil")
	}

	if entitlementEntity.ActiveFrom != nil && entitlementEntity.ActiveFrom.After(calculatedAt) {
		// Not yet active entitlement we don't need to process it yet
		return nil, nil
	}

	if entitlementEntity.DeletedAt != nil ||
		(entitlementEntity.ActiveTo != nil && entitlementEntity.ActiveTo.Before(calculatedAt)) {
		// entitlement got deleted while processing changes => let's create a delete event so that we are not working

		snapshot, err := w.createDeletedSnapshotEvent(ctx,
			entitlement.EntitlementDeletedEvent{
				Entitlement: *entitlementEntity,
				Namespace: models.NamespaceID{
					ID: entitlementEntity.Namespace,
				},
			}, calculatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to create entitlement delete snapshot event: %w", err)
		}

		_ = w.highWatermarkCache.Add(entitlementEntity.ID, highWatermarkCacheEntry{
			HighWatermark: calculatedAt.Add(-defaultClockDrift),
			IsDeleted:     true,
		})

		return snapshot, nil
	}

	var err error
	var snapshot marshaler.Event
	if opts.useNegCache {
		snapshot, err = w.createSnapshotEventNegCache(ctx, entitlementEntity, calculatedAt, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create entitlement update snapshot event[negcache]: %w", err)
		}
	} else {
		snapshot, err = w.createSnapshotEvent(ctx, entitlementEntity, calculatedAt, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create entitlement update snapshot event: %w", err)
		}
	}

	_ = w.highWatermarkCache.Add(entitlementEntity.ID, highWatermarkCacheEntry{
		HighWatermark: calculatedAt.Add(-defaultClockDrift),
	})

	return snapshot, nil
}

type recalcAction string

const (
	recalcActionRecalculate recalcAction = "recalculate"
	recalcActionUseCache    recalcAction = "use-cache"
)

func (w *Worker) createSnapshotEventNegCache(ctx context.Context, entitlementEntity *entitlement.Entitlement, calculatedAt time.Time, opts handleEntitlementEventOptions) (marshaler.Event, error) {
	if len(opts.rawIngestedEvents) == 0 {
		return nil, fmt.Errorf("no raw ingested events provided")
	}

	feature, err := w.entitlement.Feature.GetFeature(ctx, entitlementEntity.Namespace, entitlementEntity.FeatureID, feature.IncludeArchivedFeatureTrue)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	if feature.MeterSlug == nil {
		return nil, fmt.Errorf("feature has no meter slug")
	}

	meterEntity, err := w.meter.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		IDOrSlug:  *feature.MeterSlug,
		Namespace: entitlementEntity.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get meter: %w", err)
	}

	target := negcache.TargetEntitlement{
		Entitlement: *entitlementEntity,
		Feature:     *feature,
		Meter:       meterEntity,
	}

	action := recalcActionUseCache

	ent, err := w.negCache.HandleEntitlementEvent(ctx, negcache.IngestEventInput{
		Target:        target,
		DedupedEvents: opts.rawIngestedEvents,
	})
	if err != nil {
		switch {
		case errors.Is(err, negcache.ErrUnsupportedMeterAggregation):
			// TODO: we need to fall back to the nonCached version as we don't need to touch redis for this
			action = recalcActionRecalculate
		case errors.Is(err, negcache.ErrConcurrentUpdate):
			// If there's a race condition that's due to rebalancing, so let's remove the entry
			// and once rebalancing is done, we will continue to cache
			if err := w.negCache.Remove(ctx, target); err != nil {
				w.opts.Logger.Error("failed to remove entitlement from negcache", "error", err)
			}
			action = recalcActionRecalculate
		case errors.Is(err, negcache.ErrEntryNotFound):
			action = recalcActionRecalculate
		default:
			return nil, fmt.Errorf("failed to handle entitlement event: %w", err)
		}
	}

	if ent == nil {
		action = recalcActionRecalculate
	}

	if ent != nil && action == recalcActionUseCache {
		thHit, err := w.hitsWatchedThresholds(ctx, ent)
		if err != nil {
			return nil, fmt.Errorf("failed to check if entitlement hits watched thresholds: %w", err)
		}

		if thHit {
			action = recalcActionRecalculate
		}
	}

	if action == recalcActionRecalculate {
		value, err := w.negCache.HandleRecalculation(ctx, target, func(ctx context.Context) (*snapshot.EntitlementValue, error) {
			res, err := w.entitlement.Entitlement.GetEntitlementValue(ctx, entitlementEntity.Namespace, entitlementEntity.SubjectKey, entitlementEntity.ID, calculatedAt)
			if err != nil {
				return nil, fmt.Errorf("failed to get entitlement value: %w", err)
			}

			value, err := entitlementdriver.MapEntitlementValueToAPI(res)
			if err != nil {
				return nil, fmt.Errorf("failed to map entitlement value: %w", err)
			}

			return lo.ToPtr((snapshot.EntitlementValue)(value)), nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to handle recalculation: %w", err)
		}

		if value == nil {
			return nil, fmt.Errorf("unexpected nil: entitlement value")
		}

		subject := models.Subject{
			Key: entitlementEntity.SubjectKey,
		}
		if w.opts.SubjectResolver != nil {
			subject, err = w.opts.SubjectResolver.GetSubjectByKey(ctx, entitlementEntity.Namespace, entitlementEntity.SubjectKey)
			if err != nil {
				return nil, fmt.Errorf("failed to get subject ID: %w", err)
			}
		}

		event := marshaler.WithSource(
			opts.source,
			snapshot.SnapshotEvent{
				Entitlement: *entitlementEntity,
				Namespace: models.NamespaceID{
					ID: entitlementEntity.Namespace,
				},
				Subject:   subject,
				Feature:   *feature,
				Operation: snapshot.ValueOperationUpdate,

				CalculatedAt: &calculatedAt,

				Value:              value,
				CurrentUsagePeriod: entitlementEntity.CurrentUsagePeriod,
			},
		)

		return event, nil
	}

	return nil, nil
}

func (w *Worker) createSnapshotEvent(ctx context.Context, entitlementEntity *entitlement.Entitlement, calculatedAt time.Time, opts handleEntitlementEventOptions) (marshaler.Event, error) {
	feature, err := w.entitlement.Feature.GetFeature(ctx, entitlementEntity.Namespace, entitlementEntity.FeatureID, feature.IncludeArchivedFeatureTrue)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	calculationStart := time.Now()

	value, err := w.entitlement.Entitlement.GetEntitlementValue(ctx, entitlementEntity.Namespace, entitlementEntity.SubjectKey, entitlementEntity.ID, calculatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement value: %w", err)
	}

	if value == nil {
		return nil, fmt.Errorf("unexpected nil: entitlement value")
	}

	w.metricRecalculationTime.Record(ctx, time.Since(calculationStart).Milliseconds(), metric.WithAttributes(
		attribute.String(metricAttributeKeyEntitltementType, string(entitlementEntity.EntitlementType)),
	))

	mappedValues, err := entitlementdriver.MapEntitlementValueToAPI(value)
	if err != nil {
		return nil, fmt.Errorf("failed to map entitlement value: %w", err)
	}

	subject := models.Subject{
		Key: entitlementEntity.SubjectKey,
	}
	if w.opts.SubjectResolver != nil {
		subject, err = w.opts.SubjectResolver.GetSubjectByKey(ctx, entitlementEntity.Namespace, entitlementEntity.SubjectKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get subject ID: %w", err)
		}
	}

	event := marshaler.WithSource(
		opts.source,
		snapshot.SnapshotEvent{
			Entitlement: *entitlementEntity,
			Namespace: models.NamespaceID{
				ID: entitlementEntity.Namespace,
			},
			Subject:   subject,
			Feature:   *feature,
			Operation: snapshot.ValueOperationUpdate,

			CalculatedAt: &calculatedAt,

			Value:              convert.ToPointer((snapshot.EntitlementValue)(mappedValues)),
			CurrentUsagePeriod: entitlementEntity.CurrentUsagePeriod,
		},
	)

	return event, nil
}

func (w *Worker) createDeletedSnapshotEvent(ctx context.Context, delEvent entitlement.EntitlementDeletedEvent, calculationTime time.Time) (marshaler.Event, error) {
	namespace := delEvent.Namespace.ID

	feature, err := w.entitlement.Feature.GetFeature(ctx, namespace, delEvent.FeatureID, feature.IncludeArchivedFeatureTrue)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	subject := models.Subject{
		Key: delEvent.SubjectKey,
	}

	if w.opts.SubjectResolver != nil {
		subject, err = w.opts.SubjectResolver.GetSubjectByKey(ctx, namespace, delEvent.SubjectKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get subject: %w", err)
		}
	}

	event := marshaler.WithSource(
		metadata.ComposeResourcePath(namespace, metadata.EntityEntitlement, delEvent.ID),
		snapshot.SnapshotEvent{
			Entitlement: delEvent.Entitlement,
			Namespace: models.NamespaceID{
				ID: namespace,
			},
			Subject:   subject,
			Feature:   *feature,
			Operation: snapshot.ValueOperationDelete,

			CalculatedAt: convert.ToPointer(calculationTime),

			CurrentUsagePeriod: delEvent.CurrentUsagePeriod,
		},
	)

	return event, nil
}
