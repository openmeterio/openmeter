package sink

import (
	"context"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
)

type Storage interface {
	BatchInsert(ctx context.Context, messages []sinkmodels.SinkMessage) error
	HasEvent(ctx context.Context, message sinkmodels.SinkMessage) (bool, error)
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

	fallbackNow := clock.Now()

	for _, message := range messages {
		rawEvent := streaming.RawEvent{
			Namespace:  message.Namespace,
			ID:         message.Serialized.Id,
			Type:       message.Serialized.Type,
			Source:     message.Serialized.Source,
			Subject:    message.Serialized.Subject,
			Time:       time.Unix(message.Serialized.Time, 0),
			Data:       message.Serialized.Data,
			IngestedAt: lo.CoalesceOrEmpty(lo.FromPtr(message.IngestedAt), fallbackNow),
			StoredAt:   lo.CoalesceOrEmpty(lo.FromPtr(message.StoredAt), fallbackNow),
			StoreRowID: ulid.Make().String(),
		}

		rawEvents = append(rawEvents, rawEvent)
	}

	if err := c.config.Streaming.BatchInsert(ctx, rawEvents); err != nil {
		return fmt.Errorf("failed to store events: %w", err)
	}

	return nil
}

func (c *ClickHouseStorage) HasEvent(ctx context.Context, message sinkmodels.SinkMessage) (bool, error) {
	if message.Serialized == nil {
		return false, fmt.Errorf("failed to check event durability: message payload is missing")
	}

	if message.Namespace == "" {
		return false, fmt.Errorf("failed to check event durability: namespace is missing")
	}

	if message.Serialized.Id == "" || message.Serialized.Source == "" {
		return false, fmt.Errorf("failed to check event durability: id and source are required")
	}

	limit := 1
	id := message.Serialized.Id
	source := message.Serialized.Source

	events, err := c.config.Streaming.ListEventsV2(ctx, streaming.ListEventsV2Params{
		Namespace: message.Namespace,
		Limit:     &limit,
		ID: &filter.FilterString{
			Eq: &id,
		},
		Source: &filter.FilterString{
			Eq: &source,
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to check event durability: %w", err)
	}

	return len(events) > 0, nil
}
