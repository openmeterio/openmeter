package balanceworker

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/hashicorp/go-multierror"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/event/spec"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/sink/flushhandler/ingestnotification"
	"github.com/openmeterio/openmeter/internal/watermill"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (w *Worker) handleBatchedIngestEvent(ctx context.Context, event ingestnotification.BatchedIngestEvent) ([]*message.Message, error) {
	filters := slicesx.Map(event.Events, func(e ingestnotification.IngestEventData) IngestEventQueryFilter {
		return IngestEventQueryFilter{
			Namespace:  e.Namespace.ID,
			SubjectKey: e.SubjectKey,
			MeterSlugs: e.MeterSlugs,
		}
	})
	affectedEntitlements, repoErr := w.connectors.repo.ListAffectedEntitlements(ctx, filters)
	if repoErr != nil {
		ev, err := spec.NewCloudEvent(
			spec.EventSpec{
				Source: spec.ComposeResourcePathRaw(string(ingestnotification.BatchedIngestEvent{}.Spec().Subsystem)), // TODO: this should be a different source
			},
			event,
		)
		if err != nil {
			return nil, err
		}

		msg, err := w.opts.Marshaler.MarshalEvent(ev)
		if err != nil {
			return nil, err
		}
		return nil, watermill.NewRetryableError([]*message.Message{msg}, repoErr)
	}

	failedEntitlementInfo := make([]IngestEventDataResponse, 0)
	var handlingError error

	result := make([]*message.Message, 0, len(affectedEntitlements))
	for _, entitlement := range affectedEntitlements {
		messages, err := w.handleEntitlementUpdateEvent(
			ctx,
			NamespacedID{Namespace: entitlement.Namespace, ID: entitlement.EntitlementID},
			spec.ComposeResourcePath(entitlement.Namespace, spec.EntityEvent),
		)
		if err != nil {
			// TODO: add error information too
			handlingError = multierror.Append(handlingError, err)
			failedEntitlementInfo = append(failedEntitlementInfo, entitlement)
			continue
		}

		result = append(result, messages...)
	}

	var err error
	if len(failedEntitlementInfo) > 0 {
		err = w.newEntitlementProcessingFailedError(failedEntitlementInfo, handlingError)
	}

	return result, err
}

func (w *Worker) GetEntitlementsAffectedByMeterSubject(ctx context.Context, namespace string, meterSlugs []string, subject string) ([]NamespacedID, error) {
	featuresByMeter, err := w.connectors.entitlement.Feature.ListFeatures(ctx, productcatalog.ListFeaturesParams{
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

	entitlements, err := w.connectors.entitlement.Entitlement.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
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
