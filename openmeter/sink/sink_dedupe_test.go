package sink

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
)

type countingDeduplicator struct {
	existing              dedupe.ItemSet
	checkUniqueBatchCalls int
	checkUniqueBatchSizes []int
	checkUniqueCalls      int
	isUniqueCalls         int
	setCalls              int
	setBatchSizes         []int
}

func newCountingDeduplicator(existing []dedupe.Item) *countingDeduplicator {
	itemSet := dedupe.ItemSet{}
	for _, item := range existing {
		itemSet[item] = struct{}{}
	}

	return &countingDeduplicator{
		existing: itemSet,
	}
}

func (d *countingDeduplicator) IsUnique(context.Context, string, event.Event) (bool, error) {
	d.isUniqueCalls++
	return false, nil
}

func (d *countingDeduplicator) CheckUnique(context.Context, dedupe.Item) (bool, error) {
	d.checkUniqueCalls++
	return false, nil
}

func (d *countingDeduplicator) Set(_ context.Context, items ...dedupe.Item) ([]dedupe.Item, error) {
	d.setCalls++
	d.setBatchSizes = append(d.setBatchSizes, len(items))

	existingItems := []dedupe.Item{}
	for _, item := range items {
		if _, ok := d.existing[item]; ok {
			existingItems = append(existingItems, item)
			continue
		}
		d.existing[item] = struct{}{}
	}

	return existingItems, nil
}

func (d *countingDeduplicator) CheckUniqueBatch(_ context.Context, items []dedupe.Item) (dedupe.CheckUniqueBatchResult, error) {
	d.checkUniqueBatchCalls++
	d.checkUniqueBatchSizes = append(d.checkUniqueBatchSizes, len(items))

	result := dedupe.CheckUniqueBatchResult{
		UniqueItems:           make(dedupe.ItemSet, len(items)),
		AlreadyProcessedItems: make(dedupe.ItemSet, len(items)),
	}

	for _, item := range items {
		if _, ok := d.existing[item]; ok {
			result.AlreadyProcessedItems[item] = struct{}{}
			continue
		}
		result.UniqueItems[item] = struct{}{}
	}

	return result, nil
}

func (d *countingDeduplicator) Close() error {
	return nil
}

type countingStorage struct {
	durable        dedupe.ItemSet
	hasEventsCalls int
	hasEventsSizes []int
}

func newCountingStorage(durable []dedupe.Item) *countingStorage {
	itemSet := dedupe.ItemSet{}
	for _, item := range durable {
		itemSet[item] = struct{}{}
	}

	return &countingStorage{
		durable: itemSet,
	}
}

func (s *countingStorage) BatchInsert(_ context.Context, _ []sinkmodels.SinkMessage) error {
	return nil
}

func (s *countingStorage) HasEvents(_ context.Context, items []dedupe.Item) (dedupe.ItemSet, error) {
	s.hasEventsCalls++
	s.hasEventsSizes = append(s.hasEventsSizes, len(items))

	result := dedupe.ItemSet{}
	for _, item := range items {
		if _, ok := s.durable[item]; ok {
			result[item] = struct{}{}
		}
	}

	return result, nil
}

func TestFilterMessagesForInsert_BatchedAndCrashSafeBoundaries(t *testing.T) {
	redisExistingNotDurable := testSinkMessage("tenant-a", "evt-redis-not-durable", "gateway")
	redisExistingDurable := testSinkMessage("tenant-a", "evt-redis-durable", "gateway")
	redisUniqueDurable := testSinkMessage("tenant-a", "evt-unique-durable", "gateway")
	redisUniqueNotDurable := testSinkMessage("tenant-a", "evt-unique-not-durable", "gateway")

	deduplicator := newCountingDeduplicator([]dedupe.Item{
		redisExistingNotDurable.GetDedupeItem(),
		redisExistingDurable.GetDedupeItem(),
	})
	storage := newCountingStorage([]dedupe.Item{
		redisExistingDurable.GetDedupeItem(),
		redisUniqueDurable.GetDedupeItem(),
	})

	s := &Sink{
		config: SinkConfig{
			Deduplicator: deduplicator,
			Storage:      storage,
		},
	}

	filtered, err := s.filterMessagesForInsert(t.Context(), []sinkmodels.SinkMessage{
		redisExistingNotDurable,
		redisExistingDurable,
		redisUniqueDurable,
		redisUniqueNotDurable,
	})
	require.NoError(t, err)
	require.Equal(t, []sinkmodels.SinkMessage{
		redisUniqueNotDurable,
		redisExistingNotDurable,
	}, filtered)

	require.Equal(t, 1, deduplicator.checkUniqueBatchCalls)
	require.Equal(t, []int{4}, deduplicator.checkUniqueBatchSizes)
	require.Equal(t, 0, deduplicator.isUniqueCalls)
	require.Equal(t, 0, deduplicator.checkUniqueCalls)

	require.Equal(t, 2, storage.hasEventsCalls)
	require.Equal(t, []int{2, 2}, storage.hasEventsSizes)
}

func TestDedupeSet_UsesSingleBatchCall(t *testing.T) {
	deduplicator := newCountingDeduplicator(nil)
	storage := newCountingStorage(nil)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	s := &Sink{
		config: SinkConfig{
			Logger:       logger,
			Tracer:       noop.NewTracerProvider().Tracer("test"),
			Deduplicator: deduplicator,
			Storage:      storage,
		},
	}

	err := s.dedupeSet(t.Context(), []sinkmodels.SinkMessage{
		testSinkMessage("tenant-a", "evt-1", "gateway"),
		testSinkMessage("tenant-a", "evt-2", "gateway"),
		testSinkMessage("tenant-a", "evt-3", "gateway"),
	})
	require.NoError(t, err)
	require.Equal(t, 1, deduplicator.setCalls)
	require.Equal(t, []int{3}, deduplicator.setBatchSizes)
}

func testSinkMessage(namespace, id, source string) sinkmodels.SinkMessage {
	return sinkmodels.SinkMessage{
		Namespace: namespace,
		Serialized: &serializer.CloudEventsKafkaPayload{
			Id:      id,
			Source:  source,
			Type:    "model.inference",
			Subject: "subject-1",
			Time:    time.Now().Unix(),
			Data:    `{"tokens":42}`,
		},
		Status: sinkmodels.ProcessingStatus{
			State: sinkmodels.OK,
		},
	}
}
