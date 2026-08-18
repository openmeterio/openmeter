package sink

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/testutils"
)

type hasEventStreamingStub struct {
	*testutils.MockStreamingConnector
	err    error
	calls  []streaming.ListEventsV2Params
	exists map[dedupe.Item]struct{}
}

func newHasEventStreamingStub(t testing.TB) *hasEventStreamingStub {
	return &hasEventStreamingStub{
		MockStreamingConnector: testutils.NewMockStreamingConnector(t),
		exists:                 map[dedupe.Item]struct{}{},
	}
}

func (s *hasEventStreamingStub) ListEventsV2(_ context.Context, params streaming.ListEventsV2Params) ([]streaming.RawEvent, error) {
	s.calls = append(s.calls, params)

	if s.err != nil {
		return nil, s.err
	}

	source := lo.Empty[string]()
	if params.Source != nil && params.Source.Eq != nil {
		source = *params.Source.Eq
	}

	events := []streaming.RawEvent{}
	ids := lo.FromPtr(params.ID.In)
	sort.Strings(ids)

	for _, id := range ids {
		item := dedupe.Item{
			Namespace: params.Namespace,
			Source:    source,
			ID:        id,
		}
		if _, ok := s.exists[item]; !ok {
			continue
		}
		events = append(events, streaming.RawEvent{
			Namespace:  params.Namespace,
			ID:         id,
			Source:     source,
			Time:       time.Now(),
			StoreRowID: id + "-row",
		})
	}

	return events, nil
}

func TestClickHouseStorageHasEvents_BatchesByNamespaceAndSource(t *testing.T) {
	stub := newHasEventStreamingStub(t)
	storage := &ClickHouseStorage{
		config: ClickHouseStorageConfig{
			Streaming: stub,
		},
	}

	items := []dedupe.Item{
		{Namespace: "tenant-b", Source: "gateway", ID: "evt-1"},
		{Namespace: "tenant-b", Source: "gateway", ID: "evt-2"},
		{Namespace: "tenant-b", Source: "worker", ID: "evt-3"},
		{Namespace: "tenant-c", Source: "gateway", ID: "evt-4"},
	}

	stub.exists[items[0]] = struct{}{}
	stub.exists[items[2]] = struct{}{}

	found, err := storage.HasEvents(t.Context(), items)
	require.NoError(t, err)
	require.Equal(t, dedupe.ItemSet{
		items[0]: {},
		items[2]: {},
	}, found)

	require.Len(t, stub.calls, 3)
	for _, call := range stub.calls {
		require.NotNil(t, call.ID)
		require.NotNil(t, call.ID.In)
		require.NotNil(t, call.Source)
		require.NotNil(t, call.Source.Eq)
		require.NotNil(t, call.Limit)
		require.LessOrEqual(t, len(*call.ID.In), existsQueryBatchSize)
	}
}
