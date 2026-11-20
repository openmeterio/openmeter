package outbox

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func TestScopedDrainPreservesBusinessEventsAndTransactionOrder(t *testing.T) {
	// Given one independent business event and a transaction with a business
	// event between two balance updates, all persisted on the same topic.
	p, client, raw := newManualTestPublisher(t)
	p.cfg.DrainEventType = "balance.updated"
	enqueueTransaction(t, p, "independent-business")

	require.NoError(t, transaction.RunWithNoValue(t.Context(), enttx.NewCreator(client), func(ctx context.Context) error {
		for _, id := range []string{"first-update", "mixed-business", "last-update"} {
			msg := testMessage(ctx, id, "shared")
			if id != "mixed-business" {
				msg.Metadata.Set(marshaler.CloudEventsHeaderType, p.cfg.DrainEventType)
			}

			if err := p.Publish(testTopic, msg); err != nil {
				return err
			}
		}

		return nil
	}))

	update := testMessage(t.Context(), "standalone-update", "shared")
	update.Metadata.Set(marshaler.CloudEventsHeaderType, p.cfg.DrainEventType)
	require.NoError(t, p.Publish(testTopic, update))
	require.NoError(t, client.EventOutbox.Update().Where(eventoutbox.MessageIDEQ("mixed-business")).SetAttempts(1).Exec(t.Context()))

	raw.setOnSend(func(m publishedMessage) error {
		if m.metadata[marshaler.CloudEventsHeaderType] != p.cfg.DrainEventType {
			return errors.New("trial publisher cannot send business events")
		}

		return nil
	})

	// When the trial worker drains with a lower retry limit, it still cannot skip
	// the mixed business row as if the normal worker had exhausted its retries.
	p.cfg.MaxAttempts = 1
	require.NoError(t, p.drain(t.Context()))
	require.NoError(t, p.drain(t.Context()))
	require.Equal(t, []string{"standalone-update"}, raw.deliveredIDs())

	// Then business rows keep their delivery attempts and later updates wait for them.
	rows, err := client.EventOutbox.Query().Order(eventoutbox.ByID()).All(t.Context())
	require.NoError(t, err)
	require.Len(t, rows, 4)
	for i, expected := range []struct {
		id       string
		attempts int
	}{{"independent-business", 0}, {"first-update", 0}, {"mixed-business", 1}, {"last-update", 0}} {
		require.Equal(t, expected.id, rows[i].MessageID)
		require.Equal(t, expected.attempts, rows[i].Attempts)
		require.Empty(t, raw.attemptsFor(expected.id))
	}

	// The normal worker can deliver the mixed transaction in order.
	p.cfg.DrainEventType = ""
	p.cfg.MaxAttempts = 10
	raw.setOnSend(nil)
	require.NoError(t, p.drain(t.Context()))
	require.Equal(t, []string{"standalone-update", "independent-business", "first-update", "mixed-business", "last-update"}, raw.deliveredIDs())
	eventuallyRowCount(t, client, 0)
}
