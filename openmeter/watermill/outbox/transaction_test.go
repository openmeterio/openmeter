package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// Manual drains make retry boundaries deterministic without changing the delivery path.
func newManualTestPublisher(t *testing.T) (*Publisher, *db.Client, *recordingPublisher) {
	t.Helper()
	raw := newRecordingPublisher()
	p, client := newTestPublisher(t, raw)
	p.cancel()
	p.workers.Wait()
	return p, client, raw
}

func enqueueTransaction(t *testing.T, p *Publisher, ids ...string) {
	t.Helper()
	require.NoError(t, transaction.RunWithNoValue(t.Context(), enttx.NewCreator(p.cfg.DB), func(ctx context.Context) error {
		messages := make([]*message.Message, len(ids))
		for i, id := range ids {
			messages[i] = testMessage(ctx, id, "same-key")
		}
		return p.Publish(testTopic, messages...)
	}))
}

func TestPublisherPersistsOuterTransactionIdentity(t *testing.T) {
	// Given nested scopes, including a rolled-back savepoint.
	p, client, _ := newManualTestPublisher(t)
	creator := enttx.NewCreator(client)
	var transactionID string
	rollback := errors.New("rollback nested scope")
	require.NoError(t, transaction.RunWithNoValue(t.Context(), creator, func(ctx context.Context) error {
		tx, err := entutils.GetDriverFromContext(ctx)
		require.NoError(t, err)
		transactionID = tx.ID()
		require.NoError(t, p.Publish(testTopic, testMessage(ctx, "outer", "key")))
		require.NoError(t, transaction.RunWithNoValue(ctx, creator, func(ctx context.Context) error {
			return p.Publish(testTopic, testMessage(ctx, "nested", "key"))
		}))
		err = transaction.RunWithNoValue(ctx, creator, func(ctx context.Context) error {
			require.NoError(t, p.Publish(testTopic, testMessage(ctx, "rolled-back", "key")))
			return rollback
		})
		require.ErrorIs(t, err, rollback)
		return p.Publish(testTopic, testMessage(ctx, "after-rollback", "key"))
	}))

	// When an independent transaction enqueues another event.
	enqueueTransaction(t, p, "independent")

	// Then nested scopes share the root identity, and rollback leaves no event.
	rows, err := client.EventOutbox.Query().Order(eventoutbox.ByID()).All(t.Context())
	require.NoError(t, err)
	require.Len(t, rows, 4)
	require.NotEmpty(t, transactionID)
	for i, id := range []string{"outer", "nested", "after-rollback"} {
		require.Equal(t, id, rows[i].MessageID)
		require.Equal(t, transactionID, rows[i].TransactionID)
	}
	require.NotEqual(t, transactionID, rows[3].TransactionID)
}

func TestPublisherRetriesOnlyUnsentTransactionSuffix(t *testing.T) {
	// Given A/B/C in one source transaction and D in another.
	p, client, raw := newManualTestPublisher(t)
	enqueueTransaction(t, p, "A", "B", "C")
	enqueueTransaction(t, p, "D")
	brokerErr := errors.New("B failed")
	raw.setOnSend(func(msg publishedMessage) error {
		if msg.id == "B" {
			return brokerErr
		}
		return nil
	})

	// When B fails, the successful prefix and its failure count are committed.
	require.ErrorIs(t, p.drain(t.Context()), brokerErr)
	require.Equal(t, []string{"A", "D"}, raw.deliveredIDs())
	require.Empty(t, raw.attemptsFor("C"))
	rows, err := client.EventOutbox.Query().Order(eventoutbox.ByID()).All(t.Context())
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "B", rows[0].MessageID)
	require.Equal(t, 1, rows[0].Attempts)
	require.Equal(t, "C", rows[1].MessageID)
	require.Zero(t, rows[1].Attempts)

	// Then a later drain resumes at B, without resending A.
	raw.setOnSend(nil)
	require.NoError(t, p.drain(t.Context()))
	require.Equal(t, []string{"A", "D", "B", "C"}, raw.deliveredIDs())
	count, err := client.EventOutbox.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, count, "acknowledged rows are hard-deleted")
}

func TestPublisherAbandonsExhaustedEventAndContinuesTransaction(t *testing.T) {
	// Given a two-attempt limit and an event that always fails.
	p, client, raw := newManualTestPublisher(t)
	p.cfg.MaxAttempts = 2
	enqueueTransaction(t, p, "A", "B", "C")
	brokerErr := errors.New("B failed")
	raw.setOnSend(func(msg publishedMessage) error {
		if msg.id == "B" {
			return brokerErr
		}
		return nil
	})

	// When the first attempt fails, C waits. The second failure releases C.
	require.ErrorIs(t, p.drain(t.Context()), brokerErr)
	require.Equal(t, []string{"A"}, raw.deliveredIDs())
	require.Empty(t, raw.attemptsFor("C"))
	require.ErrorIs(t, p.drain(t.Context()), brokerErr)
	require.Equal(t, []string{"A", "C"}, raw.deliveredIDs())

	// Then only B remains for inspection, and future drains skip it.
	row, err := client.EventOutbox.Query().Only(t.Context())
	require.NoError(t, err)
	require.Equal(t, "B", row.MessageID)
	require.Equal(t, 2, row.Attempts)
	require.NoError(t, p.drain(t.Context()))
	require.Len(t, raw.attemptsFor("B"), 2)
}

func TestConcurrentPublishersClaimWholeTransactions(t *testing.T) {
	// Given two relay instances and A/B/C in the same source transaction.
	p1, client, raw := newManualTestPublisher(t)
	p2, err := NewPublisher(t.Context(), p1.cfg)
	require.NoError(t, err)
	p2.cancel()
	p2.workers.Wait()
	t.Cleanup(func() { require.NoError(t, p2.Close()) })
	enqueueTransaction(t, p1, "A", "B", "C")
	enqueueTransaction(t, p1, "D")
	releaseFirst := make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(releaseFirst) }) }
	defer release()
	raw.setOnSend(func(msg publishedMessage) error {
		if msg.id == "A" {
			<-releaseFirst
		}
		return nil
	})

	// When one relay is sending A, another may claim D but not B or C.
	done := make(chan error, 1)
	go func() { done <- p1.drain(t.Context()) }()
	require.Equal(t, "A", nextAttempt(t, raw).id)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	require.NoError(t, p2.drain(ctx))
	require.Equal(t, "D", nextAttempt(t, raw).id)
	require.Empty(t, raw.attemptsFor("B"))
	require.Empty(t, raw.attemptsFor("C"))

	// Then releasing A delivers its siblings in order, each exactly once.
	release()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("transaction drain did not finish")
	}
	require.Equal(t, []string{"D", "A", "B", "C"}, raw.deliveredIDs())
	count, err := client.EventOutbox.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, count)
}
