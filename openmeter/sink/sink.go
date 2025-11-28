package sink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
	kafkametrics "github.com/openmeterio/openmeter/pkg/kafka/metrics"
	kafkastats "github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
)

type Sink struct {
	config            SinkConfig
	buffer            *SinkBuffer
	flushTimer        *time.Timer
	flushEventCounter metric.Int64Counter
	messageCounter    metric.Int64Counter
	namespaceRefetch  *time.Timer
	topicResolver     topicresolver.Resolver
	meterCache        *NamespacedMeterCache

	kafkaMetrics *kafkametrics.Metrics

	mu        sync.Mutex
	isRunning atomic.Bool

	namespaceTopicRegexp *regexp.Regexp
}

type SinkConfig struct {
	Logger       *slog.Logger
	Tracer       trace.Tracer
	MetricMeter  metric.Meter
	Storage      Storage
	Deduplicator dedupe.Deduplicator
	Consumer     *kafka.Consumer
	// MinCommitCount is the minimum number of messages to wait before flushing the buffer.
	// Whichever happens earlier MinCommitCount or MaxCommitWait will trigger a flush.
	MinCommitCount int
	// MaxCommitWait is the maximum time to wait before flushing the buffer
	MaxCommitWait time.Duration
	// The time, in milliseconds, spent waiting in poll if data is not available in the buffer.
	// If 0, returns immediately with any records that are available currently in the buffer, else returns empty.
	MaxPollTimeout time.Duration
	// NamespaceRefetch is the interval to refetch exsisting namespaces and meters
	// this information is used to configure which topics the consumer subscribes and
	// the meter configs used in event validation.
	NamespaceRefetch time.Duration

	// NamespaceRefetchTimeout is the timeout for updating namespaces and consumer subscription.
	// It must be less than NamespaceRefetch interval.
	NamespaceRefetchTimeout time.Duration

	// NamespaceTopicRegexp defines the regular expression to match/validate topic names the sink-worker needs to subscribe to.
	NamespaceTopicRegexp string

	// FlushEventHandlers is an optional lifecycle hook, allowing to act on successful batch
	// flushes. To prevent blocking the main sink logic this is always called in a go routine.
	FlushEventHandler flushhandler.FlushEventHandler

	// FlushSuccessTimeout is the timeout for the OnFlushSuccess callback,
	// after this period the context of the callback will be canceled.
	FlushSuccessTimeout time.Duration

	// DrainTimeout is the maximum time to wait before draining the buffer and closing the sink.
	DrainTimeout time.Duration

	TopicResolver topicresolver.Resolver

	// MeterRefetchInterval is the interval to refetch meters from the database
	MeterRefetchInterval time.Duration

	// MeterService is the service to fetch meters from the database
	MeterService meter.Service

	// LogDroppedEvents controls whether dropped events are logged
	LogDroppedEvents bool
}

func (s *SinkConfig) Validate() error {
	if s.Logger == nil {
		return errors.New("logger is required")
	}

	if s.Tracer == nil {
		return errors.New("tracer is required")
	}

	if s.MetricMeter == nil {
		return errors.New("metric meter is required")
	}

	if s.Storage == nil {
		return errors.New("storage is required")
	}

	if s.Consumer == nil {
		return errors.New("consumer is required")
	}

	if s.MinCommitCount < 1 {
		return errors.New("MinCommitCount must be greater than 0")
	}

	if s.MaxCommitWait == 0 {
		return errors.New("MaxCommitWait must be greater than 0")
	}

	if s.MaxPollTimeout == 0 {
		return errors.New("MaxPollTimeout must be greater than 0")
	}

	if s.NamespaceRefetch == 0 {
		return errors.New("NamespaceRefetch must be greater than 0")
	}

	if s.NamespaceRefetchTimeout != 0 && s.NamespaceRefetchTimeout > s.NamespaceRefetch {
		return errors.New("NamespaceRefetchTimeout must be less than or equal to NamespaceRefetch")
	}

	if s.NamespaceTopicRegexp == "" {
		return errors.New("NamespaceTopicRegexp must no be empty")
	}

	if s.FlushSuccessTimeout == 0 {
		return errors.New("FlushSuccessTimeout must be greater than 0")
	}

	if s.DrainTimeout == 0 {
		return errors.New("DrainTimeout must be greater than 0")
	}

	if s.TopicResolver == nil {
		return errors.New("topic resolver is required")
	}

	if s.MeterRefetchInterval <= 0 {
		return errors.New("MeterRefetchInterval must be greater than 0")
	}

	if s.MeterService == nil {
		return errors.New("meter service is required")
	}

	return nil
}

