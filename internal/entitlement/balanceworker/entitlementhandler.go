package balanceworker

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/internal/entitlement/driver"
	"github.com/openmeterio/openmeter/internal/entitlement/snapshot"
	"github.com/openmeterio/openmeter/internal/event/metadata"
	"github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func (w *Worker) handleEntitlementDeleteEvent(ctx context.Context, delEvent entitlement.EntitlementDeletedEvent) (marshaler.Event, error) {
	namespace := delEvent.Namespace.ID

	feature, err := w.entitlement.Feature.GetFeature(ctx, namespace, delEvent.FeatureID, productcatalog.IncludeArchivedFeatureTrue)
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

	calculationTime := time.Now()

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

	_ = w.highWatermarkCache.Add(delEvent.ID, highWatermarkCacheEntry{
		HighWatermark: calculationTime.Add(-defaultClockDrift),
		IsDeleted:     true,
	})

	return event, nil
}

func (w *Worker) handleEntitlementUpdateEvent(ctx context.Context, entitlementID NamespacedID, source string) (marshaler.Event, error) {
	calculatedAt := time.Now()

	if entry, ok := w.highWatermarkCache.Get(entitlementID.ID); ok {
		if entry.HighWatermark.After(calculatedAt) || entry.IsDeleted {
			return nil, nil
		}
	}

	snapshot, err := w.createSnapshotEvent(ctx, entitlementID, source, calculatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create entitlement update snapshot event: %w", err)
	}

	_ = w.highWatermarkCache.Add(entitlementID.ID, highWatermarkCacheEntry{
		HighWatermark: calculatedAt.Add(-defaultClockDrift),
	})

	return snapshot, nil
}

func (w *Worker) createSnapshotEvent(ctx context.Context, entitlementID NamespacedID, source string, calculatedAt time.Time) (marshaler.Event, error) {
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

	entitlementEntity := &entitlements.Items[0]
	if entitlementEntity.DeletedAt != nil {
		// entitlement got deleted while processing changes => let's create a delete event so that we are not working
		// on entitlement updates that are not relevant anymore
		return w.handleEntitlementDeleteEvent(ctx, entitlement.EntitlementDeletedEvent{
			Entitlement: *entitlementEntity,
			Namespace:   models.NamespaceID{ID: entitlementID.Namespace},
		})
	}

	feature, err := w.entitlement.Feature.GetFeature(ctx, entitlementID.Namespace, entitlementEntity.FeatureID, productcatalog.IncludeArchivedFeatureTrue)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	value, err := w.entitlement.Entitlement.GetEntitlementValue(ctx, entitlementID.Namespace, entitlementEntity.SubjectKey, entitlementEntity.ID, calculatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement value: %w", err)
	}

	mappedValues, err := entitlementdriver.MapEntitlementValueToAPI(value)
	if err != nil {
		return nil, fmt.Errorf("failed to map entitlement value: %w", err)
	}

	subject := models.Subject{
		Key: entitlementEntity.SubjectKey,
	}
	if w.opts.SubjectResolver != nil {
		subject, err = w.opts.SubjectResolver.GetSubjectByKey(ctx, entitlementID.Namespace, entitlementEntity.SubjectKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get subject ID: %w", err)
		}
	}

	event := marshaler.WithSource(
		source,
		snapshot.SnapshotEvent{
			Entitlement: *entitlementEntity,
			Namespace: models.NamespaceID{
				ID: entitlementID.Namespace,
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
