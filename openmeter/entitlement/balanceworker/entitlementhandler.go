package balanceworker

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func (w *Worker) handleEntitlementEvent(ctx context.Context, entitlementID NamespacedID, source string) (marshaler.Event, error) {
	calculatedAt := time.Now()

	if entry, ok := w.highWatermarkCache.Get(entitlementID.ID); ok {
		if entry.HighWatermark.After(calculatedAt) || entry.IsDeleted {
			return nil, nil
		}
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
	return w.processEntitlementEntity(ctx, &entitlementEntity, calculatedAt, source)
}

func (w *Worker) processEntitlementEntity(ctx context.Context, entitlementEntity *entitlement.Entitlement, calculatedAt time.Time, source string) (marshaler.Event, error) {
	if entitlementEntity == nil {
		return nil, fmt.Errorf("entitlement entity is nil")
	}

	if entitlementEntity.DeletedAt != nil {
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

	snapshot, err := w.createSnapshotEvent(ctx, entitlementEntity, source, calculatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create entitlement update snapshot event: %w", err)
	}

	_ = w.highWatermarkCache.Add(entitlementEntity.ID, highWatermarkCacheEntry{
		HighWatermark: calculatedAt.Add(-defaultClockDrift),
	})

	return snapshot, nil
}

func (w *Worker) createSnapshotEvent(ctx context.Context, entitlementEntity *entitlement.Entitlement, source string, calculatedAt time.Time) (marshaler.Event, error) {
	feature, err := w.entitlement.Feature.GetFeature(ctx, entitlementEntity.Namespace, entitlementEntity.FeatureID, feature.IncludeArchivedFeatureTrue)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	calculationStart := time.Now()

	value, err := w.entitlement.Entitlement.GetEntitlementValue(ctx, entitlementEntity.Namespace, entitlementEntity.SubjectKey, entitlementEntity.ID, calculatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement value: %w", err)
	}

	w.metricRecalculationTime.Record(
		ctx,
		time.Since(calculationStart).Milliseconds(),
		metric.WithAttributes(
			attribute.String(metricAttributeKeyEntitltementType, string(entitlementEntity.EntitlementType)),
		),
	)

	w.metricRecalculationTimeOld.Record(ctx, time.Since(calculationStart).Milliseconds(), metric.WithAttributes(
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
		source,
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
