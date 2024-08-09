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

	subjectID := ""
	if w.opts.SubjectIDResolver != nil {
		subjectID, err = w.opts.SubjectIDResolver.GetSubjectIDByKey(ctx, namespace, delEvent.SubjectKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get subject ID: %w", err)
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
			Subject: models.SubjectKeyAndID{
				Key: delEvent.SubjectKey,
				ID:  subjectID,
			},
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
	entitlement, err := w.entitlement.Entitlement.GetEntitlement(ctx, entitlementID.Namespace, entitlementID.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement: %w", err)
	}

	feature, err := w.entitlement.Feature.GetFeature(ctx, entitlementID.Namespace, entitlement.FeatureID, productcatalog.IncludeArchivedFeatureTrue)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	value, err := w.entitlement.Entitlement.GetEntitlementValue(ctx, entitlementID.Namespace, entitlement.SubjectKey, entitlement.ID, calculatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement value: %w", err)
	}

	mappedValues, err := entitlementdriver.MapEntitlementValueToAPI(value)
	if err != nil {
		return nil, fmt.Errorf("failed to map entitlement value: %w", err)
	}

	subjectID := ""
	if w.opts.SubjectIDResolver != nil {
		subjectID, err = w.opts.SubjectIDResolver.GetSubjectIDByKey(ctx, entitlementID.Namespace, entitlement.SubjectKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get subject ID: %w", err)
		}
	}

	event := marshaler.WithSource(
		source,
		snapshot.SnapshotEvent{
			Entitlement: *entitlement,
			Namespace: models.NamespaceID{
				ID: entitlementID.Namespace,
			},
			Subject: models.SubjectKeyAndID{
				Key: entitlement.SubjectKey,
				ID:  subjectID,
			},
			Feature:   *feature,
			Operation: snapshot.ValueOperationUpdate,

			CalculatedAt: &calculatedAt,

			Value:              convert.ToPointer((snapshot.EntitlementValue)(mappedValues)),
			CurrentUsagePeriod: entitlement.CurrentUsagePeriod,
		},
	)

	return event, nil
}
