package balanceworker

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/entitlement/httpdriver"
	"github.com/openmeterio/openmeter/internal/entitlement/snapshot"
	"github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/spec"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func (w *Worker) handleEntitlementDeleteEvent(ctx context.Context, delEvent entitlement.EntitlementDeletedEvent) ([]*message.Message, error) {
	namespace := delEvent.Namespace.ID

	feature, err := w.connectors.Feature.GetFeature(ctx, namespace, delEvent.FeatureID, productcatalog.IncludeArchivedFeatureTrue)
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

	calculationTime := w.getCalculationTime()

	event, err := spec.NewCloudEvent(
		spec.EventSpec{
			Source:  spec.ComposeResourcePath(namespace, spec.EntityEntitlement, delEvent.ID),
			Subject: spec.ComposeResourcePath(namespace, spec.EntitySubjectKey, delEvent.SubjectKey),
		},
		snapshot.EntitlementBalanceSnapshotEvent{
			Entitlement: delEvent.Entitlement,
			Namespace: models.NamespaceID{
				ID: namespace,
			},
			Subject: models.SubjectKeyAndID{
				Key: delEvent.SubjectKey,
				ID:  subjectID,
			},
			Feature:   *feature,
			Operation: snapshot.BalanceOperationDelete,

			CalculatedAt: convert.ToPointer(calculationTime),

			CurrentUsagePeriod: delEvent.CurrentUsagePeriod,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud event: %w", err)
	}

	wmMessage, err := w.opts.Marshaler.MarshalEvent(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cloud event: %w", err)
	}

	_ = w.highWatermarkCache.Add(delEvent.ID, highWatermarkCacheEntry{
		HighWatermark: calculationTime,
		IsDeleted:     true,
	})

	return []*message.Message{wmMessage}, nil
}

func (w *Worker) handleEntitlementUpdateEvent(ctx context.Context, entitlementID NamespacedID, source string) ([]*message.Message, error) {
	calculatedAt := w.getCalculationTime()

	if entry, ok := w.highWatermarkCache.Get(entitlementID.ID); ok {
		if entry.HighWatermark.After(calculatedAt) || entry.IsDeleted {
			return nil, nil
		}
	}

	wmMessage, err := w.createSnapshotEvent(ctx, entitlementID, source, calculatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create entitlement update snapshot event: %w", err)
	}

	_ = w.highWatermarkCache.Add(entitlementID.ID, highWatermarkCacheEntry{
		HighWatermark: calculatedAt,
	})

	return []*message.Message{wmMessage}, nil
}

func (w *Worker) createSnapshotEvent(ctx context.Context, entitlementID NamespacedID, source string, calculatedAt time.Time) (*message.Message, error) {
	entitlement, err := w.connectors.Entitlement.GetEntitlement(ctx, entitlementID.Namespace, entitlementID.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement: %w", err)
	}

	feature, err := w.connectors.Feature.GetFeature(ctx, entitlementID.Namespace, entitlement.FeatureID, productcatalog.IncludeArchivedFeatureTrue)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	value, err := w.connectors.Entitlement.GetEntitlementValue(ctx, entitlementID.Namespace, entitlement.SubjectKey, entitlement.ID, calculatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement value: %w", err)
	}

	mappedValues, err := httpdriver.MapEntitlementValueToAPI(value)
	if err != nil {
		return nil, fmt.Errorf("failed to map entitlement value: %w", err)
	}

	subjectID := ""
	if w.opts.SubjectIDResolver != nil {
		subjectID, err = w.opts.SubjectIDResolver.GetSubjectIDByKey(ctx, entitlementID.Namespace, entitlementID.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get subject ID: %w", err)
		}
	}

	event, err := spec.NewCloudEvent(
		spec.EventSpec{
			Source:  source,
			Subject: spec.ComposeResourcePath(entitlementID.Namespace, spec.EntitySubjectKey, entitlement.SubjectKey),
		},
		snapshot.EntitlementBalanceSnapshotEvent{
			Entitlement: *entitlement,
			Namespace: models.NamespaceID{
				ID: entitlementID.Namespace,
			},
			Subject: models.SubjectKeyAndID{
				Key: entitlement.SubjectKey,
				ID:  subjectID,
			},
			Feature:   *feature,
			Operation: snapshot.BalanceOperationUpdate,

			CalculatedAt: &calculatedAt,

			Balance:            convert.ToPointer((snapshot.EntitlementValue)(mappedValues)),
			CurrentUsagePeriod: entitlement.CurrentUsagePeriod,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud event: %w", err)
	}

	wmMessage, err := w.opts.Marshaler.MarshalEvent(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cloud event: %w", err)
	}

	return wmMessage, nil
}
