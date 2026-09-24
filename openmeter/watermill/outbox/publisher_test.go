package outbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

const testTopic = "system-events"

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
		DB:        client,
		Publisher: raw,
		Topic:     testTopic,
		Logger:    testutils.NewDiscardLogger(t),
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
	require.Equal(t, "outer", nextAttempt(t, raw).id)
	require.Equal(t, "nested", nextAttempt(t, raw).id)
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

func TestPublisherFailureDoesNotBlockAnotherKey(t *testing.T) {
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

	require.NoError(t, p.Publish(testTopic, testMessage(t.Context(), "healthy", "key-b")))
	require.Eventually(t, func() bool { return len(raw.attemptsFor("healthy")) == 1 }, 5*time.Second, 20*time.Millisecond)
	require.Equal(t, 1, len(raw.attemptsFor("healthy")))
	eventuallyRowCount(t, client, 1)
	require.Eventually(t, func() bool { return len(raw.attemptsFor("failed")) >= 2 }, 5*time.Second, 20*time.Millisecond,
		"next publish should retry pending rows")
}

func TestPublisherRetriesFailedHeadBeforeLaterSameKeyEvent(t *testing.T) {
	// Given a broker failure for the first event on one message key.
	raw := newRecordingPublisher()
	var failHead atomic.Bool
	failHead.Store(true)
	raw.setOnSend(func(attempt publishedMessage) error {
		if attempt.id == "head" && failHead.Load() {
			return errors.New("injected broker failure")
		}
		return nil
	})
	p, client := newTestPublisher(t, raw)
	require.NoError(t, p.Publish(testTopic, testMessage(t.Context(), "head", "customer-1")))
	require.Equal(t, "head", nextAttempt(t, raw).id)
	eventuallyRowCount(t, client, 1)

	// When a later same-key event and an unrelated event are committed, the
	// unrelated event can pass while the same-key event remains pending.
	require.NoError(t, p.Publish(testTopic, testMessage(t.Context(), "follower", "customer-1")))
	require.NoError(t, p.Publish(testTopic, testMessage(t.Context(), "unrelated", "customer-2")))
	require.Eventually(t, func() bool {
		for _, id := range raw.deliveredIDs() {
			if id == "unrelated" {
				return true
			}
		}
		return false
	}, 5*time.Second, 20*time.Millisecond)
	require.Empty(t, raw.attemptsFor("follower"))
	eventuallyRowCount(t, client, 2)

	// Then a later publish retries the head and delivers the follower after it.
	failHead.Store(false)
	require.NoError(t, p.Publish(testTopic, testMessage(t.Context(), "wake", "customer-3")))
	require.Eventually(t, func() bool {
		ids := raw.deliveredIDs()
		for i := range ids {
			if ids[i] == "head" {
				for _, later := range ids[i+1:] {
					if later == "follower" {
						return true
					}
				}
			}
		}
		return false
	}, 5*time.Second, 20*time.Millisecond)
	eventuallyRowCount(t, client, 0)
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

func TestConcurrentEnqueuePreservesMessageKeyOrder(t *testing.T) {
	for _, rollbackFirst := range []bool{false, true} {
		name := "first commits"
		if rollbackFirst {
			name = "first rolls back"
		}
		t.Run(name, func(t *testing.T) {
			// Given an uncommitted event and a second writer for its message key.
			raw := newRecordingPublisher()
			p, client := newTestPublisher(t, raw)
			creator := enttx.NewCreator(client)
			firstEnqueued := make(chan struct{})
			releaseFirst := make(chan struct{})
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(releaseFirst) }) }
			defer release()
			firstDone := make(chan error, 1)
			secondDone := make(chan error, 1)
			rollback := errors.New("rollback first writer")

			go func() {
				firstDone <- transaction.RunWithNoValue(t.Context(), creator, func(ctx context.Context) error {
					if err := p.Publish(testTopic, testMessage(ctx, "same-key-first", "customer-1")); err != nil {
						return err
					}
					close(firstEnqueued)
					<-releaseFirst
					if rollbackFirst {
						return rollback
					}
					return nil
				})
			}()
			select {
			case <-firstEnqueued:
			case err := <-firstDone:
				t.Fatalf("first enqueue failed: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("first enqueue did not complete")
			}
			secondStarted := make(chan struct{})
			go func() {
				secondDone <- transaction.RunWithNoValue(t.Context(), creator, func(ctx context.Context) error {
					close(secondStarted)
					return p.Publish(testTopic, testMessage(ctx, "same-key-second", "customer-1"))
				})
			}()
			<-secondStarted
			select {
			case err := <-secondDone:
				t.Fatalf("same-key enqueue passed the uncommitted predecessor: %v", err)
			case <-time.After(100 * time.Millisecond):
			}

			// When a different message key is published, it can commit and deliver
			// while the first writer is still open.
			otherDone := make(chan error, 1)
			go func() {
				otherDone <- p.Publish(testTopic, testMessage(t.Context(), "other-key", "customer-2"))
			}()
			select {
			case err := <-otherDone:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("different-key enqueue blocked by the first writer")
			}
			require.Equal(t, "other-key", nextAttempt(t, raw).id)
			noAttempt(t, raw)

			release()
			// Then the second same-key event follows a committed first event, or
			// becomes the only event when the first writer rolls back.
			firstErr := <-firstDone
			if rollbackFirst {
				require.ErrorIs(t, firstErr, rollback)
			} else {
				require.NoError(t, firstErr)
			}
			require.NoError(t, <-secondDone)
			if !rollbackFirst {
				require.Equal(t, "same-key-first", nextAttempt(t, raw).id)
			}
			require.Equal(t, "same-key-second", nextAttempt(t, raw).id)
			eventuallyRowCount(t, client, 0)
		})
	}
}

func TestConcurrentPublishersKeepEachMessageKeyInOrder(t *testing.T) {
	// Given two relay instances and a blocked broker send for the first key.
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
		DB:        client,
		Publisher: raw,
		Topic:     testTopic,
		Logger:    testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, p2.Close()) })

	require.NoError(t, p1.Publish(testTopic, testMessage(t.Context(), "first", "customer-1")))
	require.Equal(t, "first", nextAttempt(t, raw).id)
	// When another event for the same key and one for a different key arrive,
	// the different key can be delivered while the first send remains blocked.
	require.NoError(t, p2.Publish(testTopic, testMessage(t.Context(), "second", "customer-1")))
	require.NoError(t, p2.Publish(testTopic, testMessage(t.Context(), "unrelated", "customer-2")))
	require.Equal(t, "unrelated", nextAttempt(t, raw).id)
	noAttempt(t, raw)

	release()
	// Then the queued same-key event is delivered after the first.
	require.Equal(t, "second", nextAttempt(t, raw).id)
	eventuallyRowCount(t, client, 0)
}

func TestPublisherDrainsBeyondOneBoundedPass(t *testing.T) {
	// Given more events than two complete drain passes in one transaction.
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

	// When the commit wakes the drainer, it schedules continuation itself.
	// Then every event is delivered in order without another publish.
	require.Eventually(t, func() bool { return len(raw.deliveredIDs()) == eventCount }, 20*time.Second, 20*time.Millisecond)
	eventuallyRowCount(t, client, 0)
	ids := raw.deliveredIDs()
	for i, id := range ids {
		require.Equal(t, fmt.Sprintf("event-%03d", i), id)
	}
}

var _ message.Publisher = (*recordingPublisher)(nil)
