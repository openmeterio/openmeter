package ingestnotification

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel/metric"

	"github.com/oklog/ulid/v2"
	eventmodels "github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/spec"
	"github.com/openmeterio/openmeter/internal/sink/flushhandler"
	sinkmodels "github.com/openmeterio/openmeter/internal/sink/models"
	"github.com/openmeterio/openmeter/openmeter/event/publisher"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type handler struct {
	publisher publisher.TopicPublisher
	logger    *slog.Logger
}

func NewHandler(logger *slog.Logger, metricMeter metric.Meter, publisher publisher.TopicPublisher) (flushhandler.FlushEventHandler, error) {
	handler := &handler{
		publisher: publisher,
		logger:    logger,
	}

	return flushhandler.NewFlushEventHandler(
		flushhandler.FlushEventHandlerOptions{
			Name:        "ingest_notification",
			Callback:    handler.OnFlushSuccess,
			Logger:      logger,
			MetricMeter: metricMeter,
		})
}

// OnFlushSuccess takes a look at the incoming messages and in case something is
// affecting a ledger balance it will create the relevant event.
func (h *handler) OnFlushSuccess(ctx context.Context, events []sinkmodels.SinkMessage) error {
	var finalErr error

	// Filter meaningful events for downstream
	filtered := slicesx.Filter(events, func(event sinkmodels.SinkMessage) bool {
		// We explicityl ignore non-parseable & non-meter affecting events
		return event.Serialized != nil && len(event.Meters) > 0
	})

	if len(filtered) == 0 {
		h.logger.Debug("no events to process in batch for ingest notification")
		return nil
	}

	// Map the filtered events to the ingest event
	iEvents := slicesx.Map(filtered, func(message sinkmodels.SinkMessage) IngestEventData {
		return IngestEventData{
			Namespace:  eventmodels.NamespaceID{ID: message.Namespace},
			SubjectKey: message.Serialized.Subject,
			MeterSlugs: h.getMeterSlugsFromMeters(message.Meters),
		}
	})

	event, err := spec.NewCloudEvent(spec.EventSpec{
		ID:     ulid.Make().String(), // If we're using ID for correlation then this breaks that chain
		Source: spec.ComposeResourcePathRaw(string(BatchedIngestEvent{}.Spec().Subsystem)),
	}, BatchedIngestEvent{
		Events: iEvents,
	})
	if err != nil {
		finalErr = errors.Join(finalErr, err)
		h.logger.Error("failed to create change notification", "error", err)
		return finalErr
	}

	if err := h.publisher.Publish(event); err != nil {
		finalErr = errors.Join(finalErr, err)
		h.logger.Error("failed to publish change notification", "error", err)
	}

	return finalErr
}

func (h *handler) getMeterSlugsFromMeters(meters []models.Meter) []string {
	slugs := make([]string, len(meters))
	for i, meter := range meters {
		slugs[i] = meter.Slug
	}

	return slugs
}