func NewSink(config SinkConfig) (*Sink, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid sink configuration: %w", err)
	}

	namespaceTopicRegexp, err := regexp.Compile(config.NamespaceTopicRegexp)
	if err != nil {
		return nil, fmt.Errorf("invalid namespace topic regexp: %w", err)
	}

	// Warn if deduplicator is not set
	if config.Deduplicator == nil {
		config.Logger.Warn("deduplicator is not set, deduplication will be disabled")
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

	kafkaMetrics, err := kafkametrics.New(config.MetricMeter)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client metrics: %w", err)
	}

	// Default NamespaceRefetchTimeout to 2/3 of NamespaceRefetch if not set.
	if config.NamespaceRefetchTimeout == 0 {
		config.NamespaceRefetchTimeout = (config.NamespaceRefetch / 3) * 2
	}

	meterCache, err := NewNamespaceStore(NamespacedMeterCacheConfig{
		PeriodicRefetchInterval: config.MeterRefetchInterval,
		Logger:                  config.Logger,
		MeterService:            config.MeterService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace meter cache: %w", err)
	}

	sink := &Sink{
		config:               config,
		buffer:               NewSinkBuffer(),
		flushEventCounter:    flushEventCounter,
		messageCounter:       messageCounter,
		kafkaMetrics:         kafkaMetrics,
		topicResolver:        config.TopicResolver,
		namespaceTopicRegexp: namespaceTopicRegexp,
		meterCache:           meterCache,
	}

	return sink, nil
}

// flush flushes the 1. buffer to storage, 2. sets dedupe and 3. store the offset
// called when max wait time or min commit count reached
func (s *Sink) flush(ctx context.Context) error {
	logger := s.config.Logger.With("operation", "flush")

	s.clearFlushTimer()
	defer s.setFlushTimer(ctx)

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
			logger.ErrorContext(ctx, "failed to resume partitions after flush", "err", err)
		}
	}()

	messages := s.buffer.Dequeue()

	// Start tracing
	ctx, flushSpan := s.config.Tracer.Start(ctx, "flush", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attribute.Int("size", len(messages))))
	defer flushSpan.End()

	// Dedupe messages so if we have multiple messages in the same batch
	dedupedMessages := dedupeSinkMessages(messages)

	// If deduplicator is set, let's reexecute the deduplication to decrease the number of messages double persisted
	if s.config.Deduplicator != nil && len(dedupedMessages) > 0 {
		dedupeResults, err := s.config.Deduplicator.CheckUniqueBatch(ctx, lo.Map(dedupedMessages, func(message sinkmodels.SinkMessage, _ int) dedupe.Item {
			return message.GetDedupeItem()
		}))
		if err != nil {
			return fmt.Errorf("failed to check uniqueness of kafka messages: %w", err)
		}

		updatedDedupedMessages := make([]sinkmodels.SinkMessage, 0, len(dedupedMessages))
		for _, message := range dedupedMessages {
			if _, ok := dedupeResults.UniqueItems[message.GetDedupeItem()]; ok {
				updatedDedupedMessages = append(updatedDedupedMessages, message)
			}
		}

		dedupedMessages = updatedDedupedMessages
	}

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
				logger.ErrorContext(ctx, "consistency failure", "err", err, "messages", messages)
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

	// Call FlushEventHandler if set
	if s.config.FlushEventHandler != nil {
		go func() {
			ctx, cancel := context.WithTimeout(ctx, s.config.FlushSuccessTimeout)
			defer cancel()

			err := s.config.FlushEventHandler.OnFlushSuccess(ctx, messages)
			if err != nil {
				logger.ErrorContext(ctx, "failed to invoke OnFlushSuccess callback", "err", err)
			}
		}()
	}

	return nil
}

