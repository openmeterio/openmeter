package kafka

import (
	"encoding/json"
	"net/http"
	"slices"
	"sync"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill/message"
)

// SubscriptionStatus observes the configured topic subscriptions. Its zero value is unready.
// It is separate from process health: rebalances can temporarily make it unready.
type SubscriptionStatus struct {
	mu       sync.RWMutex
	topics   []string
	sessions map[*subscriptionHandler]*subscriptionSession
	closed   bool
}

func NewSubscriptionStatus(topics []string) *SubscriptionStatus {
	configuredTopics := slices.Clone(topics)
	slices.Sort(configuredTopics)

	return &SubscriptionStatus{
		topics:   slices.Compact(configuredTopics),
		sessions: make(map[*subscriptionHandler]*subscriptionSession),
	}
}

type subscriptionSession struct {
	session     sarama.ConsumerGroupSession
	initialized map[string]map[int32]struct{}
}

type SubscriptionSnapshot struct {
	Ready       bool               `json:"ready"`
	Assigned    map[string][]int32 `json:"assigned"`
	Initialized map[string][]int32 `json:"initialized"`
}

func (s *SubscriptionStatus) Snapshot() SubscriptionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := SubscriptionSnapshot{
		Assigned:    make(map[string][]int32),
		Initialized: make(map[string][]int32),
	}
	if s.closed {
		return status
	}

	// Watermill creates one consumer-group session per subscribed topic.
	// Knowing the full topic list prevents readiness passing after just the first subscription.
	status.Ready = len(s.topics) > 0 && len(s.sessions) == len(s.topics)
	for _, observed := range s.sessions {
		if observed.session.Context().Err() != nil {
			status.Ready = false
			continue
		}

		// Claims can change within a session during cooperative rebalances.
		for topic, partitions := range observed.session.Claims() {
			if !slices.Contains(s.topics, topic) {
				status.Ready = false
			}

			status.Assigned[topic] = append(status.Assigned[topic], partitions...)
			if status.Initialized[topic] == nil {
				status.Initialized[topic] = make([]int32, 0, len(partitions))
			}

			for _, partition := range partitions {
				if _, ok := observed.initialized[topic][partition]; ok {
					status.Initialized[topic] = append(status.Initialized[topic], partition)
				} else {
					status.Ready = false
				}
			}
		}
	}

	for topic := range status.Assigned {
		slices.Sort(status.Assigned[topic])
		slices.Sort(status.Initialized[topic])
	}

	return status
}

func (s *SubscriptionStatus) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	status := s.Snapshot()
	w.Header().Set("Content-Type", "application/json")
	if !status.Ready {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	_ = json.NewEncoder(w).Encode(status)
}

// Watermill exposes consumer-group handler decoration through its tracer hook.
// The other methods pass through unchanged to preserve the subscriber's tracing behavior.
type subscriptionObserver struct {
	status *SubscriptionStatus
}

func (o subscriptionObserver) WrapConsumer(c sarama.Consumer) sarama.Consumer {
	return c
}

func (o subscriptionObserver) WrapPartitionConsumer(c sarama.PartitionConsumer) sarama.PartitionConsumer {
	return c
}

func (o subscriptionObserver) WrapSyncProducer(_ *sarama.Config, p sarama.SyncProducer) sarama.SyncProducer {
	return p
}

func (o subscriptionObserver) WrapConsumerGroupHandler(handler sarama.ConsumerGroupHandler) sarama.ConsumerGroupHandler {
	return &subscriptionHandler{ConsumerGroupHandler: handler, status: o.status}
}

type subscriptionHandler struct {
	sarama.ConsumerGroupHandler
	status *SubscriptionStatus
}

func (h *subscriptionHandler) Setup(session sarama.ConsumerGroupSession) error {
	if err := h.ConsumerGroupHandler.Setup(session); err != nil {
		return err
	}

	h.status.mu.Lock()
	defer h.status.mu.Unlock()
	if h.status.sessions == nil {
		h.status.sessions = make(map[*subscriptionHandler]*subscriptionSession)
	}

	h.status.sessions[h] = &subscriptionSession{
		session:     session,
		initialized: make(map[string]map[int32]struct{}),
	}

	return nil
}

func (h *subscriptionHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.status.mu.Lock()
	if observed := h.status.sessions[h]; observed != nil && observed.session == session {
		delete(h.status.sessions, h)
	}

	h.status.mu.Unlock()

	return h.ConsumerGroupHandler.Cleanup(session)
}

func (h *subscriptionHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// Sarama selects the starting offset before calling ConsumeClaim, even when
	// the topic is empty. Setup alone is too early for newest-offset subscribers.
	h.status.mu.Lock()
	if observed := h.status.sessions[h]; observed != nil && observed.session == session {
		if observed.initialized[claim.Topic()] == nil {
			observed.initialized[claim.Topic()] = make(map[int32]struct{})
		}

		observed.initialized[claim.Topic()][claim.Partition()] = struct{}{}
	}

	h.status.mu.Unlock()
	defer h.claimStopped(session, claim)

	return h.ConsumerGroupHandler.ConsumeClaim(session, claim)
}

func (h *subscriptionHandler) claimStopped(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) {
	h.status.mu.Lock()
	defer h.status.mu.Unlock()
	if observed := h.status.sessions[h]; observed != nil && observed.session == session {
		delete(observed.initialized[claim.Topic()], claim.Partition())
	}
}

type observedSubscriber struct {
	message.Subscriber
	status *SubscriptionStatus
}

func (s *observedSubscriber) Close() error {
	s.status.mu.Lock()
	s.status.closed = true
	s.status.mu.Unlock()

	return s.Subscriber.Close()
}
