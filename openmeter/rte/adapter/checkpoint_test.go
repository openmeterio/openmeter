package adapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func testCheckpointStorage[K rte.Key, V any](t *testing.T, writer, reader rte.Store[K, V], key K, value V) {
	t.Helper()

	ctx := t.Context()
	at := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)

	// Given a balance that has not been calculated, the first attempt has no checkpoint.
	got, err := reader.GetCheckpoints(ctx, key)
	require.NoError(t, err)
	require.Empty(t, got)

	// When the later calculation finishes first, its state and both checkpoints are retained.
	history := rte.Checkpoints{
		{At: at, Data: json.RawMessage(`"later"`)},
		{At: at.Add(-time.Hour), Data: json.RawMessage(`"earlier"`)},
	}
	winner := rte.StoreInput[K, V]{Key: key, State: rte.State[V]{CalculatedAt: at, Value: value}, Checkpoints: &history}
	result, err := writer.Store(ctx, winner)
	require.NoError(t, err)
	require.True(t, result.Updated)

	// Reusing the existing timeline writes only the new public result.
	reused := winner
	reused.State.CalculatedAt = at.Add(time.Second)
	reused.Checkpoints = nil
	result, err = writer.Store(ctx, reused)
	require.NoError(t, err)
	require.True(t, result.Updated)

	late := winner
	late.State.CalculatedAt = at.Add(-time.Minute)
	late.Checkpoints = &rte.Checkpoints{}
	result, err = writer.Store(ctx, late)
	require.NoError(t, err)
	require.False(t, result.Updated)

	got, err = reader.GetCheckpoints(ctx, key)
	require.NoError(t, err)
	require.Equal(t, history, got)

	// A failure after storage (for example, when writing the outbox) rolls back the timeline too.
	late.State.CalculatedAt = at.Add(time.Minute)
	rollback := errors.New("abort calculation")
	err = transaction.RunWithNoValue(ctx, writer, func(ctx context.Context) error {
		if _, err := writer.Store(ctx, late); err != nil {
			return err
		}

		return rollback
	})
	require.ErrorIs(t, err, rollback)

	got, err = reader.GetCheckpoints(ctx, key)
	require.NoError(t, err)
	require.Equal(t, history, got)

	// A newer refresh that discards its last checkpoint clears the persisted history.
	result, err = writer.Store(ctx, late)
	require.NoError(t, err)
	require.True(t, result.Updated)

	got, err = reader.GetCheckpoints(ctx, key)
	require.NoError(t, err)
	require.Empty(t, got)
}
