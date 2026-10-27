package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionStatusWaitsForEveryAssignedPartition(t *testing.T) {
	// Given an empty topic with four assigned partitions and no consumed messages.
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	session := &subscriptionTestSession{ctx: ctx, claims: map[string][]int32{"events": {0, 1, 2, 3}}}
	status := NewSubscriptionStatus([]string{"events"})
	delegate := &subscriptionTestHandler{started: make(chan int32, 4)}
	handler := (subscriptionObserver{status: status}).WrapConsumerGroupHandler(delegate)
	require.False(t, status.Snapshot().Ready)
	require.NoError(t, handler.Setup(session))
	require.False(t, status.Snapshot().Ready, "group setup happens before offset selection")

	// When only three partition claims have started, the endpoint must stay unready.
	finished := make(chan error, 4)
	for partition := range int32(3) {
		go func() {
			finished <- handler.ConsumeClaim(session, subscriptionTestClaim{topic: "events", partition: partition})
		}()
		<-delegate.started
	}

	response := httptest.NewRecorder()
	status.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz/subscriptions", nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	var partial SubscriptionSnapshot
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &partial))
	require.Equal(t, []int32{0, 1, 2, 3}, partial.Assigned["events"])
	require.Equal(t, []int32{0, 1, 2}, partial.Initialized["events"])

	go func() {
		finished <- handler.ConsumeClaim(session, subscriptionTestClaim{topic: "events", partition: 3})
	}()
	<-delegate.started

	// Then readiness passes without waiting for the first message or backlog processing.
	response = httptest.NewRecorder()
	status.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz/subscriptions", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "application/json", response.Header().Get("Content-Type"))
	var ready SubscriptionSnapshot
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &ready))
	require.True(t, ready.Ready)
	require.Equal(t, ready.Assigned, ready.Initialized)

	// Snapshot callers cannot mutate the tracker's assignment or initialized state.
	snapshot := status.Snapshot()
	snapshot.Assigned["events"][0] = 99
	snapshot.Initialized["events"][0] = 99
	require.Equal(t, []int32{0, 1, 2, 3}, status.Snapshot().Assigned["events"])
	require.Equal(t, []int32{0, 1, 2, 3}, status.Snapshot().Initialized["events"])

	cancel()
	require.False(t, status.Snapshot().Ready)
	for range 4 {
		require.NoError(t, <-finished)
	}

	require.NoError(t, handler.Cleanup(session))
	require.False(t, status.Snapshot().Ready)
}

func TestSubscriptionStatusWaitsForEveryTopic(t *testing.T) {
	// Given a worker that will subscribe to multiple topics.
	status := NewSubscriptionStatus([]string{"system", "ingest", "balance"})
	observer := subscriptionObserver{status: status}
	var handlers []sarama.ConsumerGroupHandler
	var sessions []*subscriptionTestSession
	for range 3 {
		handlers = append(handlers, observer.WrapConsumerGroupHandler(&subscriptionTestHandler{}))
		sessions = append(sessions, &subscriptionTestSession{ctx: t.Context()})
	}

	// When the first two subscriptions join with no assigned partitions, the worker is still unready.
	require.NoError(t, handlers[0].Setup(sessions[0]))
	require.False(t, status.Snapshot().Ready)
	require.NoError(t, handlers[1].Setup(sessions[1]))
	require.False(t, status.Snapshot().Ready)

	// Then all configured subscriptions must join, including idle subscriptions.
	require.NoError(t, handlers[2].Setup(sessions[2]))
	require.True(t, status.Snapshot().Ready)
	require.NoError(t, handlers[1].Cleanup(sessions[1]))
	require.False(t, status.Snapshot().Ready)
	require.NoError(t, handlers[1].Setup(sessions[1]))
	require.True(t, status.Snapshot().Ready)
}

