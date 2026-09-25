package outbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

const (
	testTopic  = "system-events"
	drainLimit = 100
)

type publishedMessage struct {
	topic    string
	id       string
	payload  []byte
	metadata message.Metadata
}

type recordingPublisher struct {
	mu        sync.Mutex
	attempts  []publishedMessage
	delivered []publishedMessage
	next      chan publishedMessage
	onSend    func(publishedMessage) error
	closed    bool
}

func newRecordingPublisher() *recordingPublisher {
	return &recordingPublisher{next: make(chan publishedMessage, 4*drainLimit)}
}

func (p *recordingPublisher) Publish(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		attempt := publishedMessage{
			topic:    topic,
			id:       msg.UUID,
			payload:  append([]byte(nil), msg.Payload...),
			metadata: message.Metadata{},
		}
		for key, value := range msg.Metadata {
			attempt.metadata[key] = value
		}
		p.mu.Lock()
		p.attempts = append(p.attempts, attempt)
		onSend := p.onSend
		p.mu.Unlock()
		p.next <- attempt
		if onSend != nil {
			if err := onSend(attempt); err != nil {
				return err
			}
		}
		p.mu.Lock()
		p.delivered = append(p.delivered, attempt)
		p.mu.Unlock()
	}
	return nil
}

func (p *recordingPublisher) setOnSend(onSend func(publishedMessage) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onSend = onSend
}

func (p *recordingPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

func (p *recordingPublisher) attemptsFor(id string) []publishedMessage {
	p.mu.Lock()
	defer p.mu.Unlock()
	var matches []publishedMessage
	for _, attempt := range p.attempts {
		if attempt.id == id {
			matches = append(matches, attempt)
		}
	}
	return matches
}

func (p *recordingPublisher) deliveredIDs() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	ids := make([]string, len(p.delivered))
	for i, delivered := range p.delivered {
		ids[i] = delivered.id
	}
	return ids
}

func newTestPublisher(t *testing.T, raw *recordingPublisher) (*Publisher, *db.Client) {
	t.Helper()
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	client := testDB.EntDriver.Client()
	t.Cleanup(func() {
		_ = client.Close()
		testDB.Close(t)
	})
	p, err := NewPublisher(t.Context(), Config{
		DB:               client,
		Publisher:        raw,
		Topic:            testTopic,
		Logger:           testutils.NewDiscardLogger(t),
		DrainLimit:       drainLimit,
		DrainTimeout:     30 * time.Second,
		DrainConcurrency: 2,
		RetryInterval:    time.Minute,
		MaxAttempts:      10,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, p.Close()) })
	return p, client
}

func testMessage(ctx context.Context, id, messageKey string) *message.Message {
	msg := message.NewMessage(id, []byte("payload-"+id))
	msg.Metadata.Set("ce_type", "test.created")
	if messageKey != "" {
		msg.Metadata.Set("x-kafka-partition-key", messageKey)
	}
	msg.SetContext(ctx)
	return msg
}

func nextAttempt(t *testing.T, raw *recordingPublisher) publishedMessage {
	t.Helper()
	select {
	case attempt := <-raw.next:
		return attempt
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for broker publication")
		return publishedMessage{}
	}
}

func noAttempt(t *testing.T, raw *recordingPublisher) {
	t.Helper()
	select {
	case attempt := <-raw.next:
		t.Fatalf("unexpected broker publication: %s", attempt.id)
	case <-time.After(100 * time.Millisecond):
	}
}

func eventuallyRowCount(t *testing.T, client *db.Client, want int) {
	t.Helper()
	require.Eventually(t, func() bool {
		count, err := client.EventOutbox.Query().Count(t.Context())
		return err == nil && count == want
	}, 5*time.Second, 20*time.Millisecond)
}

func TestPublisherDeliversOnlyAfterOuterCommit(t *testing.T) {
	raw := newRecordingPublisher()
	p, client := newTestPublisher(t, raw)
	creator := enttx.NewCreator(client)

	err := transaction.RunWithNoValue(t.Context(), creator, func(ctx context.Context) error {
		require.NoError(t, p.Publish(testTopic, testMessage(ctx, "outer", "customer-1")))
		require.NoError(t, transaction.RunWithNoValue(ctx, creator, func(nested context.Context) error {
			return p.Publish(testTopic, testMessage(nested, "nested", "customer-1"))
		}))
		count, err := client.EventOutbox.Query().Count(t.Context())
		require.NoError(t, err)
		require.Zero(t, count, "other connections must not see uncommitted outbox rows")
		noAttempt(t, raw)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"outer", "nested"}, []string{nextAttempt(t, raw).id, nextAttempt(t, raw).id})
	eventuallyRowCount(t, client, 0)
}

