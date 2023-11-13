package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
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

// flush flushes the 1. buffer to storage, 2. sets dedupe and 3. commits the offset to Kafka
// called when max wait time or min commit count reached
func (s *Sink) flush() error {
	ctx := context.TODO()
	logger := s.config.Logger.With("operation", "flush")

	// Stop polling new messages from Kafka until we finalize batch
	s.clearFlushTimer()
	defer s.setFlushTimer()

	messages := s.buffer.Dequeue()

	// Nothing to flush
	if len(messages) == 0 {
		return nil
	}

	// Start tracing
	ctx, flushSpan := s.config.Tracer.Start(ctx, "flush", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attribute.Int("size", len(messages))))
	defer flushSpan.End()

	// Dedupe messages so if we have multiple messages in the same batch
	dedupedMessages := dedupeSinkMessages(messages)

	// 1. Persist to storage or deadletter
	if len(dedupedMessages) > 0 {
		// Persist events to permanent storage
		deadletterMessages, err := s.persistToStorage(ctx, messages)
		if err != nil {
			return fmt.Errorf("failed to persist: %w", err)
		}

		// Deadletter failed messages if any
		if len(deadletterMessages) > 0 {
			err := s.deadLetter(ctx, deadletterMessages...)
			if err != nil {
				return fmt.Errorf("failed to deadletter messages: %s", err)
			}
		}
	}

	// 2. Commit Offset to Kafka
	// Least once guarantee, if offset commit fails we will potentially process the same messages again as they are not committed yet
	// If Redis write succeeds but Kafka commit fails we will reprocess messages without side effect as they will be dropped at duplicate check
	// If Redis write fails we will double write them to ClickHouse and we have to rely on ClickHouse's deduplication
	offsetCommitFailure := false
	err := s.offsetCommit(ctx, messages)
	if err != nil {
		logger.Error("failed to commit offset to kafka", "err", err)
		offsetCommitFailure = true
		// do not return error here as we want to write Redis even if commit fails as we already saved them in ClickHouse
	}

	// 3. Sink to Redis
	// Least once guarantee, if Redis write fails we will accept messages with same idempotency key in future and
	// we have to rely on ClickHouse's deduplication.
	// Deduplicator is an optional dependency so we check if it's set
	if s.config.Deduplicator != nil && len(dedupedMessages) > 0 {
		err := s.dedupeSet(ctx, dedupedMessages)
		if err != nil {
			// When both offset commit and dedupe sink fails we need to reconcile the state based on logs
			if offsetCommitFailure {
				logger.Error("consistency failure", "err", err, "messages", messages)
			}

			return fmt.Errorf("failed to sink to redis: %s", err)
		}
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
			case DEADLETTER:
				statusAttr = attribute.String("status", "deadletter")
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
func (s *Sink) persistToStorage(ctx context.Context, messages []SinkMessage) ([]SinkMessage, error) {
	logger := s.config.Logger.With("operation", "persistToStorage")
	persistCtx, persistSpan := s.config.Tracer.Start(ctx, "persist")
	defer persistSpan.End()

	deadletterMessages := []SinkMessage{}
	batch := []SinkMessage{}

	// Flter out deadletter and drop messages
	for _, message := range messages {
		if message.Error != nil {
			switch message.Error.ProcessingControl {
			case DEADLETTER:
				logger.Debug("deadlettering message", "error", message.Error, "message", string(message.KafkaMessage.Value), "namespace", message.Namespace)
				deadletterMessages = append(deadletterMessages, message)
				continue
			case DROP:
				// Do nothing
				logger.Debug("dropping message", "error", message.Error, "message", string(message.KafkaMessage.Value), "namespace", message.Namespace)
				continue
			default:
				return deadletterMessages, fmt.Errorf("unknown error type: %s", message.Error)
			}
		}
		batch = append(batch, message)
	}

	// Storage Batch insert
	if len(batch) > 0 {
		storageCtx, storageSpan := s.config.Tracer.Start(persistCtx, "storage-batch-insert")
		err := s.config.Storage.BatchInsert(storageCtx, batch)
		if err != nil {
			// Note: a single error in batch will make the whole batch fail
			if perr, ok := err.(*ProcessingError); ok {
				switch perr.ProcessingControl {
				case DEADLETTER:
					storageSpan.SetStatus(codes.Error, "deadletter")
					deadletterMessages = append(deadletterMessages, batch...)
				case DROP:
					storageSpan.SetStatus(codes.Error, "drop")
				default:
					storageSpan.SetStatus(codes.Error, "unknown processing error type")
					storageSpan.RecordError(err)
					storageSpan.End()
					return deadletterMessages, fmt.Errorf("unknown error type: %s", err)
				}
			} else {
				// Throwing and error means we will retry the whole batch again
				storageSpan.SetStatus(codes.Error, "failure")
				storageSpan.RecordError(err)
				storageSpan.End()
				return deadletterMessages, fmt.Errorf("failed to sink to storage: %s", err)
			}
		}
		logger.Debug("succeeded to sink to storage", "buffer size", len(messages))
	}

	return deadletterMessages, nil
}

// deadLetter stores invalid message, useful permanent non-recoverable errors like json parsing
func (s *Sink) deadLetter(ctx context.Context, messages ...SinkMessage) error {
	logger := s.config.Logger.With("operation", "deadLetter")
	_, deadletterSpan := s.config.Tracer.Start(ctx, "deadletter")

	err := s.config.Storage.BatchInsertInvalid(ctx, messages)
	if err != nil {
		deadletterSpan.SetStatus(codes.Error, "deadletter failure")
		deadletterSpan.RecordError(err)
		deadletterSpan.End()

		return fmt.Errorf("storing invalid messages: %w", err)
	}

	logger.Debug("succeeded to store invalid", "messages", len(messages))
	deadletterSpan.End()
	return nil
}

// offsetCommit commits the offset to Kafka with retry
func (s *Sink) offsetCommit(ctx context.Context, messages []SinkMessage) error {
	logger := s.config.Logger.With("operation", "offsetCommit")
	offsetCtx, offsetSpan := s.config.Tracer.Start(ctx, "offset-commit")

	offsetStore := NewOffsetStore()
	for _, message := range messages {
		offsetStore.Add(message.KafkaMessage.TopicPartition)
	}
	offsets := offsetStore.Get()

	// We retry with exponential backoff as it's critical that either step #2 or #3 succeeds.
	err := retry.Do(
		func() error {
			commitedOffsets, err := s.config.Consumer.CommitOffsets(offsets)
			if err != nil {
				return err
			}
			logger.Debug("succeeded to commit offset to kafka", "offsets", commitedOffsets)
			return nil
		},
		retry.Context(offsetCtx),
		retry.OnRetry(func(n uint, err error) {
			offsetSpan.AddEvent("retry", trace.WithAttributes(attribute.Int("count", int(n))))
			logger.Warn("failed to commit kafka offset, will retry", "err", err, "retry", n)
		}),
	)
	if err != nil {
		offsetSpan.SetStatus(codes.Error, "offset commit failure")
		offsetSpan.RecordError(err)
		offsetSpan.End()
		return fmt.Errorf("failed to commit offset to kafka: %w", err)
	}

	offsetSpan.End()
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

				// Store message, this won't commit offset immediately just store it for the next manual commit
				_, err = s.config.Consumer.StoreMessage(e)
				if err != nil {
					// Stop processing, non-recoverable error
					return fmt.Errorf("failed to store kafka message for upcoming offset commit: %w", err)
				}

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

func (s *Sink) rebalance(c *kafka.Consumer, event kafka.Event) error {
	logger := s.config.Logger.With("operation", "rebalance")

	switch e := event.(type) {
	case kafka.AssignedPartitions:
		logger.Info("kafka assigned partitions", "partitions", e.Partitions)
		err := s.config.Consumer.Assign(e.Partitions)
		if err != nil {
			return fmt.Errorf("failed to assign partitions: %w", err)
		}
	case kafka.RevokedPartitions:
		logger.Info("kafka revoked partitions", "partitions", e.Partitions)

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

		err := s.flush()
		if err != nil {
			// Stop processing, non-recoverable error
			return fmt.Errorf("failed to flush: %w", err)
		}

		err = s.config.Consumer.Unassign()
		if err != nil {
			return fmt.Errorf("failed to unassign partitions: %w", err)
		}
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
	s.running = false
	if s.namespaceRefetch != nil {
		s.namespaceRefetch.Stop()
	}
	if s.flushTimer != nil {
		s.flushTimer.Stop()
	}
	return s.config.Consumer.Close()
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
