package outbox

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
)

func TestDrainBudgetCommitsSuccessfulPrefix(t *testing.T) {
	// Given one transaction whose second send consumes the drain budget.
	p, client, raw := newManualTestPublisher(t)
	p.cfg.DrainTimeout = time.Second
	enqueueTransaction(t, p, "A", "B", "C")
	// Remove enqueue notifications so only the drain can request continuation.
	for len(p.wake) > 0 {
		<-p.wake
	}
	raw.setOnSend(func(msg publishedMessage) error {
		if msg.id == "B" {
			time.Sleep(p.cfg.DrainTimeout)
		}
		return nil
	})

	// When the budget expires during B, its acknowledgment still gets committed.
	require.NoError(t, p.drain(t.Context()))
	require.Equal(t, []string{"A", "B"}, raw.deliveredIDs())
	require.Empty(t, raw.attemptsFor("C"))
	row, err := client.EventOutbox.Query().Only(t.Context())
	require.NoError(t, err)
	require.Equal(t, "C", row.MessageID)
	require.NotEmpty(t, p.wake, "productive partial batches must schedule continuation")

	// Then the next drain resumes at C without resending the committed prefix.
	raw.setOnSend(nil)
	require.NoError(t, p.drain(t.Context()))
	require.Equal(t, []string{"A", "B", "C"}, raw.deliveredIDs())
}

func TestDrainBudgetPersistsLateFailure(t *testing.T) {
	// Given a successful A followed by a B failure after the budget expires.
	p, client, raw := newManualTestPublisher(t)
	p.cfg.DrainTimeout = time.Second
	enqueueTransaction(t, p, "A", "B", "C")
	brokerErr := errors.New("slow broker failure")
	raw.setOnSend(func(msg publishedMessage) error {
		if msg.id == "B" {
			time.Sleep(p.cfg.DrainTimeout)
			return brokerErr
		}
		return nil
	})

	// When the send returns, bookkeeping uses the still-live application context.
	require.ErrorIs(t, p.drain(t.Context()), brokerErr)
	rows, err := client.EventOutbox.Query().Order(eventoutbox.ByID()).All(t.Context())
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "B", rows[0].MessageID)
	require.Equal(t, 1, rows[0].Attempts)
	require.Equal(t, "C", rows[1].MessageID)
	require.Zero(t, rows[1].Attempts)
	require.Empty(t, raw.attemptsFor("C"))

	// Then a later drain retries only the remaining suffix.
	raw.setOnSend(nil)
	require.NoError(t, p.drain(t.Context()))
	require.Equal(t, []string{"A", "B", "C"}, raw.deliveredIDs())
}

func TestWorkerBudgetAllowsRetryExhaustion(t *testing.T) {
	// Given a real worker whose first send fails after its one-second budget.
	fixture, client, raw := newManualTestPublisher(t)
	cfg := fixture.cfg
	cfg.DrainTimeout = time.Second
	cfg.DrainConcurrency = 1
	cfg.MaxAttempts = 1
	raw.setOnSend(func(msg publishedMessage) error {
		if msg.id == "A" {
			time.Sleep(cfg.DrainTimeout)
			return errors.New("slow broker failure")
		}
		return nil
	})
	p, err := NewPublisher(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, p.Close()) })

	// When run invokes drain, it must not impose a canceling transaction deadline.
	enqueueTransaction(t, p, "A", "B")
	require.Eventually(t, func() bool { return len(raw.deliveredIDs()) == 1 }, 5*time.Second, 20*time.Millisecond)
	eventuallyRowCount(t, client, 1)

	// Then A reaches its attempt limit and B progresses without waiting for the minute tick.
	row, err := client.EventOutbox.Query().Only(t.Context())
	require.NoError(t, err)
	require.Equal(t, "A", row.MessageID)
	require.Equal(t, 1, row.Attempts)
	require.Len(t, raw.attemptsFor("A"), 1)
	require.Equal(t, []string{"B"}, raw.deliveredIDs())
}