func TestSubscriptionStatusSessionLifecycle(t *testing.T) {
	t.Run("joined replica with no assignments is ready", func(t *testing.T) {
		status := NewSubscriptionStatus([]string{"events"})
		handler := (subscriptionObserver{status: status}).WrapConsumerGroupHandler(&subscriptionTestHandler{})
		require.NoError(t, handler.Setup(&subscriptionTestSession{ctx: t.Context()}))
		require.True(t, status.Snapshot().Ready)
	})

	t.Run("failed setup is not ready", func(t *testing.T) {
		status := NewSubscriptionStatus([]string{"events"})
		setupErr := errors.New("setup failed")
		handler := (subscriptionObserver{status: status}).WrapConsumerGroupHandler(&subscriptionTestHandler{setupErr: setupErr})
		require.ErrorIs(t, handler.Setup(&subscriptionTestSession{ctx: t.Context()}), setupErr)
		require.False(t, status.Snapshot().Ready)
	})

	t.Run("claim failure clears readiness and preserves the error", func(t *testing.T) {
		status := NewSubscriptionStatus([]string{"events"})
		claimErr := errors.New("claim failed")
		handler := (subscriptionObserver{status: status}).WrapConsumerGroupHandler(&subscriptionTestHandler{claimErr: claimErr})
		session := &subscriptionTestSession{ctx: t.Context(), claims: map[string][]int32{"events": {0}}}
		require.NoError(t, handler.Setup(session))
		require.ErrorIs(t, handler.ConsumeClaim(session, subscriptionTestClaim{topic: "events"}), claimErr)
		require.False(t, status.Snapshot().Ready)
	})

	t.Run("old session callbacks cannot change a replacement session", func(t *testing.T) {
		// Given an initialized claim from a session that is being replaced.
		oldCtx, cancelOld := context.WithCancel(t.Context())
		defer cancelOld()
		status := NewSubscriptionStatus([]string{"events"})
		delegate := &subscriptionTestHandler{started: make(chan int32, 1)}
		handler := (subscriptionObserver{status: status}).WrapConsumerGroupHandler(delegate)
		oldSession := &subscriptionTestSession{ctx: oldCtx, claims: map[string][]int32{"events": {0}}}
		require.NoError(t, handler.Setup(oldSession))
		finished := make(chan error, 2)
		go func() {
			finished <- handler.ConsumeClaim(oldSession, subscriptionTestClaim{topic: "events"})
		}()
		<-delegate.started
		require.True(t, status.Snapshot().Ready)

		// When a new session is ready, the old claim may still be draining.
		newCtx, cancelNew := context.WithCancel(t.Context())
		defer cancelNew()
		newSession := &subscriptionTestSession{ctx: newCtx, claims: map[string][]int32{"events": {0}}}
		require.NoError(t, handler.Setup(newSession))
		require.False(t, status.Snapshot().Ready)
		go func() {
			finished <- handler.ConsumeClaim(newSession, subscriptionTestClaim{topic: "events"})
		}()
		<-delegate.started
		require.True(t, status.Snapshot().Ready)
		cancelOld()
		require.NoError(t, <-finished)
		require.NoError(t, handler.Cleanup(oldSession))

		// Then old-session cleanup does not reset the new session.
		require.True(t, status.Snapshot().Ready)
		cancelNew()
		require.NoError(t, <-finished)
		require.NoError(t, handler.Cleanup(newSession))
		require.False(t, status.Snapshot().Ready)
	})

	t.Run("cooperative assignment changes require newly added claims", func(t *testing.T) {
		status := NewSubscriptionStatus([]string{"events"})
		session := &subscriptionTestSession{ctx: t.Context()}
		handler := (subscriptionObserver{status: status}).WrapConsumerGroupHandler(&subscriptionTestHandler{})
		require.NoError(t, handler.Setup(session))
		require.True(t, status.Snapshot().Ready)
		session.claims = map[string][]int32{"events": {7}}
		require.False(t, status.Snapshot().Ready)
		session.claims = nil
		require.True(t, status.Snapshot().Ready)
	})

	t.Run("subscriber shutdown immediately clears readiness", func(t *testing.T) {
		status := NewSubscriptionStatus([]string{"events"})
		handler := (subscriptionObserver{status: status}).WrapConsumerGroupHandler(&subscriptionTestHandler{})
		require.NoError(t, handler.Setup(&subscriptionTestSession{ctx: t.Context()}))
		require.True(t, status.Snapshot().Ready)
		closeErr := errors.New("close failed")
		subscriber := &observedSubscriber{Subscriber: &subscriptionTestSubscriber{closeErr: closeErr}, status: status}
		require.ErrorIs(t, subscriber.Close(), closeErr)
		require.False(t, status.Snapshot().Ready)
		require.NoError(t, handler.Setup(&subscriptionTestSession{ctx: t.Context()}))
		require.False(t, status.Snapshot().Ready, "late callbacks cannot revive a closed subscriber")
	})
}

type subscriptionTestSession struct {
	sarama.ConsumerGroupSession
	ctx    context.Context
	claims map[string][]int32
}

func (s *subscriptionTestSession) Context() context.Context   { return s.ctx }
func (s *subscriptionTestSession) Claims() map[string][]int32 { return s.claims }

type subscriptionTestClaim struct {
	sarama.ConsumerGroupClaim
	topic     string
	partition int32
}

func (c subscriptionTestClaim) Topic() string    { return c.topic }
func (c subscriptionTestClaim) Partition() int32 { return c.partition }

type subscriptionTestHandler struct {
	started  chan int32
	setupErr error
	claimErr error
}

func (h *subscriptionTestHandler) Setup(sarama.ConsumerGroupSession) error   { return h.setupErr }
func (h *subscriptionTestHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (h *subscriptionTestHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	if h.claimErr != nil {
		return h.claimErr
	}

	h.started <- claim.Partition()
	<-session.Context().Done()

	return nil
}

type subscriptionTestSubscriber struct {
	message.Subscriber
	closeErr error
}

func (s *subscriptionTestSubscriber) Close() error { return s.closeErr }