// reportFlushMetrics reports metrics to OTel
//
// TODO: figure out if this needs to return an error or not
func (s *Sink) reportFlushMetrics(ctx context.Context, messages []sinkmodels.SinkMessage) error { //nolint: unparam
	namespacesReport := map[string]int64{}

	for _, message := range messages {
		namespaceAttr := attribute.String("namespace", message.Namespace)
		statusAttr := attribute.String("status", message.Status.State.String())

		// Count events per namespace
		namespacesReport[message.Namespace]++
		s.flushEventCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr, statusAttr))
	}

	return nil
}

// persist persists a batch of messages to storage or deadletters
func (s *Sink) persistToStorage(ctx context.Context, messages []sinkmodels.SinkMessage) error {
	logger := s.config.Logger.With("operation", "persistToStorage")
	persistCtx, persistSpan := s.config.Tracer.Start(ctx, "persist")
	defer persistSpan.End()

	batch := []sinkmodels.SinkMessage{}

	// Flter out dropped messages
	for _, message := range messages {
		switch message.Status.State {
		case sinkmodels.OK:
			// Do nothing: include in batch
		case sinkmodels.DROP:
			// Skip event from batch
			if s.config.LogDroppedEvents {
				logger.WarnContext(ctx, "event dropped",
					slog.String("namespace", message.Namespace),
					slog.String("event", string(message.KafkaMessage.Value)),
					slog.String("error", message.Status.DropError.Error()),
					slog.String("status", message.Status.State.String()),
				)
			}

			continue
		default:
			return fmt.Errorf("unknown state type: %s", message.Status.State.String())
		}
		batch = append(batch, message)
	}

	// Storage Batch insert
	if len(batch) > 0 {
		storageCtx, storageSpan := s.config.Tracer.Start(persistCtx, "storage-batch-insert")
		defer storageSpan.End()
		err := s.config.Storage.BatchInsert(storageCtx, batch)
		if err != nil {
			// Returning and error means we will retry the whole batch again
			storageSpan.SetStatus(codes.Error, "failure")
			storageSpan.RecordError(err)
			return fmt.Errorf("failed to sink to storage: %s", err)
		}
		logger.Debug("succeeded to sink to storage", "buffer size", len(messages))
	}

	return nil
}

