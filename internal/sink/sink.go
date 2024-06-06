package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/internal/dedupe"
	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/internal/meter"
)

var namespaceTopicRegexp = regexp.MustCompile(`^om_([A-Za-z0-9]+(?:_[A-Za-z0-9]+)*)_events$`)

type SinkMessage struct {
	Namespace    string
	KafkaMessage *kafka.Message
	Serialized   *serializer.CloudEventsKafkaPayload
	Error        *ProcessingError
}

type Sink struct {
	config            SinkConfig
	running           bool
	buffer            *SinkBuffer
	flushTimer        *time.Timer
	flushEventCounter metric.Int64Counter
	messageCounter    metric.Int64Counter
	namespaceStore    *NamespaceStore
	namespaceRefetch  *time.Timer

	mu sync.Mutex
}

type SinkConfig struct {
	Logger          *slog.Logger
	Tracer          trace.Tracer
	MetricMeter     metric.Meter
	MeterRepository meter.Repository
	Storage         Storage
	Deduplicator    dedupe.Deduplicator
	Consumer        *kafka.Consumer
	// MinCommitCount is the minimum number of messages to wait before flushing the buffer.
	// Whichever happens earlier MinCommitCount or MaxCommitWait will trigger a flush.
	MinCommitCount int
	// MaxCommitWait is the maximum time to wait before flushing the buffer
	MaxCommitWait time.Duration
	// NamespaceRefetch is the interval to refetch exsisting namespaces and meters
	// this information is used to configure which topics the consumer subscribes and
	// the meter configs used in event validation.
	NamespaceRefetch time.Duration
	// OnFlushSuccess is an optional lifecycle hook
	OnFlushSuccess func(string, int64)
}

func NewSink(config SinkConfig) (*Sink, error) {
	if config.Deduplicator == nil {
		config.Logger.Warn("deduplicator is not set, deduplication will be disabled")
	}

	// Defaults
	if config.MinCommitCount == 0 {
		config.MinCommitCount = 1
	}
	if config.MaxCommitWait == 0 {
		config.MaxCommitWait = 1 * time.Second
	}
	if config.NamespaceRefetch == 0 {
		config.NamespaceRefetch = 15 * time.Second
	}

	// Initialize OTel metrics
	messageCounter, err := config.MetricMeter.Int64Counter(
		"sink.kafka.messages",
		metric.WithDescription("The number of messages received from Kafka"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create messages counter: %w", err)
	}

	flushEventCounter, err := config.MetricMeter.Int64Counter(
		"sink.flush.events",
		metric.WithDescription("The number of events processed"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create events counter: %w", err)
	}

	sink := &Sink{
		config:            config,
		buffer:            NewSinkBuffer(),
		namespaceStore:    NewNamespaceStore(),
		flushEventCounter: flushEventCounter,
		messageCounter:    messageCounter,
	}

	return sink, nil
}

// flush flushes the 1. buffer to storage, 2. sets dedupe and 3. store the offset
// called when max wait time or min commit count reached
func (s *Sink) flush() error {
	ctx := context.TODO()
	logger := s.config.Logger.With("operation", "flush")

	s.clearFlushTimer()
	defer s.setFlushTimer()

	// Nothing to flush
	if s.buffer.Size() == 0 {
		logger.Debug("buffer is empty: nothing to flush")
		return nil
	}

	// Use mutex locking to avoid interruption of Sink during flush operation in order to avoid
	// inconsistent state where data is already stored in storage (Clickhouse), but not in
	// Kafka or Redis which would result in stored events being reprocessed which would violate
	// "exactly once" guarantee.
	logger.Debug("acquiring lock to prevent closing sink during flush operation")
	s.mu.Lock()
	defer func() {
		s.mu.Unlock()
		logger.Debug("releasing lock")
	}()

	ctx, lockSpan := s.config.Tracer.Start(ctx, "flush-lock")
	defer lockSpan.End()

	// Pause partitions to avoid processing new messages while we flush
	err := s.pause()
	if err != nil {
		return fmt.Errorf("failed to pause partitions before flush: %w", err)
	}
	defer func() {
		err = s.resume()
		if err != nil {
			logger.Error("failed to resume partitions after flush", "err", err)
		}
	}()

	messages := s.buffer.Dequeue()

	// Start tracing
	ctx, flushSpan := s.config.Tracer.Start(ctx, "flush", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attribute.Int("size", len(messages))))
	defer flushSpan.End()

	// Dedupe messages so if we have multiple messages in the same batch
	dedupedMessages := dedupeSinkMessages(messages)

	// 1. Persist to storage
	if len(dedupedMessages) > 0 {
		// Persist events to permanent storage
		err := s.persistToStorage(ctx, dedupedMessages)
		if err != nil {
			return fmt.Errorf("failed to persist: %w", err)
		}
	}

	// 2. Store Offset
	// Least once guarantee, if offset commit fails we will re-process the same messages again as they are not committed yet
	var offsetStoreErr error
	// Order to ensure we commit largest offset last
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].KafkaMessage.TopicPartition.Offset < messages[j].KafkaMessage.TopicPartition.Offset
	})
	for _, message := range messages {
		_, err = s.config.Consumer.StoreMessage(message.KafkaMessage)
		if err != nil {
			offsetStoreErr = err
		}
	}

	// 3. Sink to Redis
	// Least once guarantee, if Redis write fails we potenitally accept messages with same idempotency key in future.
	// Deduplicator is an optional dependency so we check if it's set
	if s.config.Deduplicator != nil && len(dedupedMessages) > 0 {
		err := s.dedupeSet(ctx, dedupedMessages)
		if err != nil {
			// Try to commit offset if dedupe fails to ensure consistency
			if offsetStoreErr == nil {
				_, err := s.config.Consumer.Commit()
				if err != nil {
					return fmt.Errorf("failed to commit offset: %w", err)
				}
			}

			// When both offset commit and dedupe sink fails we need to reconcile the state based on logs
			if offsetStoreErr != nil {
				logger.Error("consistency failure", "err", err, "messages", messages)
			}

			// Return error, stop consuming
			return fmt.Errorf("failed to sink to redis: %s", err)
		}
	}

	// Return offset store error if any as above we don't return error
	// to ensure we set deduplication IDs in Redis
	if offsetStoreErr != nil {
		return fmt.Errorf("failed to store offset: %w", offsetStoreErr)
	}

	// Metrics and logs
	logger.Debug("succeeded to flush", "buffer size", len(messages))
	err = s.reportFlushMetrics(ctx, messages)
	if err != nil {
		flushSpan.SetStatus(codes.Error, "failed to report flush metrics")
		flushSpan.RecordError(err)
		return fmt.Errorf("failed to report flush metrics: %w", err)
	}

	return nil
}

