package balanceworker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker/events"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	pkgmodels "github.com/openmeterio/openmeter/pkg/models"
)

func (w *Worker) handleBatchedIngestEvent(ctx context.Context, event ingestevents.EventBatchedIngest) error {
	affectedEntitlements, err := w.opts.Repo.ListEntitlementsAffectedByIngestEvents(ctx, IngestEventQueryFilter{
		Namespace:    event.Namespace.ID,
		EventSubject: event.SubjectKey,
		MeterSlugs:   event.MeterSlugs,
	})
	if err != nil {
		return fmt.Errorf("failed to list affected entitlements: %w", err)
	}

	eventTimestamps := lo.Map(event.RawEvents, func(event serializer.CloudEventsKafkaPayload, _ int) time.Time {
		return time.Unix(event.Time, 0)
	})

	var errs []error

	for _, ent := range affectedEntitlements {
		// We don't care about deleted entitlements as a final event is sent when the entitlement is deleted
		if ent.DeletedAt != nil {
			continue
		}

		couldEventAffectEntitlement := lo.SomeBy(eventTimestamps, func(timestamp time.Time) bool {
			return ent.GetEntitlementActivityPeriod().Contains(timestamp)
		})

		// If the event cannot affect the entitlement, we can skip it
		if !couldEventAffectEntitlement {
			continue
		}

		err := w.opts.EventBus.Publish(ctx, events.RecalculateEvent{
			Entitlement: pkgmodels.NamespacedID{Namespace: ent.Namespace, ID: ent.EntitlementID},

			OriginalEventSource: metadata.ComposeResourcePath(ent.Namespace, metadata.EntityEvent),
			AsOf:                event.StoredAt,
			SourceOperation:     events.OperationTypeIngest,
			RawIngestedEvents:   event.RawEvents,
		})
		if err != nil {
			return fmt.Errorf("failed to publish recalculate event: %w", err)
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
