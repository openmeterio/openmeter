package sink

import (
	"context"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type Storage interface {
	BatchInsert(ctx context.Context, messages []sinkmodels.SinkMessage) error
}

type ClickHouseStorageConfig struct {
	Streaming streaming.Connector
}

func (c ClickHouseStorageConfig) Validate() error {
	if c.Streaming == nil {
		return fmt.Errorf("streaming connection is required")
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
	var rawEvents []streaming.RawEvent

	for _, message := range messages {
		storedAt := time.Now()
		ingestedAt := storedAt

		if message.KafkaMessage != nil {
			for _, header := range message.KafkaMessage.Headers {
				// Parse ingested_at header
				if header.Key == "ingested_at" {
					var err error

					ingestedAt, err = time.Parse(time.RFC3339, string(header.Value))
					if err != nil {
						return fmt.Errorf("failed to parse ingested_at header: %s", err)
					}
				}
			}
		}

		rawEvent := streaming.RawEvent{
			Namespace:  message.Namespace,
			ID:         message.Serialized.Id,
			Type:       message.Serialized.Type,
			Source:     message.Serialized.Source,
			Subject:    message.Serialized.Subject,
			Time:       time.Unix(message.Serialized.Time, 0),
			Data:       message.Serialized.Data,
			IngestedAt: ingestedAt,
			StoredAt:   storedAt,
			StoreRowID: ulid.Make().String(),
		}

		rawEvents = append(rawEvents, rawEvent)
	}

	if err := c.config.Streaming.BatchInsert(ctx, rawEvents); err != nil {
		return fmt.Errorf("failed to store events: %w", err)
	}

	return nil
}
