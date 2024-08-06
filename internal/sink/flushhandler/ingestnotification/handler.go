package ingestnotification

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel/metric"

	eventmodels "github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/spec"
	"github.com/openmeterio/openmeter/internal/sink/flushhandler"
	sinkmodels "github.com/openmeterio/openmeter/internal/sink/models"
	"github.com/openmeterio/openmeter/openmeter/event"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type handler struct {
	publisher event.Publisher
	logger    *slog.Logger
	config    HandlerConfig
}

type HandlerConfig struct {
	MaxEventsInBatch int
}

func NewHandler(logger *slog.Logger, metricMeter metric.Meter, publisher event.Publisher, config HandlerConfig) (flushhandler.FlushEventHandler, error) {
	handler := &handler{
		publisher: publisher,
		logger:    logger,
		config:    config,
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

	// We need to chunk the events to not exceed message size limits
	chunkedEvents := slicesx.Chunk(iEvents, h.config.MaxEventsInBatch)
	for _, chunk := range chunkedEvents {
		event, err := spec.NewCloudEvent(spec.EventSpec{
			Source: spec.ComposeResourcePathRaw(string(EventBatchedIngest{}.Spec().Subsystem)),
		}, EventBatchedIngest{
			Events: chunk,
		})
		if err != nil {
			finalErr = errors.Join(finalErr, err)
			h.logger.Error("failed to create change notification", "error", err)
			continue
		}

		if err := h.publisher.Publish(ctx, event); err != nil {
			finalErr = errors.Join(finalErr, err)
			h.logger.Error("failed to publish change notification", "error", err)
		}
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