// dedupeSet sets the dedupe keys in Deduplicator with retry
func (s *Sink) dedupeSet(ctx context.Context, messages []sinkmodels.SinkMessage) error {
	logger := s.config.Logger.With("operation", "dedupeSet")
	dedupeCtx, dedupeSet := s.config.Tracer.Start(ctx, "dedupe-set")
	dedupeSet.SetAttributes(
		attribute.Int("size", len(messages)),
	)

	dedupeItems := []dedupe.Item{}
	for _, message := range messages {
		switch message.Status.State {
		case sinkmodels.OK:
			dedupeItems = append(dedupeItems, dedupe.Item{
				Namespace: message.Namespace,
				ID:        message.Serialized.Id,
				Source:    message.Serialized.Source,
			})
		case sinkmodels.DROP:
			// Let's not insert already dropped messages into the deduplicator as this signals
			// that we had a problem validating the message, we could redo any validation as needed
			// later but if any error was transient let's retry the message if it's sent again.

			if s.config.LogDroppedEvents {
				logger.WarnContext(ctx, "event dropped",
					slog.String("namespace", message.Namespace),
					slog.String("event", string(message.KafkaMessage.Value)),
					slog.String("error", message.Status.DropError.Error()),
					slog.String("status", message.Status.State.String()),
				)
			}

			continue
		default:
			logger.ErrorContext(ctx, "unknown state type in dedup set", "state", message.Status.State.String())
		}
	}

	if len(dedupeItems) == 0 {
		logger.Debug("no dedupe items to set")
		return nil
	}

	// We retry with exponential backoff as it's critical that either step #2 or #3 succeeds.
	err := retry.Do(
		func() error {
			existingItems, err := s.config.Deduplicator.Set(dedupeCtx, dedupeItems...)
			if err != nil {
				return err
			}

			if len(existingItems) > 0 {
				logger.ErrorContext(ctx, "dedupe: some items already existed in redis",
					"items", lo.Map(existingItems, func(item dedupe.Item, _ int) string {
						return item.Key()
					}),
					"code", "dedupe_set_failure",
				)
			}

			return nil
		},
		retry.Context(dedupeCtx),
		retry.OnRetry(func(n uint, err error) {
			dedupeSet.AddEvent("retry", trace.WithAttributes(attribute.Int("count", int(n))))
			logger.WarnContext(ctx, "failed to sink to redis, will retry", "err", err, "retry", n)
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

func (s *Sink) updateTopicSubscription(ctx context.Context, metadataTimeout time.Duration) error {
	logger := s.config.Logger.With("operation", "updateTopicSubscription")

	logger.Debug("fetching metadata")
	meta, err := s.config.Consumer.GetMetadata(nil, true, int(metadataTimeout.Milliseconds()))
	if err != nil {
		return fmt.Errorf("failed to fetch all topics from Kafka cluster: %w", err)
	}

	topics := make([]string, 0, len(meta.Topics))
	for _, topic := range meta.Topics {
		if !s.namespaceTopicRegexp.MatchString(topic.Topic) {
			logger.Debug("skipping topic as does not match regexp", "topic", topic.Topic, "regexp", s.namespaceTopicRegexp.String())

			continue
		}

		logger.Debug("found matching topic", "topic", topic.Topic, "regexp", s.namespaceTopicRegexp.String())
		topics = append(topics, topic.Topic)
	}

	if len(topics) == 0 {
		logger.WarnContext(ctx, "no topics found to be subscribed to", "regexp", s.namespaceTopicRegexp.String())

		return nil
	}

	err = s.config.Consumer.SubscribeTopics(topics, s.rebalance)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topics: %s", err)
	}

	logger.Debug("successfully subscribed to topics", "topics", topics)

	return nil
}

func (s *Sink) subscribeToNamespaces(ctx context.Context) error {
	logger := s.config.Logger.With("operation", "subscribeToNamespaces")

	ctx, cancel := context.WithTimeout(ctx, s.config.NamespaceRefetchTimeout)
	defer cancel()

	logger.Debug("updating topic subscription: started")

	// Set getting timeout for getting metadata from Kafka to make sure that we can
	metadataTimeout := s.config.NamespaceRefetchTimeout / 2

	err := s.updateTopicSubscription(ctx, metadataTimeout)
	if err != nil {
		return fmt.Errorf("failed to update topic subscription: %w", err)
	}

	logger.Debug("updating topic subscription: done")

	return nil
}

// Periodically flush even if MinCommitCount threshold not reached yet but MaxCommitWait time elapsed
func (s *Sink) setFlushTimer(ctx context.Context) {
	logger := s.config.Logger.With("operation", "setFlushTimer")

	flush := func() {
		err := s.flush(ctx)
		if err != nil {
			// TODO: should we panic?
			logger.ErrorContext(ctx, "failed to flush", "err", err)
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
func (s *Sink) Run(ctx context.Context) error {
	if s.isRunning.Load() {
		return nil
	}

	if err := s.meterCache.Start(ctx); err != nil {
		return fmt.Errorf("failed to start meter cache: %w", err)
	}

	logger := s.config.Logger.With("operation", "run")
	if s.config.FlushEventHandler != nil {
		logger.Info("starting flush event handler")
		if err := s.config.FlushEventHandler.Start(ctx); err != nil {
			return fmt.Errorf("failed to start flush event handler: %w", err)
		}
	}
	logger.Info("starting sink")

	// Fetch namespaces and meters and subscribe to them
	err := s.subscribeToNamespaces(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to namespaces: %s", err)
	}

	// Periodically refetch namespaces and meters
	var refetch func()
	refetch = func() {
		if !s.isRunning.Load() || ctx.Err() != nil {
			logger.DebugContext(ctx, "skipping subscribing to namespaces as either context is canceled or sink is stopped")

			return
		}

		if err = s.subscribeToNamespaces(ctx); err != nil {
			logger.ErrorContext(ctx, "failed to subscribe to namespaces", "err", err)
		}

		s.namespaceRefetch = time.AfterFunc(s.config.NamespaceRefetch, refetch)
	}

	s.namespaceRefetch = time.AfterFunc(s.config.NamespaceRefetch, refetch)

	// Reset state
	s.isRunning.Store(true)

	// Start flush timer, this will be cleared and restarted by flush
	s.setFlushTimer(ctx)

	for s.isRunning.Load() {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled: %w", ctx.Err())

		default:
			ev := s.config.Consumer.Poll(int(s.config.MaxPollTimeout.Milliseconds()))
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				sinkMessage, err := s.parseMessage(ctx, e)
				if err != nil {
					return fmt.Errorf("failed to parse message: %w", err)
				}

				// Message counter
				namespaceAttr := attribute.String("namespace", sinkMessage.Namespace)
				s.messageCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr))

				s.buffer.Add(*sinkMessage)

				logger.DebugContext(ctx, "event added to buffer",
					"partition", e.TopicPartition.Partition,
					"offset", e.TopicPartition.Offset,
					"event", sinkMessage.Serialized,
					"state", sinkMessage.Status.State.String(),
				)

				// Flush buffer and commit messages
				if s.buffer.Size() >= s.config.MinCommitCount {
					err = s.flush(ctx)
					if err != nil {
						// Stop processing, non-recoverable error
						return fmt.Errorf("failed to flush: %w", err)
					}
				}
			case kafka.Error:
				attrs := []any{
					slog.Int("code", int(e.Code())),
					slog.String("error", e.Error()),
				}

				// Log Kafka client "local" errors on warning level as those are mostly informational and the client is
				// able to handle/recover from them automatically.
				// See: https://github.com/confluentinc/librdkafka/blob/master/src/rdkafka.h#L415
				if e.Code() <= -100 {
					logger.WarnContext(ctx, "kafka local error", attrs...)
				} else {
					logger.ErrorContext(ctx, "kafka broker error", attrs...)
				}

			case kafka.OffsetsCommitted:
				// do nothing, this is an ack of the periodic offset commit
				logger.Debug("kafka offset committed", "offset", e.Offsets)
			case *kafka.Stats:
				// Report Kafka client metrics
				if s.kafkaMetrics == nil {
					continue
				}

				go func() {
					var stats kafkastats.Stats

					if err = json.Unmarshal([]byte(e.String()), &stats); err != nil {
						logger.WarnContext(ctx, "failed to unmarshal Kafka client stats", slog.String("err", err.Error()))
					}

					ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()

					s.kafkaMetrics.Add(ctx, &stats)
				}()
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
		logger.Error("unexpected event type", "event", e)
	}

	return nil
}

func (s *Sink) parseMessage(ctx context.Context, e *kafka.Message) (*sinkmodels.SinkMessage, error) {
	if e == nil {
		return nil, fmt.Errorf("kafka message is nil")
	}

	sinkMessage := &sinkmodels.SinkMessage{
		KafkaMessage: e,
		Status: sinkmodels.ProcessingStatus{
			State: sinkmodels.OK,
		},
	}

	// Get Namespace from Kafka message header
	var namespace string
	for _, header := range e.Headers {
		if header.Key == kafkaingest.HeaderKeyNamespace {
			namespace = string(header.Value)

			break
		}
	}

	if namespace == "" {
		sinkMessage.Status = sinkmodels.ProcessingStatus{
			State: sinkmodels.DROP,
			DropError: fmt.Errorf("failed to get namespace as header (%q) for Kafka Message is missing: %s",
				kafkaingest.HeaderKeyNamespace,
				e.TopicPartition.String(),
			),
		}

		return sinkMessage, nil
	}
	sinkMessage.Namespace = namespace

	// Parse Kafka Event
	kafkaCloudEvent := serializer.CloudEventsKafkaPayload{}
	err := json.Unmarshal(e.Value, &kafkaCloudEvent)
	if err != nil {
		sinkMessage.Status = sinkmodels.ProcessingStatus{
			State:     sinkmodels.DROP,
			DropError: errors.New("failed to json parse kafka message"),
		}

		// We should never have events we can't json parse, so we drop them
		return sinkMessage, nil
	}

	if err = serializer.ValidateKafkaPayloadToCloudEvent(kafkaCloudEvent); err != nil {
		sinkMessage.Status = sinkmodels.ProcessingStatus{
			State:     sinkmodels.DROP,
			DropError: fmt.Errorf("failed to validate cloudevents message: %w", err),
		}

		return sinkMessage, nil
	}

	sinkMessage.Serialized = &kafkaCloudEvent

	// Dedupe, this stores key in store which means if sink fails and restarts it will not process the same message again
	// Dedupe is an optional dependency, so we check if it's set
	if s.config.Deduplicator != nil {
		isUnique, err := s.config.Deduplicator.CheckUnique(ctx, sinkMessage.GetDedupeItem())
		if err != nil {
			// Stop processing, non-recoverable error
			return sinkMessage, fmt.Errorf("failed to check uniqueness of kafka message: %w", err)
		}

		if !isUnique {
			sinkMessage.Status = sinkmodels.ProcessingStatus{
				State:     sinkmodels.DROP,
				DropError: errors.New("skipping non unique message"),
			}

			return sinkMessage, nil
		}
	}

	// Let's resolve affected meters
	affectedMeters, err := s.meterCache.GetAffectedMeters(ctx, sinkMessage)
	if err != nil {
		return sinkMessage, fmt.Errorf("failed to get affected meters: %w", err)
	}

	sinkMessage.Meters = affectedMeters

	return sinkMessage, err
}

func (s *Sink) Close() error {
	if !s.isRunning.Load() {
		return nil
	}

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

	s.isRunning.Store(false)

	if s.namespaceRefetch != nil {
		logger.Info("stopping namespace fetcher")
		s.namespaceRefetch.Stop()
	}

	if s.flushTimer != nil {
		logger.Info("stopping flush timer")
		s.flushTimer.Stop()
	}

	return nil
}

// dedupeSinkMessages removes duplicates from a list of events
func dedupeSinkMessages(events []sinkmodels.SinkMessage) []sinkmodels.SinkMessage {
	keys := make(map[string]bool)
	list := []sinkmodels.SinkMessage{}

	for _, event := range events {
		switch event.Status.State {
		case sinkmodels.OK:
			key := dedupe.Item{
				Namespace: event.Namespace,
				ID:        event.Serialized.Id,
				Source:    event.Serialized.Source,
			}.Key()

			if _, value := keys[key]; !value {
				keys[key] = true
				list = append(list, event)
			}
		case sinkmodels.DROP:
			continue
		default:
			// TODO: we should log/error in this case
			continue
		}
	}

	return list
}