// reportFlushMetrics reports metrics to OTel
func (s *Sink) reportFlushMetrics(ctx context.Context, messages []SinkMessage) error {
	namespacesReport := map[string]int64{}

	for _, message := range messages {
		namespaceAttr := attribute.String("namespace", message.Namespace)
		statusAttr := attribute.String("status", "success")

		// Count events per namespace
		namespacesReport[message.Namespace]++

		if message.Error != nil {
			switch message.Error.ProcessingControl {
			case INVALID:
				statusAttr = attribute.String("status", "invalid")
			case DROP:
				statusAttr = attribute.String("status", "drop")
			default:
				return fmt.Errorf("unknown error type: %s", message.Error)
			}
		}
		s.flushEventCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr, statusAttr))
	}

	if s.config.OnFlushSuccess != nil {
		for namespace, count := range namespacesReport {
			s.config.OnFlushSuccess(namespace, count)
		}
	}

	return nil
}

// persist persists a batch of messages to storage or deadletters
func (s *Sink) persistToStorage(ctx context.Context, messages []SinkMessage) error {
	logger := s.config.Logger.With("operation", "persistToStorage")
	persistCtx, persistSpan := s.config.Tracer.Start(ctx, "persist")
	defer persistSpan.End()

	batch := []SinkMessage{}

	// Flter out dropped messages
	for _, message := range messages {
		if message.Error != nil {
			switch message.Error.ProcessingControl {
			case INVALID:
				// Do nothing: include in batch
			case DROP:
				// Skip message from batch
				logger.Debug("dropping message", "error", message.Error, "message", string(message.KafkaMessage.Value), "namespace", message.Namespace)
				continue
			default:
				return fmt.Errorf("unknown error type: %s", message.Error)
			}
		}
		batch = append(batch, message)
	}

	// Storage Batch insert
	if len(batch) > 0 {
		storageCtx, storageSpan := s.config.Tracer.Start(persistCtx, "storage-batch-insert")
		err := s.config.Storage.BatchInsert(storageCtx, batch)
		if err != nil {
			// Returning and error means we will retry the whole batch again
			storageSpan.SetStatus(codes.Error, "failure")
			storageSpan.RecordError(err)
			storageSpan.End()
			return fmt.Errorf("failed to sink to storage: %s", err)
		}
		logger.Debug("succeeded to sink to storage", "buffer size", len(messages))
	}

	return nil
}

