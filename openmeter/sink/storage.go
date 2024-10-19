package sink

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse_connector"
)

type Storage interface {
	BatchInsert(ctx context.Context, messages []sinkmodels.SinkMessage) error
}

type ClickHouseStorageConfig struct {
	ClickHouse      clickhouse.Conn
	Database        string
	AsyncInsert     bool
	AsyncInsertWait bool
	QuerySettings   map[string]string
}

func (c ClickHouseStorageConfig) Validate() error {
	if c.ClickHouse == nil {
		return fmt.Errorf("clickhouse connection is required")
	}

	if c.Database == "" {
		return fmt.Errorf("database is required")
	}

	return nil
}

func NewClickhouseStorage(config ClickHouseStorageConfig) (*ClickHouseStorage, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &ClickHouseStorage{
		config: config,
	}, nil
}

type ClickHouseStorage struct {
	config ClickHouseStorageConfig
}

// BatchInsert inserts multiple messages into ClickHouse.
func (c *ClickHouseStorage) BatchInsert(ctx context.Context, messages []sinkmodels.SinkMessage) error {
	var rawEvents []clickhouse_connector.CHEvent
	var meterEvents []clickhouse_connector.CHMeterEvent

	for _, message := range messages {
		var eventErr string
		if message.Status.Error != nil {
			eventErr = message.Status.Error.Error()
		}

		storedAt := time.Now()
		ingestedAt := storedAt

		if message.KafkaMessage != nil {
			for _, header := range message.KafkaMessage.Headers {
				// Parse ingested_at header
				if header.Key == "ingested_at" {
					var err error

					ingestedAt, err = time.Parse(time.RFC3339, string(header.Value))
					if err != nil {
						eventErr = fmt.Sprintf("failed to parse ingested_at header: %s", err)
					}
				}
			}
		}

		rawEvent := clickhouse_connector.CHEvent{
			Namespace:       message.Namespace,
			ValidationError: eventErr,
			ID:              message.Serialized.Id,
			Type:            message.Serialized.Type,
			Source:          message.Serialized.Source,
			Subject:         message.Serialized.Subject,
			Time:            message.Serialized.Time,
			Data:            message.Serialized.Data,
			IngestedAt:      ingestedAt,
			StoredAt:        storedAt,
		}

		rawEvents = append(rawEvents, rawEvent)

		// Meter events per meter
		for _, meterEvent := range message.MeterEvents {
			meterEvent := clickhouse_connector.CHMeterEvent{
				Namespace:   message.Namespace,
				Time:        rawEvent.Time,
				Meter:       meterEvent.Meter.ID,
				Subject:     rawEvent.Subject,
				Value:       meterEvent.Value,
				GroupBy:     meterEvent.GroupBy,
				EventID:     rawEvent.ID,
				EventSource: rawEvent.Source,
				EventType:   rawEvent.Type,
				StoredAt:    rawEvent.StoredAt,
				IngestedAt:  rawEvent.IngestedAt,
			}

			meterEvents = append(meterEvents, meterEvent)
		}
	}

	var err error

	// Insert raw events
	query := clickhouse_connector.InsertEventsQuery{
		Database:      c.config.Database,
		Events:        rawEvents,
		QuerySettings: c.config.QuerySettings,
	}
	sql, args := query.ToSQL()

	// By default, ClickHouse is writing data synchronously.
	// See https://clickhouse.com/docs/en/cloud/bestpractices/asynchronous-inserts
	if c.config.AsyncInsert {
		// With the `wait_for_async_insert` setting, you can configure
		// if you want an insert statement to return with an acknowledgment
		// either immediately after the data got inserted into the buffer.
		err = c.config.ClickHouse.AsyncInsert(ctx, sql, c.config.AsyncInsertWait, args...)
	} else {
		err = c.config.ClickHouse.Exec(ctx, sql, args...)
	}

	if err != nil {
		return fmt.Errorf("failed to batch insert raw events: %w", err)
	}

	// Insert meter events
	if len(meterEvents) > 0 {
		query := clickhouse_connector.InsertMeterEventsQuery{
			Database:      c.config.Database,
			MeterEvents:   meterEvents,
			QuerySettings: c.config.QuerySettings,
		}
		sql, args := query.ToSQL()

		if c.config.AsyncInsert {
			err = c.config.ClickHouse.AsyncInsert(ctx, sql, c.config.AsyncInsertWait, args...)
		} else {
			err = c.config.ClickHouse.Exec(ctx, sql, args...)
		}

		if err != nil {
			return fmt.Errorf("failed to batch insert meter events: %w", err)
		}
	}

	return nil
}
