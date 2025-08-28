package balanceworker

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	pkgmodels "github.com/openmeterio/openmeter/pkg/models"
)

func (w *Worker) handleBatchedIngestEvent(ctx context.Context, ingestEvent ingestevents.EventBatchedIngest) error {
	affectedEntitlements, err := w.opts.Repo.ListAffectedEntitlements(ctx,
		[]IngestEventQueryFilter{
			{
				Namespace:  ingestEvent.Namespace.ID,
				SubjectKey: ingestEvent.SubjectKey,
				MeterSlugs: ingestEvent.MeterSlugs,
			},
		})
	if err != nil {
		return fmt.Errorf("failed to list affected entitlements: %w", err)
	}

	var errs []error

	for _, ent := range affectedEntitlements {
		event, err := w.handleEntitlementEvent(
			ctx,
			pkgmodels.NamespacedID{Namespace: ent.Namespace, ID: ent.EntitlementID},
			WithSource(metadata.ComposeResourcePath(ent.Namespace, metadata.EntityEvent)),
			WithEventAt(ingestEvent.StoredAt),
			WithRawIngestedEvents(ingestEvent.RawEvents),
		)
		if err != nil {
			errs = append(errs, fmt.Errorf("handling entitlement event for %s: %w", ent.EntitlementID, err))

			continue
		}

		if err = w.opts.EventBus.Publish(ctx, event); err != nil {
			errs = append(errs, fmt.Errorf("handling entitlement event for %s: %w", ent.EntitlementID, err))
		}
	}

	err = errors.Join(errs...)
	if err != nil {
		// This is a warning, as we might succeed in retrying the event later. The DLQ Telemetry middleware will properly log
		// the error.
		w.opts.Logger.WarnContext(ctx, "error handling batched ingest event", "error", err)
	}

	return err
}

func (w *Worker) GetEntitlementsAffectedByMeterSubject(ctx context.Context, namespace string, meterSlugs []string, subject string) ([]pkgmodels.NamespacedID, error) {
	featuresByMeter, err := w.opts.Entitlement.Feature.ListFeatures(ctx, feature.ListFeaturesParams{
		Namespace:  namespace,
		MeterSlugs: meterSlugs,
	})
	if err != nil {
		return nil, err
	}

	featureIDs := make([]string, 0, len(featuresByMeter.Items))
	for _, feat := range featuresByMeter.Items {
		featureIDs = append(featureIDs, feat.ID)
	}

	entitlements, err := w.opts.Entitlement.Entitlement.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
		Namespaces:  []string{namespace},
		SubjectKeys: []string{subject},
		FeatureIDs:  featureIDs,
	})
	if err != nil {
		return nil, err
	}

	entitlementIDs := make([]pkgmodels.NamespacedID, 0, len(entitlements.Items))
	for _, ent := range entitlements.Items {
		entitlementIDs = append(entitlementIDs, pkgmodels.NamespacedID{
			ID:        ent.ID,
			Namespace: ent.Namespace,
		})
	}

	return entitlementIDs, nil
}
