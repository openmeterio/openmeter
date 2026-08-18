package sink

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

type Storage interface {
	BatchInsert(ctx context.Context, messages []sinkmodels.SinkMessage) error
	HasEvents(ctx context.Context, items []dedupe.Item) (dedupe.ItemSet, error)
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

const existsQueryBatchSize = 200

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

func (c *ClickHouseStorage) HasEvents(ctx context.Context, items []dedupe.Item) (dedupe.ItemSet, error) {
	durableItems := make(dedupe.ItemSet, len(items))

	if len(items) == 0 {
		return durableItems, nil
	}

	type groupKey struct {
		namespace string
		source    string
	}

	groupedItems := map[groupKey][]string{}

	for _, item := range items {
		if item.Namespace == "" || item.Source == "" || item.ID == "" {
			return nil, fmt.Errorf("failed to check event durability: namespace, source and id are required")
		}

		key := groupKey{namespace: item.Namespace, source: item.Source}
		groupedItems[key] = append(groupedItems[key], item.ID)
	}

	for key, ids := range groupedItems {
		ids = lo.Uniq(ids)
		slices.Sort(ids)

		for offset := 0; offset < len(ids); offset += existsQueryBatchSize {
			end := min(offset+existsQueryBatchSize, len(ids))
			batchIDs := ids[offset:end]

			foundItems, err := c.queryExistingItemsByNamespaceAndSource(ctx, key.namespace, key.source, batchIDs)
			if err != nil {
				return nil, err
			}

			for item := range foundItems {
				durableItems[item] = struct{}{}
			}
		}
	}

	return durableItems, nil
}

func (c *ClickHouseStorage) queryExistingItemsByNamespaceAndSource(ctx context.Context, namespace string, source string, ids []string) (dedupe.ItemSet, error) {
	result := dedupe.ItemSet{}

	if len(ids) == 0 {
		return result, nil
	}

	pageLimit := min(len(ids)*2, 1000)
	if pageLimit < 100 {
		pageLimit = 100
	}

	idFilter := filter.FilterString{In: &ids}
	sourceFilter := filter.FilterString{Eq: &source}

	var cursor *pagination.Cursor
	expectedIDs := map[string]struct{}{}
	for _, id := range ids {
		expectedIDs[id] = struct{}{}
	}

	for {
		events, err := c.config.Streaming.ListEventsV2(ctx, streaming.ListEventsV2Params{
			Namespace: namespace,
			Cursor:    cursor,
			Limit:     &pageLimit,
			ID:        &idFilter,
			Source:    &sourceFilter,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to check event durability: %w", err)
		}

		if len(events) == 0 {
			break
		}

		for _, event := range events {
			item := dedupe.Item{
				Namespace: namespace,
				Source:    source,
				ID:        event.ID,
			}
			result[item] = struct{}{}
		}

		if len(result) == len(expectedIDs) || len(events) < pageLimit {
			break
		}

		last := events[len(events)-1]
		cursor = &pagination.Cursor{
			ID:   last.StoreRowID,
			Time: last.Time,
		}
	}

	return result, nil
}
