package ingestnotification

import (
	"context"
	"errors"
	"log/slog"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/metric"

	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
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
	filtered := lo.Filter(events, func(event sinkmodels.SinkMessage, _ int) bool {
		// We explicityl ignore non-parseable & non-meter affecting events
		return event.Serialized != nil && len(event.Meters) > 0
	})

	if len(filtered) == 0 {
		h.logger.Debug("no events to process in batch for ingest notification")
		return nil
	}

	// Map the filtered events to the ingest event
	iEvents := slicesx.Map(filtered, func(message sinkmodels.SinkMessage) ingestevents.EventBatchedIngest {
		return ingestevents.EventBatchedIngest{
			Namespace:  eventmodels.NamespaceID{ID: message.Namespace},
			SubjectKey: message.Serialized.Subject,
			MeterSlugs: h.getMeterSlugsFromMeters(message.Meters),
		}
	})

	// Let's group the events by subject
	iEventsBySubject := lo.GroupBy(iEvents, func(event ingestevents.EventBatchedIngest) string {
		return event.Namespace.ID + "/" + event.SubjectKey
	})

	// Let's merge the events by subject
	iEvents = make([]ingestevents.EventBatchedIngest, 0, len(iEventsBySubject))
	for _, events := range iEventsBySubject {
		if len(events) == 0 {
			continue
		}

		if len(events) == 1 {
			iEvents = append(iEvents, events[0])
			continue
		}

		meterSlugs := make([]string, 0, len(events))

		for _, event := range events[1:] {
			meterSlugs = append(meterSlugs, event.MeterSlugs...)
		}

		iEvents = append(iEvents, ingestevents.EventBatchedIngest{
			Namespace:  events[0].Namespace,
			SubjectKey: events[0].SubjectKey,
			MeterSlugs: lo.Uniq(meterSlugs),
		})
	}

	// We need to chunk the events to not exceed message size limits
	for _, event := range iEvents {
		if err := h.publisher.Publish(ctx, event); err != nil {
			finalErr = errors.Join(finalErr, err)
			h.logger.ErrorContext(ctx, "failed to publish change notification", "error", err)
		}
	}

	return finalErr
}

func (h *handler) getMeterSlugsFromMeters(meters []meter.Meter) []string {
	slugs := make([]string, len(meters))
	for i, meter := range meters {
		slugs[i] = meter.Key
	}

	return slugs
}
