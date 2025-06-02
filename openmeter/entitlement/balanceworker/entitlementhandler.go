package balanceworker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/convert"
)

type handleEntitlementEventOptions struct {
	// Source is the source of the event, e.g. the "subject" field from the upstream cloudevents event causing the change
	source string

	// EventAt is the time of the event, e.g. the "time" field from the upstream cloudevents event causing the change
	eventAt time.Time

	// SourceOperation is the operation that caused the entitlement change (if empty update is assumed)
	sourceOperation *snapshot.ValueOperationType

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

func WithSourceOperation(sourceOperation snapshot.ValueOperationType) handleOption {
	return func(o *handleEntitlementEventOptions) {
		o.sourceOperation = &sourceOperation
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

type highWatermarkCacheAction string

const (
	recalculateAction highWatermarkCacheAction = "recalculate"
	skipEventAction   highWatermarkCacheAction = "skipEvent"
)

func (w *Worker) checkHighWatermarkCache(ctx context.Context, entitlementID NamespacedID, opts handleEntitlementEventOptions) highWatermarkCacheAction {
	// Always emit reset events
	// TODO[later]: Only calculate if there's need for the explicit reset event
	if lo.FromPtr(opts.sourceOperation) == snapshot.ValueOperationReset {
		return recalculateAction
	}

	if entry, ok := w.highWatermarkCache.Get(entitlementID.ID); ok {
		if entry.HighWatermark.After(opts.eventAt) || entry.IsDeleted {
			if entry.IsDeleted {
				w.metricHighWatermarkCacheStats.Add(ctx, 1, metric.WithAttributes(metricAttributeHighWatermarkCacheHitDeleted))
			} else {
				w.metricHighWatermarkCacheStats.Add(ctx, 1, metric.WithAttributes(metricAttributeHighWatermarkCacheHit))
			}

			return skipEventAction
		}

		w.metricHighWatermarkCacheStats.Add(ctx, 1, metric.WithAttributes(metricAttributeHighWatermarkCacheStale))
	} else {
		w.metricHighWatermarkCacheStats.Add(ctx, 1, metric.WithAttributes(metricAttributeHighWatermarkCacheMiss))
	}

	return recalculateAction
}

func (w *Worker) handleEntitlementEvent(ctx context.Context, entitlementID NamespacedID, options ...handleOption) (marshaler.Event, error) {
	calculatedAt := time.Now()

	opts := getOptions(options...)

	if opts.eventAt.IsZero() {
		// TODO: set to error when the queue has been flushed
		w.opts.Logger.Warn("eventAt is zero, ignoring event", "entitlementID", entitlementID, "source", opts.source)
		return nil, nil
	}

	inScope, err := w.filters.IsNamespaceInScope(ctx, entitlementID.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to check if entitlement is in scope: %w", err)
	}
	if !inScope {
		return nil, nil
	}

	action := w.checkHighWatermarkCache(ctx, entitlementID, opts)

	if action == skipEventAction {
		return nil, nil
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

	inScope, err = w.filters.IsEntitlementInScope(ctx, entitlementEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to check if entitlement is in scope: %w", err)
	}
	if !inScope {
		return nil, nil
	}
	return w.processEntitlementEntity(ctx, &entitlementEntity, calculatedAt, options...)
}

func (w *Worker) processEntitlementEntity(ctx context.Context, entitlementEntity *entitlement.Entitlement, calculatedAt time.Time, options ...handleOption) (marshaler.Event, error) {
	if entitlementEntity == nil {
		return nil, fmt.Errorf("entitlement entity is nil")
	}

	opts := getOptions(options...)

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

	// Reset events are always recalculated asOf the time of reset, so that we have a snapshot of initial grants
	// and overages.
	if lo.FromPtr(opts.sourceOperation) == snapshot.ValueOperationReset {
		if entitlementEntity.CurrentUsagePeriod == nil {
			return nil, fmt.Errorf("entitlement has no current usage period, cannot create snapshot event")
		}

		snapshot, err := w.createSnapshotEvent(ctx, entitlementEntity, entitlementEntity.CurrentUsagePeriod.From, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create entitlement update snapshot event: %w", err)
		}

		return snapshot, nil
	}

	snapshot, err := w.createSnapshotEvent(ctx, entitlementEntity, calculatedAt, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create entitlement update snapshot event: %w", err)
	}

	_ = w.highWatermarkCache.Add(entitlementEntity.ID, highWatermarkCacheEntry{
		HighWatermark: calculatedAt.Add(-defaultClockDrift),
	})

	return snapshot, nil
}

type snapshotToEventInput struct {
	Entitlement       *entitlement.Entitlement
	Feature           *feature.Feature
	Value             *snapshot.EntitlementValue
	CalculatedAt      time.Time
	Source            string
	OverrideOperation *snapshot.ValueOperationType
}

func (i *snapshotToEventInput) Validate() error {
	var errs []error

	if i.Value == nil {
		errs = append(errs, fmt.Errorf("entitlement value is required"))
	}

	if i.Entitlement == nil {
		errs = append(errs, fmt.Errorf("entitlement is required"))
	}

	if i.Feature == nil {
		errs = append(errs, fmt.Errorf("feature is required"))
	}

	if i.CalculatedAt.IsZero() {
		errs = append(errs, fmt.Errorf("calculatedAt is required"))
	}

	if i.Source == "" {
		errs = append(errs, fmt.Errorf("source is required"))
	}

	if i.OverrideOperation != nil {
		if err := i.OverrideOperation.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("overrideOperation is invalid: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (w *Worker) snapshotToEvent(ctx context.Context, in snapshotToEventInput) (marshaler.Event, error) {
	if err := in.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	subject := models.Subject{
		Key: in.Entitlement.SubjectKey,
	}

	if w.opts.SubjectResolver != nil {
		var err error
		subject, err = w.opts.SubjectResolver.GetSubjectByKey(ctx, in.Entitlement.Namespace, in.Entitlement.SubjectKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get subject ID: %w", err)
		}
	}

	event := marshaler.WithSource(
		in.Source,
		snapshot.SnapshotEvent{
			Entitlement: *in.Entitlement,
			Namespace: models.NamespaceID{
				ID: in.Entitlement.Namespace,
			},
			Subject:   subject,
			Feature:   *in.Feature,
			Operation: lo.FromPtrOr(in.OverrideOperation, snapshot.ValueOperationUpdate),

			CalculatedAt: &in.CalculatedAt,

			Value:              in.Value,
			CurrentUsagePeriod: in.Entitlement.CurrentUsagePeriod,
		},
	)
	return event, nil
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

	return w.snapshotToEvent(ctx, snapshotToEventInput{
		Entitlement:       entitlementEntity,
		Feature:           feature,
		Value:             convert.ToPointer((snapshot.EntitlementValue)(mappedValues)),
		CalculatedAt:      calculatedAt,
		Source:            opts.source,
		OverrideOperation: opts.sourceOperation,
	})
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
