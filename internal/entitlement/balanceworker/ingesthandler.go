package balanceworker

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/hashicorp/go-multierror"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/event/metadata"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	ingestevents "github.com/openmeterio/openmeter/internal/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (w *Worker) handleBatchedIngestEvent(ctx context.Context, event ingestevents.EventBatchedIngest) ([]*message.Message, error) {
	filters := slicesx.Map(event.Events, func(e ingestevents.IngestEventData) IngestEventQueryFilter {
		return IngestEventQueryFilter{
			Namespace:  e.Namespace.ID,
			SubjectKey: e.SubjectKey,
			MeterSlugs: e.MeterSlugs,
		}
	})
	affectedEntitlements, err := w.repo.ListAffectedEntitlements(ctx, filters)
	if err != nil {
		return nil, err
	}

	var handlingError error

	result := make([]*message.Message, 0, len(affectedEntitlements))
	for _, entitlement := range affectedEntitlements {
		messages, err := w.handleEntitlementUpdateEvent(
			ctx,
			NamespacedID{Namespace: entitlement.Namespace, ID: entitlement.EntitlementID},
			metadata.ComposeResourcePath(entitlement.Namespace, metadata.EntityEvent),
		)
		if err != nil {
			// TODO: add error information too
			handlingError = multierror.Append(handlingError, err)
			continue
		}

		result = append(result, messages...)
	}

	return result, handlingError
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
