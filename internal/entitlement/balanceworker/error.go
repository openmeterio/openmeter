package balanceworker

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/spec"
	"github.com/openmeterio/openmeter/internal/sink/flushhandler/ingestnotification"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type entitlementProcessingFailedError struct {
	retryMessages []*message.Message
	err           error
}

func (e *entitlementProcessingFailedError) Error() string {
	return fmt.Errorf("entitlement processing failed for %v entitlements: %w", len(e.retryMessages), e.err).Error()
}

func (e *entitlementProcessingFailedError) RetryMessages() []*message.Message {
	return nil
}

// newEntitlementProcessingFailedError attempts to create a RetryableError, but if it fails it returns a regular error
func (w *Worker) newEntitlementProcessingFailedError(entitlements []IngestEventDataResponse, ogErr error) error {
	// We turn the entitlements into a single ingestnotification.BatchedIngestEvent message that exactly covers the provided entitlements

	// Map the filtered events to the ingest event
	iEvents := slicesx.Map(
		slicesx.Filter(entitlements, func(ent IngestEventDataResponse) bool {
			return ent.MeterSlug != nil
		}),
		func(ent IngestEventDataResponse) ingestnotification.IngestEventData {
			// We can optimize this if needed so we group the events more efficiently
			return ingestnotification.IngestEventData{
				Namespace:  models.NamespaceID{ID: ent.Namespace},
				SubjectKey: ent.SubjectKey,
				MeterSlugs: []string{*ent.MeterSlug},
			}
		})

	event, err := spec.NewCloudEvent(spec.EventSpec{
		ID:     ulid.Make().String(), // If we're using ID for correlation then this breaks that chain
		Source: spec.ComposeResourcePathRaw(string(ingestnotification.EventBatchedIngest{}.Spec().Subsystem)),
	}, ingestnotification.EventBatchedIngest{
		Events: iEvents,
	})
	if err != nil {
		return err
	}

	msg, err := w.opts.Marshaler.MarshalEvent(event)
	if err != nil {
		return err
	}

	return &entitlementProcessingFailedError{
		retryMessages: []*message.Message{msg},
		err:           ogErr,
	}
}