// dedupeSet sets the dedupe keys in Deduplicator with retry
func (s *Sink) dedupeSet(ctx context.Context, messages []SinkMessage) error {
	logger := s.config.Logger.With("operation", "dedupeSet")
	dedupeCtx, dedupeSet := s.config.Tracer.Start(ctx, "dedupe-set")
	dedupeSet.SetAttributes(
		attribute.Int("size", len(messages)),
	)

	dedupeItems := []dedupe.Item{}
	for _, message := range messages {
		dedupeItems = append(dedupeItems, dedupe.Item{
			Namespace: message.Namespace,
			ID:        message.Serialized.Id,
			Source:    message.Serialized.Source,
		})
	}

	// We retry with exponential backoff as it's critical that either step #2 or #3 succeeds.
	err := retry.Do(
		func() error {
			return s.config.Deduplicator.Set(dedupeCtx, dedupeItems...)
		},
		retry.Context(dedupeCtx),
		retry.OnRetry(func(n uint, err error) {
			dedupeSet.AddEvent("retry", trace.WithAttributes(attribute.Int("count", int(n))))
			logger.Warn("failed to sink to redis, will retry", "err", err, "retry", n)
		}),
	)
	if err != nil {
		dedupeSet.SetStatus(codes.Error, "dedupe set failure")
		dedupeSet.RecordError(err)
		dedupeSet.End()
		return fmt.Errorf("failed to sink to redis: %s", err)
	}
	dedupeSet.End()
	logger.Debug("succeeded to sink to redis", "buffer size", len(messages))
	return nil
}

func (s *Sink) getNamespaces() (*NamespaceStore, error) {
	ctx := context.TODO()

	meters, err := s.config.MeterRepository.ListAllMeters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get meters: %s", err)
	}

	namespaceStore := NewNamespaceStore()
	for _, meter := range meters {
		namespaceStore.AddMeter(meter)
	}

	return namespaceStore, nil
}

func (s *Sink) subscribeToNamespaces() error {
	logger := s.config.Logger.With("operation", "subscribeToNamespaces")
	ns, err := s.getNamespaces()
	if err != nil {
		return fmt.Errorf("failed to get namespaces: %s", err)
	}

	// Identify if we have a new namespace
	isNewNamespace := len(ns.namespaces) != len(s.namespaceStore.namespaces)
	if !isNewNamespace {
		for key := range ns.namespaces {
			if s.namespaceStore.namespaces[key] == nil {
				isNewNamespace = true
				break
			}
		}
	}

	// We always replace store to ensure we have latest meter changes
	s.namespaceStore = ns

	// We only subscribe to topics if we have a new namespace
	if isNewNamespace {
		// We always subscribe to all namespaces as consumer.SubscribeTopics replaces the current subscription.
		topics := getTopics(*s.namespaceStore)
		logger.Info("new namespaces detected, subscribing to topics", "topics", topics)

		err = s.config.Consumer.SubscribeTopics(topics, s.rebalance)
		if err != nil {
			return fmt.Errorf("failed to subscribe to topics: %s", err)
		}
		if err != nil {
			return fmt.Errorf("failed to subscribe to topics: %s", err)
		}
	}

	return nil
}

// Periodically flush even if MinCommitCount threshold not reached yet but MaxCommitWait time elapsed
func (s *Sink) setFlushTimer() {
	logger := s.config.Logger.With("operation", "setFlushTimer")

	flush := func() {
		err := s.flush()
		if err != nil {
			// TODO: should we panic?
			logger.Error("failed to flush", "err", err)
		}
	}

	// Schedule flush
	s.flushTimer = time.AfterFunc(s.config.MaxCommitWait, flush)
}

// Clear flush timer, as there are no parallel flushes there is no need to be thread safe
func (s *Sink) clearFlushTimer() {
	if s.flushTimer != nil {
		s.flushTimer.Stop()
	}
}

