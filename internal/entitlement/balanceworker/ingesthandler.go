package balanceworker

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/event/metadata"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	ingestevents "github.com/openmeterio/openmeter/internal/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (w *Worker) handleBatchedIngestEvent(ctx context.Context, event ingestevents.EventBatchedIngest) error {
	filters := slicesx.Map(event.Events, func(e ingestevents.IngestEventData) IngestEventQueryFilter {
		return IngestEventQueryFilter{
			Namespace:  e.Namespace.ID,
			SubjectKey: e.SubjectKey,
			MeterSlugs: e.MeterSlugs,
		}
	})
	affectedEntitlements, err := w.repo.ListAffectedEntitlements(ctx, filters)
	if err != nil {
		return err
	}

	var handlingError error

	for _, entitlement := range affectedEntitlements {
		event, err := w.handleEntitlementUpdateEvent(
			ctx,
			NamespacedID{Namespace: entitlement.Namespace, ID: entitlement.EntitlementID},
			metadata.ComposeResourcePath(entitlement.Namespace, metadata.EntityEvent),
		)
		if err != nil {
			// TODO: add error information too
			handlingError = errors.Join(handlingError, err)
			continue
		}

		if err := w.opts.EventBus.Publish(ctx, event); err != nil {
			handlingError = errors.Join(handlingError, err)
		}
	}

	return handlingError
}

func (w *Worker) GetEntitlementsAffectedByMeterSubject(ctx context.Context, namespace string, meterSlugs []string, subject string) ([]NamespacedID, error) {
	featuresByMeter, err := w.entitlement.Feature.ListFeatures(ctx, productcatalog.ListFeaturesParams{
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