func TestPublisherDiscardsRolledBackEvents(t *testing.T) {
	raw := newRecordingPublisher()
	p, client := newTestPublisher(t, raw)
	creator := enttx.NewCreator(client)
	rollback := errors.New("rollback")

	err := transaction.RunWithNoValue(t.Context(), creator, func(ctx context.Context) error {
		require.NoError(t, p.Publish(testTopic, testMessage(ctx, "outer-rolled-back", "a")))
		return rollback
	})
	require.ErrorIs(t, err, rollback)
	eventuallyRowCount(t, client, 0)
	noAttempt(t, raw)

	err = transaction.RunWithNoValue(t.Context(), creator, func(ctx context.Context) error {
		require.NoError(t, p.Publish(testTopic, testMessage(ctx, "retained", "a")))
		err := transaction.RunWithNoValue(ctx, creator, func(nested context.Context) error {
			require.NoError(t, p.Publish(testTopic, testMessage(nested, "inner-rolled-back", "a")))
			return rollback
		})
		require.ErrorIs(t, err, rollback)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, "retained", nextAttempt(t, raw).id)
	noAttempt(t, raw)
	eventuallyRowCount(t, client, 0)
}

func TestPublisherPreservesEnvelopeAndBypassesNonSystemTopic(t *testing.T) {
	raw := newRecordingPublisher()
	p, client := newTestPublisher(t, raw)
	msg := testMessage(t.Context(), "standalone", "customer-1")
	require.NoError(t, p.Publish(testTopic, msg))
	attempt := nextAttempt(t, raw)
	require.Equal(t, testTopic, attempt.topic)
	require.Equal(t, msg.UUID, attempt.id)
	require.Equal(t, []byte(msg.Payload), attempt.payload)
	require.Equal(t, msg.Metadata, attempt.metadata)
	eventuallyRowCount(t, client, 0)

	raw.setOnSend(func(publishedMessage) error { return errors.New("broker failed") })
	err := p.Publish("ingest-events", testMessage(t.Context(), "bypass", ""))
	require.ErrorContains(t, err, "broker failed")
	require.Equal(t, "bypass", nextAttempt(t, raw).id)
	eventuallyRowCount(t, client, 0)

	require.NoError(t, p.Close())
	require.Error(t, p.Publish(testTopic, testMessage(t.Context(), "after-close", "")))
	raw.mu.Lock()
	require.False(t, raw.closed, "outbox does not own the broker publisher")
	raw.mu.Unlock()
}

func TestPublisherFailureDoesNotBlockOtherRows(t *testing.T) {
	raw := newRecordingPublisher()
	raw.setOnSend(func(attempt publishedMessage) error {
		if attempt.id == "failed" {
			return errors.New("broker unavailable for this key")
		}
		return nil
	})
	p, client := newTestPublisher(t, raw)
	require.NoError(t, p.Publish(testTopic, testMessage(t.Context(), "failed", "key-a")))
	require.Equal(t, "failed", nextAttempt(t, raw).id)
	eventuallyRowCount(t, client, 1)

	require.NoError(t, p.Publish(testTopic, testMessage(t.Context(), "healthy", "key-a")))
	require.Eventually(t, func() bool { return len(raw.attemptsFor("healthy")) == 1 }, 5*time.Second, 20*time.Millisecond)
	require.Equal(t, 1, len(raw.attemptsFor("healthy")))
	eventuallyRowCount(t, client, 1)
	require.Eventually(t, func() bool { return len(raw.attemptsFor("failed")) >= 2 }, 5*time.Second, 20*time.Millisecond,
		"next publish should retry pending rows")
}

func TestPublisherResendsSameMessageAfterDeleteFailure(t *testing.T) {
	raw := newRecordingPublisher()
	p, client := newTestPublisher(t, raw)
	// Stop the background drain so the injected failure can be exercised
	// deterministically through the same drain method.
	p.cancel()
	p.workers.Wait()
	msg := testMessage(t.Context(), "delete-fails-once", "customer-1")
	require.NoError(t, p.Publish(testTopic, msg))
	eventuallyRowCount(t, client, 1)

	_, err := client.ExecContext(t.Context(), `
		CREATE FUNCTION fail_outbox_delete() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'injected outbox delete failure';
		END;
		$$`)
	require.NoError(t, err)
	_, err = client.ExecContext(t.Context(), `
		CREATE TRIGGER fail_outbox_delete BEFORE DELETE ON event_outboxes
		FOR EACH ROW EXECUTE FUNCTION fail_outbox_delete()`)
	require.NoError(t, err)

	err = p.drain(t.Context())
	require.ErrorContains(t, err, "injected outbox delete failure")
	first := nextAttempt(t, raw)
	eventuallyRowCount(t, client, 1)
	_, err = client.ExecContext(t.Context(), "DROP TRIGGER fail_outbox_delete ON event_outboxes")
	require.NoError(t, err)

	require.NoError(t, p.drain(t.Context()))
	second := nextAttempt(t, raw)
	require.Equal(t, first, second)
	eventuallyRowCount(t, client, 0)
}

func TestConcurrentEnqueueDoesNotSerializeMessageKey(t *testing.T) {
	// Given an uncommitted event for a Kafka key.
	raw := newRecordingPublisher()
	p, client := newTestPublisher(t, raw)
	creator := enttx.NewCreator(client)
	err := transaction.RunWithNoValue(t.Context(), creator, func(ctx context.Context) error {
		require.NoError(t, p.Publish(testTopic, testMessage(ctx, "first", "customer-1")))

		// When a separate transaction publishes the same key, it commits and
		// delivers while the first transaction is still open.
		independent, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		require.NoError(t, p.Publish(testTopic, testMessage(independent, "second", "customer-1")))
		require.Equal(t, "second", nextAttempt(t, raw).id)
		noAttempt(t, raw)
		return nil
	})

	// Then the first event becomes eligible only after its own commit.
	require.NoError(t, err)
	require.Equal(t, "first", nextAttempt(t, raw).id)
	eventuallyRowCount(t, client, 0)
}

func TestConcurrentPublishersCanSendSameKeyIndependently(t *testing.T) {
	// Given two relay instances and a blocked broker send holding one row lock.
	raw := newRecordingPublisher()
	releaseFirst := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseFirst) }) }
	defer release()
	raw.setOnSend(func(attempt publishedMessage) error {
		if attempt.id == "first" {
			<-releaseFirst
		}
		return nil
	})
	p1, client := newTestPublisher(t, raw)
	p2, err := NewPublisher(t.Context(), Config{
		DB:               client,
		Publisher:        raw,
		Topic:            testTopic,
		Logger:           testutils.NewDiscardLogger(t),
		DrainLimit:       drainLimit,
		DrainTimeout:     30 * time.Second,
		DrainConcurrency: 2,
		RetryInterval:    time.Minute,
		MaxAttempts:      10,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, p2.Close()) })

	require.NoError(t, p1.Publish(testTopic, testMessage(t.Context(), "first", "customer-1")))
	require.Equal(t, "first", nextAttempt(t, raw).id)
	// When another relay publishes the same key, its row can progress independently.
	require.NoError(t, p2.Publish(testTopic, testMessage(t.Context(), "second", "customer-1")))
	require.Equal(t, "second", nextAttempt(t, raw).id)
	require.Len(t, raw.attemptsFor("first"), 1)

	// Then the first send completes once its broker call is released.
	release()
	eventuallyRowCount(t, client, 0)
	require.ElementsMatch(t, []string{"first", "second"}, raw.deliveredIDs())
}

func TestPublisherKeepsLargeTransactionTogether(t *testing.T) {
	// Given a source transaction larger than the drain message budget.
	raw := newRecordingPublisher()
	p, client := newTestPublisher(t, raw)
	const eventCount = 2*drainLimit + 1
	creator := enttx.NewCreator(client)
	err := transaction.RunWithNoValue(t.Context(), creator, func(ctx context.Context) error {
		messages := make([]*message.Message, eventCount)
		for i := range messages {
			messages[i] = testMessage(ctx, fmt.Sprintf("event-%03d", i), "customer-1")
		}
		return p.Publish(testTopic, messages...)
	})
	require.NoError(t, err)

	// When the commit wakes the drainer, it finishes the entire transaction.
	// Then every event is delivered in order without another publish.
	require.Eventually(t, func() bool { return len(raw.deliveredIDs()) == eventCount }, 20*time.Second, 20*time.Millisecond)
	eventuallyRowCount(t, client, 0)
	expected := make([]string, eventCount)
	for i := range expected {
		expected[i] = fmt.Sprintf("event-%03d", i)
	}
	require.Equal(t, expected, raw.deliveredIDs())
}

var _ message.Publisher = (*recordingPublisher)(nil)
