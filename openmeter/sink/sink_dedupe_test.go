package sink

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/openmeterio/openmeter/openmeter/dedupe/memorydedupe"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
)

type durableProbeStorage struct {
	persisted map[dedupe.Item]struct{}
	inserts   int
}

func newDurableProbeStorage() *durableProbeStorage {
	return &durableProbeStorage{
		persisted: map[dedupe.Item]struct{}{},
	}
}

func (s *durableProbeStorage) BatchInsert(_ context.Context, messages []sinkmodels.SinkMessage) error {
	for _, message := range messages {
		s.persisted[message.GetDedupeItem()] = struct{}{}
	}

	s.inserts += len(messages)

	return nil
}

func (s *durableProbeStorage) HasEvent(_ context.Context, message sinkmodels.SinkMessage) (bool, error) {
	_, ok := s.persisted[message.GetDedupeItem()]
	return ok, nil
}

func TestFilterMessagesForInsert_ReplaysCrashBeforeDurability(t *testing.T) {
	deduplicator, err := memorydedupe.NewDeduplicator(64)
	require.NoError(t, err)

	storage := newDurableProbeStorage()
	s := &Sink{
		config: SinkConfig{
			Deduplicator: deduplicator,
			Storage:      storage,
		},
	}

	ctx := t.Context()
	message := testSinkMessage("tenant-a", "evt-1", "gateway")

	// First delivery reserves the dedupe key and is selected for insert.
	firstAttempt, err := s.filterMessagesForInsert(ctx, []sinkmodels.SinkMessage{message})
	require.NoError(t, err)
	require.Len(t, firstAttempt, 1)

	// Crash before durable insert: key exists in dedupe, storage still has no row.
	replayAfterCrash, err := s.filterMessagesForInsert(ctx, []sinkmodels.SinkMessage{message})
	require.NoError(t, err)
	require.Len(t, replayAfterCrash, 1)

	// Replay inserts the row exactly once.
	require.NoError(t, storage.BatchInsert(ctx, replayAfterCrash))
	require.Equal(t, 1, storage.inserts)

	// Crash after durable insert: replay should be filtered out.
	replayAfterDurableInsert, err := s.filterMessagesForInsert(ctx, []sinkmodels.SinkMessage{message})
	require.NoError(t, err)
	require.Empty(t, replayAfterDurableInsert)
	require.Equal(t, 1, storage.inserts)
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
