package summeterv1

import (
	"context"
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	"github.com/samber/lo"
)

type IngestedEventHandler struct {
	meterCache *MeterCache
	logger     *slog.Logger
	engine     *Engine
}

func (h *IngestedEventHandler) HandleIngestedEvent(ctx context.Context, event *ingestevents.EventBatchedIngest) error {
	if event == nil {
		return errors.New("event is nil")
	}

	eventsByType := lo.GroupBy(event.RawEvents, func(event serializer.CloudEventsKafkaPayload) string {
		return event.Type
	})

	records := make([]Record, 0, 128)
	for eventType, cloudEvents := range eventsByType {
		affectedMeters, err := h.meterCache.GetMetersByEventTypeNamespace(ctx, eventType, event.Namespace.ID)
		if err != nil {
			return err
		}

		for _, meter := range affectedMeters {
			if !h.engine.IsOperational(meter) {
				continue
			}

			for _, cloudEvent := range cloudEvents {
				record, err := h.engine.GetRecordForMeter(ctx, meter, cloudEvent, event.StoredAt)
				if err != nil {
					return err
				}

				if record == nil {
					continue
				}

				records = append(records, *record)
			}
		}
	}

	if len(records) > 0 {
		// TODO:
		// err := h.engine.InsertRecords(ctx, h.clickhouse, records)
		//if err != nil {
		//	return err
		//}
	}

	return nil
}
