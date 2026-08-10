package sink

import (
	"context"
	"io"
	"log/slog"
	"slices"
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
	checkUniqueBatchItems [][]dedupe.Item
	checkUniqueCalls      int
	isUniqueCalls         int
	setCalls              int
	setBatchSizes         []int
	setBatchItems         [][]dedupe.Item
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
	d.setBatchItems = append(d.setBatchItems, slices.Clone(items))

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
	d.checkUniqueBatchItems = append(d.checkUniqueBatchItems, slices.Clone(items))

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
	hasEventsItems [][]dedupe.Item
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
	s.hasEventsItems = append(s.hasEventsItems, slices.Clone(items))

	result := dedupe.ItemSet{}
	for _, item := range items {
		if _, ok := s.durable[item]; ok {
			result[item] = struct{}{}
		}
	}

	return result, nil
}

func newTestSink(deduplicator dedupe.Deduplicator, storage Storage) *Sink {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return &Sink{
		config: SinkConfig{
			Logger:       logger,
			Tracer:       noop.NewTracerProvider().Tracer("test"),
			Deduplicator: deduplicator,
			Storage:      storage,
		},
	}
}

func TestPlanFlushMessages_IgnoresDropMissingIdentity(t *testing.T) {
	uniqueNotDurable := testSinkMessage("tenant-a", "evt-unique-not-durable", "gateway")
	dropMissingIdentity := sinkmodels.SinkMessage{
		Status: sinkmodels.ProcessingStatus{
			State: sinkmodels.DROP,
		},
	}

	deduplicator := newCountingDeduplicator(nil)
	storage := newCountingStorage(nil)

	s := newTestSink(deduplicator, storage)

	plan, err := s.planFlushMessages(t.Context(), []sinkmodels.SinkMessage{
		dropMissingIdentity,
		uniqueNotDurable,
	})
	require.NoError(t, err)
	require.Equal(t, []sinkmodels.SinkMessage{uniqueNotDurable}, plan.messagesToInsert)
	require.Equal(t, []dedupe.Item{uniqueNotDurable.GetDedupeItem()}, plan.dedupeItemsToSet)

	require.Equal(t, 1, deduplicator.checkUniqueBatchCalls)
	require.Equal(t, []int{1}, deduplicator.checkUniqueBatchSizes)
	require.Equal(t, 0, deduplicator.isUniqueCalls)
	require.Equal(t, 0, deduplicator.checkUniqueCalls)

	require.Equal(t, 1, storage.hasEventsCalls)
	require.Equal(t, []int{1}, storage.hasEventsSizes)
}

func TestPlanFlushMessages_UniqueDurableSetsDedupeWithoutInsert(t *testing.T) {
	uniqueDurable := testSinkMessage("tenant-a", "evt-unique-durable", "gateway")

	deduplicator := newCountingDeduplicator(nil)
	storage := newCountingStorage([]dedupe.Item{uniqueDurable.GetDedupeItem()})

	s := newTestSink(deduplicator, storage)

	plan, err := s.planFlushMessages(t.Context(), []sinkmodels.SinkMessage{uniqueDurable})
	require.NoError(t, err)
	require.Empty(t, plan.messagesToInsert)
	require.Equal(t, []dedupe.Item{uniqueDurable.GetDedupeItem()}, plan.dedupeItemsToSet)

	err = s.dedupeSet(t.Context(), plan.dedupeItemsToSet)
	require.NoError(t, err)
	require.Equal(t, 1, deduplicator.setCalls)
	require.Equal(t, []int{1}, deduplicator.setBatchSizes)
}

func TestPlanFlushMessages_ExistingNotDurableReinsertsWithoutSet(t *testing.T) {
	existingNotDurable := testSinkMessage("tenant-a", "evt-existing-not-durable", "gateway")

	deduplicator := newCountingDeduplicator([]dedupe.Item{existingNotDurable.GetDedupeItem()})
	storage := newCountingStorage(nil)

	s := newTestSink(deduplicator, storage)

	plan, err := s.planFlushMessages(t.Context(), []sinkmodels.SinkMessage{existingNotDurable})
	require.NoError(t, err)
	require.Equal(t, []sinkmodels.SinkMessage{existingNotDurable}, plan.messagesToInsert)
	require.Empty(t, plan.dedupeItemsToSet)

	err = s.dedupeSet(t.Context(), plan.dedupeItemsToSet)
	require.NoError(t, err)
	require.Equal(t, 0, deduplicator.setCalls)
}

func TestPlanFlushMessages_BatchesChecksAndSingleSetPipeline(t *testing.T) {
	uniqueNotDurable := testSinkMessage("tenant-a", "evt-unique-not-durable", "gateway")
	uniqueDurable := testSinkMessage("tenant-a", "evt-unique-durable", "gateway")
	existingNotDurable := testSinkMessage("tenant-a", "evt-existing-not-durable", "gateway")
	existingDurable := testSinkMessage("tenant-a", "evt-existing-durable", "gateway")

	deduplicator := newCountingDeduplicator([]dedupe.Item{
		existingNotDurable.GetDedupeItem(),
		existingDurable.GetDedupeItem(),
	})
	storage := newCountingStorage([]dedupe.Item{
		uniqueDurable.GetDedupeItem(),
		existingDurable.GetDedupeItem(),
	})

	s := newTestSink(deduplicator, storage)

	plan, err := s.planFlushMessages(t.Context(), []sinkmodels.SinkMessage{
		uniqueNotDurable,
		uniqueDurable,
		existingNotDurable,
		existingDurable,
	})
	require.NoError(t, err)
	require.Equal(t, []sinkmodels.SinkMessage{
		uniqueNotDurable,
		existingNotDurable,
	}, plan.messagesToInsert)
	require.Equal(t, []dedupe.Item{
		uniqueNotDurable.GetDedupeItem(),
		uniqueDurable.GetDedupeItem(),
	}, plan.dedupeItemsToSet)

	require.Equal(t, 1, deduplicator.checkUniqueBatchCalls)
	require.Equal(t, []int{4}, deduplicator.checkUniqueBatchSizes)
	require.Equal(t, 1, storage.hasEventsCalls)
	require.Equal(t, []int{4}, storage.hasEventsSizes)

	err = s.dedupeSet(t.Context(), plan.dedupeItemsToSet)
	require.NoError(t, err)
	require.Equal(t, 1, deduplicator.setCalls)
	require.Equal(t, []int{2}, deduplicator.setBatchSizes)
	require.Equal(t, []dedupe.Item{
		uniqueNotDurable.GetDedupeItem(),
		uniqueDurable.GetDedupeItem(),
	}, deduplicator.setBatchItems[0])
}

func TestDedupeSet_UsesSingleBatchCall(t *testing.T) {
	deduplicator := newCountingDeduplicator(nil)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := &Sink{
		config: SinkConfig{
			Logger:       logger,
			Tracer:       noop.NewTracerProvider().Tracer("test"),
			Deduplicator: deduplicator,
		},
	}

	err := s.dedupeSet(t.Context(), []dedupe.Item{
		{Namespace: "tenant-a", ID: "evt-1", Source: "gateway"},
		{Namespace: "tenant-a", ID: "evt-2", Source: "gateway"},
		{Namespace: "tenant-a", ID: "evt-3", Source: "gateway"},
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
