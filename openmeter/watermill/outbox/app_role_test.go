package outbox

import (
	"context"
	"errors"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func TestDeploymentQueuesOwnDeliveryAndRetries(t *testing.T) {
	// Given two roles sharing a topic and even a source transaction.
	backend, db, raw := newManualTestPublisher(t)
	cfg := backend.cfg
	cfg.AppRole = "worker"
	cfg.MaxAttempts = 2
	worker := &Publisher{cfg: cfg, wake: make(chan struct{}, 1)}

	// Older writers omit the role. The database assigns their rows to the backend.
	_, err := db.EventOutbox.Create().
		SetMessageID("legacy").
		SetTransactionID(ulid.Make().String()).
		SetTopic(testTopic).
		SetPayload([]byte("legacy")).
		SetMetadata(map[string]string{}).
		Save(t.Context())
	require.NoError(t, err)

	require.NoError(t, transaction.RunWithNoValue(t.Context(), enttx.NewCreator(db), func(ctx context.Context) error {
		require.NoError(t, backend.Publish(testTopic, testMessage(ctx, "backend", "shared")))
		return worker.Publish(testTopic, testMessage(ctx, "first", "shared"), testMessage(ctx, "second", "shared"))
	}))

	// When delivery fails, the worker records its attempt and leaves the sibling pending.
	sendErr := errors.New("broker unavailable")
	raw.setOnSend(func(publishedMessage) error { return sendErr })
	require.ErrorIs(t, worker.drain(t.Context()), sendErr)
	row, err := db.EventOutbox.Query().Where(eventoutbox.MessageIDEQ("first")).Only(t.Context())
	require.NoError(t, err)
	require.Equal(t, "worker", row.AppRole)
	require.Equal(t, 1, row.Attempts)
	require.Empty(t, raw.attemptsFor("second"))
	require.Empty(t, raw.attemptsFor("backend"))

	// Then the default role drains its own row. A worker replica retries both siblings in order.
	raw.setOnSend(nil)
	require.NoError(t, backend.drain(t.Context()))
	require.Equal(t, []string{"legacy", "backend"}, raw.deliveredIDs())

	replica := &Publisher{cfg: cfg, wake: make(chan struct{}, 1)}
	require.NoError(t, replica.drain(t.Context()))
	require.Equal(t, []string{"legacy", "backend", "first", "second"}, raw.deliveredIDs())
	eventuallyRowCount(t, db, 0)
}
