package ingestnotification

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/metric"

	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
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

func (c HandlerConfig) Validate() error {
	if c.MaxEventsInBatch <= 0 {
		return errors.New("max_events_in_batch must be greater than 0")
	}

	return nil
}

func NewHandler(logger *slog.Logger, metricMeter metric.Meter, publisher eventbus.Publisher, config HandlerConfig) (flushhandler.FlushEventHandler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

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
// We could resolve the customers in the event processing / generation instead of in the handlers. There are pros and cons to both.
func (h *handler) OnFlushSuccess(ctx context.Context, events []sinkmodels.SinkMessage) error {
	var finalErr error

	// Filter meaningful events for downstream
	filtered := lo.Filter(events, func(event sinkmodels.SinkMessage, _ int) bool {
		return event.Serialized != nil
	})

	if len(filtered) == 0 {
		h.logger.Debug("no events to process in batch for ingest notification")
		return nil
	}

	now := time.Now()

	// Map the filtered events to the ingest event
	iEvents := slicesx.Map(filtered, func(message sinkmodels.SinkMessage) ingestevents.EventBatchedIngest {
		res := ingestevents.EventBatchedIngest{
			Namespace:  eventmodels.NamespaceID{ID: message.Namespace},
			SubjectKey: message.Serialized.Subject,
			MeterSlugs: h.getMeterSlugsFromMeters(message.Meters),
			// Warning: Given this is called after the clickhouse writes have completed, it's a fair assumption that
			// the event was stored at this time to clickhouse.
			StoredAt: now,
		}

		if message.Serialized != nil {
			res.RawEvents = append(res.RawEvents, *message.Serialized)
		}

		return res
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

		chunkedEvents := lo.Chunk(events, h.config.MaxEventsInBatch)

		for _, chunk := range chunkedEvents {
			event := ingestevents.EventBatchedIngest{
				Namespace:  chunk[0].Namespace,
				SubjectKey: chunk[0].SubjectKey,
				StoredAt:   now,
			}

			event.MeterSlugs = lo.Uniq(
				slices.Concat(
					lo.Map(chunk, func(event ingestevents.EventBatchedIngest, _ int) []string {
						return event.MeterSlugs
					})...,
				),
			)

			event.RawEvents = slices.Concat(
				lo.Map(chunk, func(event ingestevents.EventBatchedIngest, _ int) []serializer.CloudEventsKafkaPayload {
					return event.RawEvents
				})...,
			)

			iEvents = append(iEvents, event)
		}
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

func (h *handler) getMeterSlugsFromMeters(meters []*meter.Meter) []string {
	slugs := make([]string, len(meters))
	for i, meter := range meters {
		slugs[i] = meter.Key
	}

	return slugs
}