// Run starts the Kafka consumer and sinks the events to Clickhouse
func (s *Sink) Run() error {
	ctx := context.TODO()
	logger := s.config.Logger.With("operation", "run")
	logger.Info("starting sink")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Fetch namespaces and meters and subscribe to them
	err := s.subscribeToNamespaces()
	if err != nil {
		return fmt.Errorf("failed to subscribe to namespaces: %s", err)
	}

	// Periodically refetch namespaces and meters
	var refetch func()
	refetch = func() {
		err := s.subscribeToNamespaces()
		if err != nil {
			// TODO: should we panic?
			logger.Error("failed to subscribe to namespaces", "err", err)
		}
		s.namespaceRefetch = time.AfterFunc(s.config.NamespaceRefetch, refetch)
	}
	s.namespaceRefetch = time.AfterFunc(s.config.NamespaceRefetch, refetch)

	// Reset state
	s.running = true

	// Start flush timer, this will be cleared and restarted by flush
	s.setFlushTimer()

	for s.running {
		select {
		case sig := <-sigchan:
			logger.Error("caught signal, terminating", "sig", sig)
			s.running = false
		default:
			ev := s.config.Consumer.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				sinkMessage := SinkMessage{
					KafkaMessage: e,
				}
				namespace, kafkaCloudEvent, err := s.ParseMessage(e)
				if err != nil {
					if perr, ok := err.(*ProcessingError); ok {
						sinkMessage.Error = perr
					} else {
						return fmt.Errorf("failed to parse message: %w", err)
					}
				}
				sinkMessage.Namespace = namespace
				sinkMessage.Serialized = kafkaCloudEvent

				// Message counter
				namespaceAttr := attribute.String("namespace", namespace)
				s.messageCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr))

				s.buffer.Add(sinkMessage)
				logger.Debug("event added to buffer", "partition", e.TopicPartition.Partition, "offset", e.TopicPartition.Offset, "event", kafkaCloudEvent)

				// Flush buffer and commit messages
				if s.buffer.Size() >= s.config.MinCommitCount {
					err = s.flush()
					if err != nil {
						// Stop processing, non-recoverable error
						return fmt.Errorf("failed to flush: %w", err)
					}
				}
			case kafka.Error:
				// Errors should generally be considered
				// informational, the client will try to
				// automatically recover.
				logger.Error("kafka error", "code", e.Code(), "event", e)
			case kafka.OffsetsCommitted:
				// do nothing, this is an ack of the periodic offset commit
				logger.Debug("kafka offset committed", "offset", e.Offsets)
			default:
				logger.Debug("kafka ignored event", "event", e)
			}
		}
	}

	logger.Info("closing sink")
	return s.Close()
}

func (s *Sink) pause() error {
	// Pause partitions to avoid processing new messages while we flush
	assignedPartitions, err := s.config.Consumer.Assignment()
	if err != nil {
		return fmt.Errorf("failed to get assigned partitions: %w", err)
	}
	err = s.config.Consumer.Pause(assignedPartitions)
	if err != nil {
		return fmt.Errorf("failed to pause partitions before flush: %w", err)
	}
	return nil
}

func (s *Sink) resume() error {
	assignedPartitions, err := s.config.Consumer.Assignment()
	if err != nil {
		return fmt.Errorf("failed to get assigned partitions: %w", err)
	}
	err = s.config.Consumer.Resume(assignedPartitions)
	if err != nil {
		return fmt.Errorf("failed to resume partitions after flush: %w", err)
	}

	return nil
}

func (s *Sink) rebalance(c *kafka.Consumer, event kafka.Event) error {
	logger := s.config.Logger.With("operation", "rebalance")

	switch e := event.(type) {
	case kafka.AssignedPartitions:
		// We resume the consumer after the partitions are assigned to start processing new messages.
		err := s.resume()
		if err != nil {
			return fmt.Errorf("failed to resume after assigned partitions: %w", err)
		}

		// Logs newly assigned partitions only (doesn't log already assigned partitions)
		logger.Info("kafka partition assignment", "partitions", prettyPartitions(e.Partitions))

		if len(e.Partitions) == 0 {
			return nil
		}

		// Consumer to use the committed offset as a start position,
		// with a fallback to `auto.offset.reset` if there is no committed offset.
		// Auto offset reset is typically should be set to latest, so we will only consume new messages.
		// Where old messages are already processed and stored in ClickHouse.
		for i := range e.Partitions {
			e.Partitions[i].Offset = kafka.OffsetStored
		}

		// IncrementalAssign adds the specified partitions to the current set of partitions to consume.
		err = s.config.Consumer.IncrementalAssign(e.Partitions)
		if err != nil {
			return fmt.Errorf("failed to assign partitions: %w", err)
		}

	case kafka.RevokedPartitions:
		// We pause the consumer before the partitions are revoked to avoid processing new messages from revoked partitions.
		// Consumption will be resumed after the new partitions are assigned. See above.
		err := s.pause()
		if err != nil {
			return fmt.Errorf("failed to pause after revoked partitions: %w", err)
		}

		// Logs revoked partitions only
		logger.Info("kafka partition revoke", "partitions", prettyPartitions(e.Partitions))

		if len(e.Partitions) == 0 {
			return nil
		}

		// Usually, the rebalance callback for `RevokedPartitions` is called
		// just before the partitions are revoked. We can be certain that a
		// partition being revoked is not yet owned by any other consumer.
		// This way, logic like storing any pending offsets or committing
		// offsets can be handled.
		// However, there can be cases where the assignment is lost
		// involuntarily. In this case, the partition might already be owned
		// by another consumer, and operations including committing
		// offsets may not work.
		if s.config.Consumer.AssignmentLost() {
			// Our consumer has been kicked out of the group and the
			// entire assignment is thus lost.
			logger.Warn("assignment lost involuntarily, commit may fail")
		}

		// IncrementalUnassign removes the specified partitions from the current set of partitions to consume.
		err = s.config.Consumer.IncrementalUnassign(e.Partitions)
		if err != nil {
			return fmt.Errorf("failed to unassign partitions: %w", err)
		}

		// Remove messages for revoked partitions from buffer
		s.buffer.RemoveByPartitions(e.Partitions)
	default:
		logger.Error("unxpected event type", "event", e)
	}

	return nil
}

