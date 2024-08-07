package ingestnotification

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel/metric"

	eventmodels "github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/sink/flushhandler"
	ingestevents "github.com/openmeterio/openmeter/internal/sink/flushhandler/ingestnotification/events"
	sinkmodels "github.com/openmeterio/openmeter/internal/sink/models"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type handler struct {
	publisher eventbus.Publisher
	logger    *slog.Logger
	config    HandlerConfig
}

type HandlerConfig struct {
	MaxEventsInBatch int
}

func NewHandler(logger *slog.Logger, metricMeter metric.Meter, publisher eventbus.Publisher, config HandlerConfig) (flushhandler.FlushEventHandler, error) {
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
	iEvents := slicesx.Map(filtered, func(message sinkmodels.SinkMessage) ingestevents.IngestEventData {
		return ingestevents.IngestEventData{
			Namespace:  eventmodels.NamespaceID{ID: message.Namespace},
			SubjectKey: message.Serialized.Subject,
			MeterSlugs: h.getMeterSlugsFromMeters(message.Meters),
		}
	})

	// We need to chunk the events to not exceed message size limits
	chunkedEvents := slicesx.Chunk(iEvents, h.config.MaxEventsInBatch)
	for _, chunk := range chunkedEvents {
		if err := h.publisher.Publish(ctx, ingestevents.EventBatchedIngest{Events: chunk}); err != nil {
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
