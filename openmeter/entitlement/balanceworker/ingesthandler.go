package balanceworker

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
)

func (w *Worker) handleBatchedIngestEvent(ctx context.Context, event ingestevents.EventBatchedIngest) error {
	affectedEntitlements, err := w.repo.ListAffectedEntitlements(ctx,
		[]IngestEventQueryFilter{
			{
				Namespace:  event.Namespace.ID,
				SubjectKey: event.SubjectKey,
				MeterSlugs: event.MeterSlugs,
			},
		})
	if err != nil {
		return err
	}

	var handlingError error

	for _, entitlement := range affectedEntitlements {
		event, err := w.handleEntitlementEvent(
			ctx,
			NamespacedID{Namespace: entitlement.Namespace, ID: entitlement.EntitlementID},
			WithEventAt(event.StoredAt),
			WithSource(metadata.ComposeResourcePath(entitlement.Namespace, metadata.EntityEvent)),
			WithRawIngestedEvents(event.RawEvents),
			WithEstimator(true),
		)
		if err != nil {
			err = fmt.Errorf("handling entitlement event for %s: %w", entitlement.EntitlementID, err)
			handlingError = errors.Join(handlingError, err)
			continue
		}

		if err := w.opts.EventBus.Publish(ctx, event); err != nil {
			handlingError = errors.Join(handlingError, fmt.Errorf("handling entitlement event for %s: %w", entitlement.EntitlementID, err))
		}
	}

	if handlingError != nil {
		w.opts.Logger.ErrorContext(ctx, "error handling batched ingest event", "error", handlingError)
	}

	return handlingError
}

func (w *Worker) GetEntitlementsAffectedByMeterSubject(ctx context.Context, namespace string, meterSlugs []string, subject string) ([]NamespacedID, error) {
	featuresByMeter, err := w.entitlement.Feature.ListFeatures(ctx, feature.ListFeaturesParams{
		Namespace:  namespace,
		MeterSlugs: meterSlugs,
	})
	if err != nil {
		return nil, err
	}

	featureIDs := make([]string, 0, len(featuresByMeter.Items))
	for _, feature := range featuresByMeter.Items {
		featureIDs = append(featureIDs, feature.ID)
	}

	entitlements, err := w.entitlement.Entitlement.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
		Namespaces:  []string{namespace},
		SubjectKeys: []string{subject},
		FeatureIDs:  featureIDs,
	})
	if err != nil {
		return nil, err
	}

	entitlementIDs := make([]NamespacedID, 0, len(entitlements.Items))
	for _, entitlement := range entitlements.Items {
		entitlementIDs = append(entitlementIDs, NamespacedID{
			ID:        entitlement.ID,
			Namespace: entitlement.Namespace,
		})
	}

	return entitlementIDs, nil
}
