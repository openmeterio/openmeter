package sink

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/testutils"
)

type hasEventStreamingStub struct {
	*testutils.MockStreamingConnector
	events     []streaming.RawEvent
	err        error
	lastParams *streaming.ListEventsV2Params
}

func newHasEventStreamingStub(t testing.TB) *hasEventStreamingStub {
	return &hasEventStreamingStub{
		MockStreamingConnector: testutils.NewMockStreamingConnector(t),
	}
}

func (s *hasEventStreamingStub) ListEventsV2(_ context.Context, params streaming.ListEventsV2Params) ([]streaming.RawEvent, error) {
	s.lastParams = &params

	if s.err != nil {
		return nil, s.err
	}

	return s.events, nil
}

func TestClickHouseStorageHasEvent(t *testing.T) {
	stub := newHasEventStreamingStub(t)
	storage := &ClickHouseStorage{
		config: ClickHouseStorageConfig{
			Streaming: stub,
		},
	}

	message := testSinkMessage("tenant-b", "evt-2", "gateway")

	found, err := storage.HasEvent(t.Context(), message)
	require.NoError(t, err)
	require.False(t, found)
	require.NotNil(t, stub.lastParams)
	require.Equal(t, "tenant-b", stub.lastParams.Namespace)
	require.NotNil(t, stub.lastParams.ID)
	require.NotNil(t, stub.lastParams.ID.Eq)
	require.Equal(t, "evt-2", *stub.lastParams.ID.Eq)
	require.NotNil(t, stub.lastParams.Source)
	require.NotNil(t, stub.lastParams.Source.Eq)
	require.Equal(t, "gateway", *stub.lastParams.Source.Eq)
	require.NotNil(t, stub.lastParams.Limit)
	require.Equal(t, 1, *stub.lastParams.Limit)

	stub.events = []streaming.RawEvent{
		{
			Namespace: "tenant-b",
			ID:        "evt-2",
			Source:    "gateway",
		},
	}

	found, err = storage.HasEvent(t.Context(), message)
	require.NoError(t, err)
	require.True(t, found)
}