func (s *Sink) ParseMessage(e *kafka.Message) (string, *serializer.CloudEventsKafkaPayload, error) {
	ctx := context.TODO()

	// Get Namespace
	namespace, err := getNamespace(*e.TopicPartition.Topic)
	if err != nil {
		return "", nil, NewProcessingError(fmt.Sprintf("failed to get namespace from topic: %s, %s", *e.TopicPartition.Topic, err), DROP)
	}

	// Parse Kafka Event
	var kafkaCloudEvent serializer.CloudEventsKafkaPayload
	err = json.Unmarshal(e.Value, &kafkaCloudEvent)
	if err != nil {
		// We should never have events we can't json parse, so we drop them
		return namespace, &kafkaCloudEvent, NewProcessingError(fmt.Sprintf("failed to json parse kafka message: %s", err), DROP)
	}

	// Dedupe, this stores key in store which means if sink fails and restarts it will not process the same message again
	// Dedupe is an optional dependency so we check if it's set
	if s.config.Deduplicator != nil {
		isUnique, err := s.config.Deduplicator.CheckUnique(ctx, dedupe.Item{
			Namespace: namespace,
			ID:        kafkaCloudEvent.Id,
			Source:    kafkaCloudEvent.Source,
		})
		if err != nil {
			// Stop processing, non-recoverable error
			return namespace, &kafkaCloudEvent, fmt.Errorf("failed to check uniqueness of kafka message: %w", err)
		}

		if !isUnique {
			return namespace, &kafkaCloudEvent, NewProcessingError("skipping non unique message", DROP)
		}
	}

	// Validation
	err = s.namespaceStore.ValidateEvent(ctx, kafkaCloudEvent, namespace)
	return namespace, &kafkaCloudEvent, err
}

func (s *Sink) Close() error {
	logger := s.config.Logger.With("operation", "close")

	logger.Info("closing sink")

	// Use mutex locking to avoid interruption Sink during flush operation in order to avoid
	// inconsistent state where data is already stored in storage (Clickhouse), but not in
	// Kafka or Redis which would result in stored events being reprocessed which would violate
	// "exactly once" guarantee.
	logger.Debug("acquiring lock to prevent closing sink during flush operation")
	s.mu.Lock()
	defer func() {
		s.mu.Unlock()
		logger.Debug("releasing lock")
	}()

	s.running = false
	if s.namespaceRefetch != nil {
		s.namespaceRefetch.Stop()
	}
	if s.flushTimer != nil {
		s.flushTimer.Stop()
	}

	return nil
}

// getNamespace from topic
func getNamespace(topic string) (string, error) {
	tmp := namespaceTopicRegexp.FindStringSubmatch(topic)
	if len(tmp) != 2 || tmp[1] == "" {
		return "", fmt.Errorf("namespace not found in topic: %s", topic)
	}
	return tmp[1], nil
}

// getTopics return topics from namespaces
func getTopics(ns NamespaceStore) []string {
	topics := []string{}
	for namespace := range ns.namespaces {
		topic := fmt.Sprintf("om_%s_events", namespace)
		topics = append(topics, topic)
	}

	return topics
}

// dedupeSinkMessages removes duplicates from a list of events
func dedupeSinkMessages(events []SinkMessage) []SinkMessage {
	keys := make(map[string]bool)
	list := []SinkMessage{}

	for _, event := range events {
		key := dedupe.Item{
			Namespace: event.Namespace,
			ID:        event.Serialized.Id,
			Source:    event.Serialized.Source,
		}.Key()
		if _, value := keys[key]; !value {
			keys[key] = true
			list = append(list, event)
		}
	}
	return list
}
