package sink

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
)

type batchDeduplicator struct {
	dedupe.Deduplicator
	calls     int
	items     []dedupe.Item
	processed dedupe.ItemSet
	err       error
}

func (d *batchDeduplicator) CheckUniqueBatch(_ context.Context, items []dedupe.Item) (dedupe.CheckUniqueBatchResult, error) {
	d.calls++
	d.items = items
	result := dedupe.CheckUniqueBatchResult{UniqueItems: dedupe.ItemSet{}, AlreadyProcessedItems: d.processed}
	for _, item := range items {
		if _, ok := d.processed[item]; !ok {
			result.UniqueItems[item] = struct{}{}
		}
	}
	return result, d.err
}

func TestDeduplicateAndResolveMeters(t *testing.T) {
	affectedMeter := &meter.Meter{Key: "usage"}
	newMessage := sinkmodels.SinkMessage{
		Namespace:  "ns",
		Serialized: &serializer.CloudEventsKafkaPayload{Id: "new", Source: "source", Type: "usage"},
	}
	oldMessage := sinkmodels.SinkMessage{
		Namespace:  "ns",
		Serialized: &serializer.CloudEventsKafkaPayload{Id: "old", Source: "source", Type: "usage"},
	}
	invalidMessage := sinkmodels.SinkMessage{
		Status: sinkmodels.ProcessingStatus{State: sinkmodels.DROP, DropError: errors.New("invalid")},
	}
	resolvedNew := sinkmodels.SinkMessage{
		Namespace:  "ns",
		Serialized: &serializer.CloudEventsKafkaPayload{Id: "new", Source: "source", Type: "usage"},
		Meters:     []*meter.Meter{affectedMeter},
	}
	resolvedOld := sinkmodels.SinkMessage{
		Namespace:  "ns",
		Serialized: &serializer.CloudEventsKafkaPayload{Id: "old", Source: "source", Type: "usage"},
		Meters:     []*meter.Meter{affectedMeter},
	}
	droppedOld := sinkmodels.SinkMessage{
		Namespace:  "ns",
		Serialized: &serializer.CloudEventsKafkaPayload{Id: "old", Source: "source", Type: "usage"},
		Status:     sinkmodels.ProcessingStatus{State: sinkmodels.DROP, DropError: errors.New("skipping non unique message")},
	}
	redisError := errors.New("redis unavailable")

	tests := []struct {
		name string
		// Includes repeated and dropped messages whose offsets must still be handled.
		inputMessages []sinkmodels.SinkMessage
		// Models identities already persisted before this batch reaches the sink.
		previouslyProcessedItems dedupe.ItemSet
		// In-batch deduplication must still work without an external deduplicator.
		deduplicatorDisabled bool
		// Returned messages eligible for ClickHouse insertion, with duplicates removed and meters resolved.
		expectedStorageBatch []sinkmodels.SinkMessage
		// The original slice after mutation, including dropped messages, used by metrics and flush callbacks.
		expectedMessagesAfterProcessing []sinkmodels.SinkMessage
		// Identities sent in the single batch lookup; empty means no lookup should occur.
		expectedDedupeLookupItems []dedupe.Item
		// Injected lookup failure that must propagate as the returned error's cause.
		deduplicatorError error
	}{
		{
			name:                            "same message multiple times in the batch",
			inputMessages:                   []sinkmodels.SinkMessage{newMessage, newMessage, newMessage},
			expectedStorageBatch:            []sinkmodels.SinkMessage{resolvedNew},
			expectedMessagesAfterProcessing: []sinkmodels.SinkMessage{resolvedNew, resolvedNew, resolvedNew},
			expectedDedupeLookupItems:       []dedupe.Item{newMessage.GetDedupeItem()},
		},
		{
			name:                            "already processed message",
			inputMessages:                   []sinkmodels.SinkMessage{oldMessage},
			previouslyProcessedItems:        dedupe.ItemSet{oldMessage.GetDedupeItem(): {}},
			expectedStorageBatch:            []sinkmodels.SinkMessage{},
			expectedMessagesAfterProcessing: []sinkmodels.SinkMessage{droppedOld},
			expectedDedupeLookupItems:       []dedupe.Item{oldMessage.GetDedupeItem()},
		},
		{
			name:                            "no duplicates",
			inputMessages:                   []sinkmodels.SinkMessage{newMessage, oldMessage},
			expectedStorageBatch:            []sinkmodels.SinkMessage{resolvedNew, resolvedOld},
			expectedMessagesAfterProcessing: []sinkmodels.SinkMessage{resolvedNew, resolvedOld},
			expectedDedupeLookupItems:       []dedupe.Item{newMessage.GetDedupeItem(), oldMessage.GetDedupeItem()},
		},
		{
			name:                            "mixed unique repeated already processed and invalid messages",
			inputMessages:                   []sinkmodels.SinkMessage{newMessage, oldMessage, newMessage, invalidMessage},
			previouslyProcessedItems:        dedupe.ItemSet{oldMessage.GetDedupeItem(): {}},
			expectedStorageBatch:            []sinkmodels.SinkMessage{resolvedNew},
			expectedMessagesAfterProcessing: []sinkmodels.SinkMessage{resolvedNew, droppedOld, resolvedNew, invalidMessage},
			expectedDedupeLookupItems:       []dedupe.Item{newMessage.GetDedupeItem(), oldMessage.GetDedupeItem()},
		},
		{
			name:                            "all messages already processed including repeated messages",
			inputMessages:                   []sinkmodels.SinkMessage{oldMessage, oldMessage},
			previouslyProcessedItems:        dedupe.ItemSet{oldMessage.GetDedupeItem(): {}},
			expectedStorageBatch:            []sinkmodels.SinkMessage{},
			expectedMessagesAfterProcessing: []sinkmodels.SinkMessage{droppedOld, droppedOld},
			expectedDedupeLookupItems:       []dedupe.Item{oldMessage.GetDedupeItem()},
		},
		{
			name:                            "disabled deduplicator still removes duplicates within the batch",
			inputMessages:                   []sinkmodels.SinkMessage{newMessage, oldMessage, newMessage},
			previouslyProcessedItems:        dedupe.ItemSet{oldMessage.GetDedupeItem(): {}},
			deduplicatorDisabled:            true,
			expectedStorageBatch:            []sinkmodels.SinkMessage{resolvedNew, resolvedOld},
			expectedMessagesAfterProcessing: []sinkmodels.SinkMessage{resolvedNew, resolvedOld, resolvedNew},
		},
		{
			name:                            "all invalid messages",
			inputMessages:                   []sinkmodels.SinkMessage{invalidMessage},
			expectedStorageBatch:            []sinkmodels.SinkMessage{},
			expectedMessagesAfterProcessing: []sinkmodels.SinkMessage{invalidMessage},
		},
		{
			name:                            "empty batch",
			inputMessages:                   []sinkmodels.SinkMessage{},
			expectedStorageBatch:            []sinkmodels.SinkMessage{},
			expectedMessagesAfterProcessing: []sinkmodels.SinkMessage{},
		},
		{
			name:                            "Redis failure leaves messages unchanged",
			inputMessages:                   []sinkmodels.SinkMessage{newMessage, oldMessage},
			expectedMessagesAfterProcessing: []sinkmodels.SinkMessage{newMessage, oldMessage},
			expectedDedupeLookupItems:       []dedupe.Item{newMessage.GetDedupeItem(), oldMessage.GetDedupeItem()},
			deduplicatorError:               redisError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a batch and the previously processed event identities
			messages := make([]sinkmodels.SinkMessage, len(tc.inputMessages))
			for i, message := range tc.inputMessages {
				messages[i] = message
				if message.Serialized != nil {
					serialized := *message.Serialized
					messages[i].Serialized = &serialized
				}
			}
			d := &batchDeduplicator{processed: tc.previouslyProcessedItems, err: tc.deduplicatorError}
			s := &Sink{
				config: SinkConfig{Deduplicator: d},
				meterCache: &NamespacedMeterCache{
					logger:     slog.New(slog.DiscardHandler),
					namespaces: map[string]MetersByType{"ns": {"usage": {affectedMeter}}},
				},
			}
			if tc.deduplicatorDisabled {
				s.config.Deduplicator = nil
			}

			// when deduplicating the batch and resolving affected meters
			batch, err := s.deduplicateAndResolveMeters(t.Context(), messages)

			// then storage output and the original callback messages match their expected states
			require.ErrorIs(t, err, tc.deduplicatorError)
			require.Equal(t, tc.expectedStorageBatch, batch)
			require.Equal(t, tc.expectedMessagesAfterProcessing, messages)
			require.Equal(t, tc.expectedDedupeLookupItems, d.items)
			if len(tc.expectedDedupeLookupItems) == 0 {
				require.Zero(t, d.calls)
			} else {
				require.Equal(t, 1, d.calls)
			}
		})
	}
}

func TestParseMessageDoesNotCheckRedis(t *testing.T) {
	d := &batchDeduplicator{}
	s := &Sink{config: SinkConfig{Deduplicator: d}}
	message, err := s.parseMessage(t.Context(), &kafka.Message{
		Headers: []kafka.Header{{Key: kafkaingest.HeaderKeyNamespace, Value: []byte("ns")}},
		Value:   []byte(`{"id":"id","source":"source","subject":"subject","type":"usage","time":1}`),
	})
	require.NoError(t, err)
	require.Equal(t, sinkmodels.OK, message.Status.State)
	require.NotNil(t, message.Serialized)
	require.Zero(t, d.calls)
}
