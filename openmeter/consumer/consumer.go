// Package consumer provides a librdkafka-based consumer implementation that processes
// messages from Kafka topics. It is designed to:
// - Use one goroutine per Kafka partition (automatically managed via rebalancing)
// - Support retries with exponential backoff
// - Require dead letter queue (DLQ) for failed messages (mandatory)
// - Support eventbus callbacks via grouphandler (compatible with watermill event handlers)
// - Provide metrics and tracing support
// - Handle correlation IDs and panic recovery
// - Manual offset commits after successful processing or DLQ delivery
//
// This consumer is an alternative to the watermill-based consumer and provides
// more direct control over the Kafka consumer lifecycle. Partition workers are
// automatically started on assignment and stopped on revocation.
package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// Config contains the configuration for the consumer.
type Config struct {
	Consumer *kafka.Consumer
	Producer *kafka.Producer

	Topics []string

	EventBus  eventbus.Publisher
	Marshaler marshaler.Marshaler

	ConsumerConfig config.ConsumerConfiguration

	Logger      *slog.Logger
	MetricMeter metric.Meter
	Tracer      trace.Tracer

	// PollTimeout is the maximum time to wait for messages when polling.
	// If 0, defaults to 100ms.
	PollTimeout time.Duration
}

func (c *Config) Validate() error {
	if c.Consumer == nil {
		return errors.New("consumer is required")
	}

	if c.Producer == nil {
		return errors.New("producer is required for DLQ")
	}

	if len(c.Topics) == 0 {
		return errors.New("at least one topic is required")
	}

	if c.Marshaler == nil {
		return errors.New("marshaler is required")
	}

	if c.Logger == nil {
		return errors.New("logger is required")
	}

	if c.MetricMeter == nil {
		return errors.New("metric meter is required")
	}

	if c.Tracer == nil {
		return errors.New("tracer is required")
	}

	if err := c.ConsumerConfig.Validate(); err != nil {
		return fmt.Errorf("consumer config: %w", err)
	}

	// DLQ is mandatory
	if !c.ConsumerConfig.DLQ.Enabled {
		return errors.New("DLQ must be enabled")
	}

	if c.ConsumerConfig.DLQ.Topic == "" {
		return errors.New("DLQ topic is required")
	}

	return nil
}

// Consumer is the interface for a librdkafka-based consumer that processes messages from Kafka topics.
// It uses one goroutine per assigned partition and provides retry, DLQ, and eventbus callback support.
type Consumer interface {
	// AddHandler adds an event handler to the consumer.
	// Handlers are called in the order they are added.
	AddHandler(handler GroupEventHandler)

	// Run starts consuming messages. It uses a single polling goroutine that routes messages
	// to dedicated partition processing goroutines.
	Run(ctx context.Context) error

	// Close closes the consumer and releases resources.
	Close() error
}

// consumer is a librdkafka-based consumer that processes messages from Kafka topics.
// It uses one goroutine per assigned partition and provides retry, DLQ, and eventbus callback support.
type consumer struct {
	consumer *kafka.Consumer
	producer *kafka.Producer

	config Config
	logger *slog.Logger

	handler   *KafkaMessageHandler
	marshaler marshaler.Marshaler

	// Metrics
	messageProcessingCount metric.Int64Counter
	messageProcessingTime  metric.Int64Histogram
	dlqMessageCount        metric.Int64Counter

	// State
	isRunning atomic.Bool
	isClosed  atomic.Bool
	wg        sync.WaitGroup

	workers workerManager
	// Partition management
	partitionWorkers map[PartitionKey]*partitionWorker
	partitionMu      sync.RWMutex

	// Message routing
	partitionChannels map[PartitionKey]chan *kafka.Message
}

// New creates a new Consumer instance.
func New(cfg Config) (Consumer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	pollTimeout := cfg.PollTimeout
	if pollTimeout == 0 {
		pollTimeout = 100 * time.Millisecond
	}

	// Initialize metrics
	messageProcessingCount, err := cfg.MetricMeter.Int64Counter(
		"consumer.message_processing_count",
		metric.WithDescription("Number of messages processed"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message processing count metric: %w", err)
	}

	messageProcessingTime, err := cfg.MetricMeter.Int64Histogram(
		"consumer.message_processing_time_ms",
		metric.WithDescription("Time spent processing a message (including retries)"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message processing time metric: %w", err)
	}

	dlqMessageCount, err := cfg.MetricMeter.Int64Counter(
		"consumer.dlq_message_count",
		metric.WithDescription("Number of messages sent to DLQ"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DLQ message count metric: %w", err)
	}

	// Create handler (handlers can be added later via AddHandler)
	handler, err := NewKafkaMessageHandler(
		cfg.Marshaler,
		cfg.MetricMeter,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create handler: %w", err)
	}

	c := &consumer{
		consumer:               cfg.Consumer,
		producer:               cfg.Producer,
		config:                 cfg,
		logger:                 cfg.Logger,
		handler:                handler,
		marshaler:              cfg.Marshaler,
		messageProcessingCount: messageProcessingCount,
		messageProcessingTime:  messageProcessingTime,
		dlqMessageCount:        dlqMessageCount,
		partitionWorkers:       make(map[PartitionKey]*partitionWorker),
		partitionChannels:      make(map[PartitionKey]chan *kafka.Message),
	}

	// Subscribe to topics
	if err := c.consumer.SubscribeTopics(cfg.Topics, nil); err != nil {
		return nil, fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	return c, nil
}

// AddHandler adds an event handler to the consumer.
// Handlers are called in the order they are added.
func (c *consumer) AddHandler(handler GroupEventHandler) {
	c.handler.AddHandler(handler)
}

// Run starts consuming messages. It uses a single polling goroutine that routes messages
// to dedicated partition processing goroutines.
func (c *consumer) Run(ctx context.Context) error {
	if !c.isRunning.CompareAndSwap(false, true) {
		return errors.New("consumer is already running")
	}

	defer c.isRunning.Store(false)

	c.logger.InfoContext(ctx, "starting consumer", "topics", c.config.Topics)

	// Start single polling goroutine
	c.wg.Add(1)
	go c.runPollingLoop(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	c.logger.InfoContext(ctx, "consumer context canceled, stopping all partition workers")
	c.stopAllPartitionWorkers()

	c.wg.Wait()
	return ctx.Err()
}

// runPollingLoop runs in a single goroutine and polls for messages and events.
// Messages are routed to partition-specific channels for processing.
func (c *consumer) runPollingLoop(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			c.logger.DebugContext(ctx, "polling loop context canceled")
			return
		default:
			// Poll for messages, rebalance events, and errors
			ev := c.consumer.Poll(int(c.config.PollTimeout.Milliseconds()))
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				// Route message to partition-specific channel
				c.routeMessage(ctx, e)
			case kafka.AssignedPartitions:
				if err := c.handlePartitionAssignment(ctx, e.Partitions); err != nil {
					c.logger.ErrorContext(ctx, "failed to handle partition assignment", "error", err)
					// Continue processing, but log the error
				}
			case kafka.RevokedPartitions:
				if err := c.handlePartitionRevocation(ctx, e.Partitions); err != nil {
					c.logger.ErrorContext(ctx, "failed to handle partition revocation", "error", err)
					// Continue processing, but log the error
				}
			case kafka.Error:
				attrs := []any{
					slog.Int("code", int(e.Code())),
					slog.String("error", e.Error()),
				}

				// Log Kafka client "local" errors on warning level
				if e.Code() <= -100 {
					c.logger.WarnContext(ctx, "kafka local error", attrs...)
				} else {
					c.logger.ErrorContext(ctx, "kafka broker error", attrs...)
				}

				// If it's a fatal error, stop consuming
				if e.IsFatal() {
					c.logger.ErrorContext(ctx, "fatal kafka error, stopping consumer", "error", e)
					return
				}
			case kafka.OffsetsCommitted:
				c.logger.DebugContext(ctx, "kafka offset committed", "offsets", e.Offsets)
			default:
				// Ignore other event types
			}
		}
	}
}

// Close closes the consumer and releases resources.
func (c *consumer) Close() error {
	if !c.isClosed.CompareAndSwap(false, true) {
		return nil
	}

	c.logger.Info("closing consumer")

	// Stop all partition workers
	c.stopAllPartitionWorkers()

	// Wait for processing to finish
	c.wg.Wait()

	// Close consumer
	if err := c.consumer.Close(); err != nil {
		return fmt.Errorf("failed to close consumer: %w", err)
	}

	return nil
}

// handlePartitionAssignment starts goroutines for newly assigned partitions.
func (c *consumer) handlePartitionAssignment(ctx context.Context, partitions []kafka.TopicPartition) error {
	c.logger.InfoContext(ctx, "handling partition assignment", "partitions", prettyPartitions(partitions))

	if len(partitions) == 0 {
		return nil
	}

	// Set offset to stored (committed offset) for assigned partitions
	for i := range partitions {
		partitions[i].Offset = kafka.OffsetStored
	}

	// Assign partitions to consumer
	if err := c.consumer.IncrementalAssign(partitions); err != nil {
		return fmt.Errorf("failed to assign partitions: %w", err)
	}

	// Start worker goroutine for each assigned partition
	c.partitionMu.Lock()
	defer c.partitionMu.Unlock()

	for _, partition := range partitions {
		key := PartitionKeyFromTopicPartition(partition)

		// Skip if worker already exists
		if _, exists := c.partitionWorkers[key]; exists {
			c.logger.DebugContext(ctx, "partition worker already exists, skipping", "partition", key)
			continue
		}

		// Create context for this partition worker
		workerCtx, cancel := context.WithCancel(ctx)

		// Create channel for messages (buffered to avoid blocking the polling loop)
		msgChan := make(chan *kafka.Message, 100)

		worker := &partitionWorker{
			key:      key,
			cancel:   cancel,
			done:     make(chan struct{}),
			msgChan:  msgChan,
			shutdown: atomic.Bool{},
		}

		c.partitionWorkers[key] = worker
		c.partitionChannels[key] = msgChan

		// Start goroutine for this partition
		c.wg.Add(1)
		go c.runPartitionWorker(workerCtx, worker)
	}

	return nil
}

// handlePartitionRevocation stops goroutines for revoked partitions with clean shutdown.
func (c *consumer) handlePartitionRevocation(ctx context.Context, partitions []kafka.TopicPartition) error {
	c.logger.InfoContext(ctx, "handling partition revocation", "partitions", prettyPartitions(partitions))

	if len(partitions) == 0 {
		return nil
	}

	// Check if assignment was lost involuntarily
	if c.consumer.AssignmentLost() {
		c.logger.WarnContext(ctx, "assignment lost involuntarily, commit may fail")
	}

	// Stop worker goroutines for revoked partitions (clean shutdown)
	workersToStop, err := c.workers.StopWorkers(ctx, slicesx.Map(partitions, func(partition kafka.TopicPartition) (PartitionKey, error) {
		return PartitionKeyFromTopicPartition(partition)
	}))
	if err != nil {
		return fmt.Errorf("failed to stop workers: %w", err)
	}

	// Wait for workers to finish processing current message
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for _, worker := range workersToStop {
		select {
		case <-worker.done:
			c.logger.DebugContext(ctx, "partition worker stopped cleanly", "partition", worker.key)
		case <-shutdownCtx.Done():
			c.logger.WarnContext(ctx, "partition worker shutdown timeout, forcing stop", "partition", worker.key)
			// Close channel to force stop
			c.safeCloseChannel(worker.msgChan, c.logger)
		}
	}

	// Unassign partitions from consumer after workers have stopped
	if err := c.consumer.IncrementalUnassign(partitions); err != nil {
		return fmt.Errorf("failed to unassign partitions: %w", err)
	}

	return nil
}

type workerManager struct {
	partitionWorkers map[PartitionKey]*partitionWorker
	partitionMu      sync.RWMutex
	logger           *slog.Logger
}

func (w *workerManager) GetWorkerByPartitionKey(key PartitionKey) (*partitionWorker, bool) {
	w.partitionMu.RLock()
	defer w.partitionMu.RUnlock()
	worker, exists := w.partitionWorkers[key]
	return worker, exists
}

// routeMessage routes a message to the appropriate partition channel.
// It skips routing if the partition worker is shutting down.
func (m *workerManager) RouteMessage(ctx context.Context, msg *kafka.Message) {
	key, err := PartitionKeyFromMessage(msg)
	if err != nil {
		m.logger.ErrorContext(ctx, "received message with nil topic, dropping", "error", err)
		return
	}

	worker, exists := m.GetWorkerByPartitionKey(key)

	if !exists || worker == nil {
		m.logger.DebugContext(ctx, "no channel for partition, message may be from unassigned partition",
			"partition", key)
		return
	}

	// Don't route messages to workers that are shutting down
	if worker.shutdown.Load() {
		m.logger.DebugContext(ctx, "partition worker is shutting down, dropping message",
			"partition", key)
		return
	}

	// Try non-blocking send first
	select {
	case worker.msgChan <- msg:
		// Message routed successfully
		return
	default:
		// Channel is full, log warning and block
		m.logger.WarnContext(ctx, "partition channel full, waiting for handler to process messages",
			"partition", key)
	}

	// Block on send (wait for handler to catch up)
	select {
	case worker.msgChan <- msg:
		// Message routed successfully
	case <-ctx.Done():
		return
	}
}

func (m *workerManager) StopWorkers(ctx context.Context, keys []PartitionKey) ([]*partitionWorker, error) {
	m.partitionMu.Lock()
	defer m.partitionMu.Unlock()

	workers := make([]*partitionWorker, 0, len(keys))
	for _, key := range keys {
		worker, exists := m.GetWorkerByPartitionKey(key)
		if !exists || worker == nil {
			return nil, fmt.Errorf("worker not found for partition: %s", key.String())
		}

		workers = append(workers, worker)
	}

	for _, worker := range workers {
		worker.shutdown.Store(true)
		worker.cancel()
	}

	for _, worker := range workers {
		delete(m.partitionWorkers, worker.key)
	}

	return workers, nil
}

// partitionWorker represents a worker goroutine for a specific partition.
type partitionWorker struct {
	key      PartitionKey
	cancel   context.CancelFunc
	done     chan struct{}
	msgChan  chan *kafka.Message
	shutdown atomic.Bool // Signals that shutdown has been initiated
}

// stopAllPartitionWorkers stops all partition workers with clean shutdown.
func (c *consumer) stopAllPartitionWorkers() {
	c.partitionMu.Lock()
	workers := make([]*partitionWorker, 0, len(c.partitionWorkers))
	for key, worker := range c.partitionWorkers {
		workers = append(workers, worker)
		c.logger.Debug("initiating shutdown for partition worker", "partition", key)
	}
	c.partitionMu.Unlock()

	if len(workers) == 0 {
		return
	}

	c.logger.Info("stopping all partition workers", "count", len(workers))

	// Initiate shutdown for all workers (stop routing new messages)
	for _, worker := range workers {
		worker.shutdown.Store(true)
		worker.cancel()
	}

	// Wait for workers to finish processing current message
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, worker := range workers {
		select {
		case <-worker.done:
			c.logger.Debug("partition worker stopped cleanly", "partition", worker.key)
		case <-shutdownCtx.Done():
			c.logger.Warn("partition worker shutdown timeout, forcing stop", "partition", worker.key)
			// Close channel to force stop
			c.safeCloseChannel(worker.msgChan, c.logger)
		}
	}

	// Clean up worker references
	c.partitionMu.Lock()
	defer c.partitionMu.Unlock()

	for key := range c.partitionWorkers {
		delete(c.partitionWorkers, key)
		delete(c.partitionChannels, key)
	}
}

// runPartitionWorker runs a worker goroutine for a specific partition.
// It receives messages from the partition channel and processes them.
func (c *consumer) runPartitionWorker(ctx context.Context, worker *partitionWorker) {
	defer c.wg.Done()
	defer close(worker.done)

	partitionStr := worker.key.String()
	logger := c.logger.With("partition", partitionStr)
	logger.InfoContext(ctx, "starting partition worker")

	defer logger.InfoContext(ctx, "partition worker stopped")

	for {
		select {
		case <-ctx.Done():
			logger.DebugContext(ctx, "partition worker context canceled")
			return
		case msg, ok := <-worker.msgChan:
			if !ok {
				// Channel closed
				if worker.shutdown.Load() {
					logger.DebugContext(ctx, "partition channel closed after shutdown, worker stopping")
				} else {
					logger.DebugContext(ctx, "partition channel closed unexpectedly")
				}
				return
			}

			// Verify message belongs to this partition
			if msg.TopicPartition.Topic == nil {
				logger.ErrorContext(ctx, "received message with nil topic, skipping")
				continue
			}
			if *msg.TopicPartition.Topic != worker.key.Topic {
				logger.ErrorContext(ctx, "message topic mismatch, skipping",
					"expected", worker.key.Topic,
					"got", *msg.TopicPartition.Topic)
				continue
			}
			if int(msg.TopicPartition.Partition) != worker.key.Partition {
				logger.ErrorContext(ctx, "message partition mismatch, skipping",
					"expected", worker.key.Partition,
					"got", msg.TopicPartition.Partition)
				continue
			}

			// Process message
			if err := c.handleMessage(ctx, msg); err != nil {
				logger.ErrorContext(ctx, "failed to handle message", "error", err)
				// Continue processing other messages
				continue
			}
		}
	}
}

// safeCloseChannel safely closes a channel, recovering from panic if channel is already closed.
func (c *consumer) safeCloseChannel(ch chan *kafka.Message, logger *slog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			// Channel was already closed, which is fine
			logger.Debug("channel was already closed", "panic", r)
		}
	}()

	// Close the channel (will panic if already closed, but we recover from it)
	close(ch)
}

// prettyPartitions formats partitions for logging.
func prettyPartitions(partitions []kafka.TopicPartition) []string {
	result := make([]string, len(partitions))
	for i, p := range partitions {
		result[i] = p.String()
	}
	return result
}

// handleMessage processes a single Kafka message with retry, timeout, and DLQ support.
func (c *consumer) handleMessage(ctx context.Context, msg *kafka.Message) error {
	start := time.Now()

	// Create message context with correlation ID
	msgCtx := c.createMessageContext(ctx, msg)

	// Get event type for metrics (extract from Kafka message)
	eventType := c.handler.ExtractEventName(msg)
	meterAttributeCEType := attribute.String("message.event_type", eventType)
	if eventType == "" {
		meterAttributeCEType = attribute.String("message.event_type", "UNKNOWN")
	}

	// Create span for tracing
	span := tracex.StartWithNoValue(msgCtx, c.config.Tracer, "consumer.message_processing", trace.WithAttributes(
		meterAttributeCEType,
		attribute.String("kafka.topic", *msg.TopicPartition.Topic),
		attribute.Int("kafka.partition", int(msg.TopicPartition.Partition)),
		attribute.Int64("kafka.offset", int64(msg.TopicPartition.Offset)),
	))

	var processingErr error
	processingErr = span.Wrap(func(ctx context.Context) error {
		// Apply timeout if configured
		if c.config.ConsumerConfig.ProcessingTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, c.config.ConsumerConfig.ProcessingTimeout)
			defer cancel()
		}

		// Process with retry
		return c.processWithRetry(ctx, msg)
	})

	// Record metrics
	if processingErr != nil {
		c.messageProcessingCount.Add(msgCtx, 1, metric.WithAttributes(
			meterAttributeCEType,
			attribute.String("status", "failed"),
		))

		// Send to DLQ (mandatory)
		if !c.isClosed.Load() {
			if err := c.sendToDLQ(msgCtx, msg, processingErr); err != nil {
				c.logger.ErrorContext(msgCtx, "failed to send message to DLQ", "error", err)
			} else {
				c.dlqMessageCount.Add(msgCtx, 1, metric.WithAttributes(meterAttributeCEType))
			}
		}

	} else {
		c.messageProcessingCount.Add(msgCtx, 1, metric.WithAttributes(
			meterAttributeCEType,
			attribute.String("status", "success"),
		))
	}

	// Commit after sending to DLQ to avoid reprocessing
	if _, err := c.consumer.CommitMessage(msg); err != nil {
		c.logger.WarnContext(msgCtx, "failed to commit message after DLQ", "error", err)
	}

	c.messageProcessingTime.Record(msgCtx, time.Since(start).Milliseconds(), metric.WithAttributes(
		meterAttributeCEType,
		attribute.String("status", lo.Ternary(processingErr != nil, "failed", "success")),
	))

	return processingErr
}

// processWithRetry processes a message with retry logic.
func (c *consumer) processWithRetry(ctx context.Context, kafkaMsg *kafka.Message) error {
	maxRetries := c.config.ConsumerConfig.Retry.MaxRetries
	if maxRetries == 0 {
		// No retries, process once
		return c.processMessage(ctx, kafkaMsg)
	}

	attempt := 0

	// Create retry context with MaxElapsedTime if configured
	retryCtx := ctx
	if c.config.ConsumerConfig.Retry.MaxElapsedTime > 0 {
		var cancel context.CancelFunc
		retryCtx, cancel = context.WithTimeout(ctx, c.config.ConsumerConfig.Retry.MaxElapsedTime)
		defer cancel()
	}

	retryOptions := []retry.Option{
		retry.Attempts(uint(maxRetries + 1)), // +1 for initial attempt
		retry.Delay(c.config.ConsumerConfig.Retry.InitialInterval),
		retry.MaxDelay(c.config.ConsumerConfig.Retry.MaxInterval),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.Context(retryCtx),
		retry.OnRetry(func(n uint, err error) {
			attempt = int(n)
			c.logger.WarnContext(ctx, "retrying message processing",
				"attempt", n+1,
				"max_retries", maxRetries,
				"error", err,
			)
		}),
	}

	err := retry.Do(
		func() error {
			return c.processMessage(ctx, kafkaMsg)
		},
		retryOptions...,
	)
	if err != nil {
		return fmt.Errorf("failed after %d attempts: %w", attempt+1, err)
	}

	return nil
}

// processMessage processes a single message through the handler chain.
func (c *consumer) processMessage(ctx context.Context, kafkaMsg *kafka.Message) error {
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			// Extract message ID for logging
			msgID := ""
			eventType := ""
			for _, h := range kafkaMsg.Headers {
				if h.Key == marshaler.CloudEventsHeaderSource {
					msgID = string(h.Value)
				}

				if h.Key == marshaler.CloudEventsHeaderType {
					eventType = string(h.Value)
				}
			}

			if msgID == "" {
				msgID = fmt.Sprintf("%s-%d-%d", *kafkaMsg.TopicPartition.Topic, kafkaMsg.TopicPartition.Partition, kafkaMsg.TopicPartition.Offset)
			}

			c.logger.ErrorContext(ctx, "panic recovered in message processing",
				"panic", r,
				"message_id", msgID,
				"message_topic", lo.FromPtr(kafkaMsg.TopicPartition.Topic),
				"message_partition", int(kafkaMsg.TopicPartition.Partition),
				"message_offset", int64(kafkaMsg.TopicPartition.Offset),
				"message_event_type", eventType,
			)
		}
	}()

	if kafkaMsg == nil {
		return errors.New("kafka message is nil")
	}

	// Process through handler
	return c.handler.Handle(ctx, kafkaMsg)
}

// sendToDLQ sends a failed message to the dead letter queue.
// DLQ is mandatory, so this function always sends messages to DLQ.
func (c *consumer) sendToDLQ(ctx context.Context, kafkaMsg *kafka.Message, processingErr error) error {
	if c.producer == nil {
		return errors.New("producer is required for DLQ")
	}

	// Use old message body for DLQ (just forward the failed message as-is)
	dlqMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &c.config.ConsumerConfig.DLQ.Topic,
			Partition: kafka.PartitionAny,
		},
		Value: kafkaMsg.Value,
		Headers: []kafka.Header{
			{Key: "original_topic", Value: []byte(*kafkaMsg.TopicPartition.Topic)},
			{Key: "original_partition", Value: []byte(fmt.Sprintf("%d", kafkaMsg.TopicPartition.Partition))},
			{Key: "original_offset", Value: []byte(fmt.Sprintf("%d", kafkaMsg.TopicPartition.Offset))},
			{Key: "error", Value: []byte(processingErr.Error())},
		},
	}

	// Copy original headers
	for _, h := range kafkaMsg.Headers {
		dlqMsg.Headers = append(dlqMsg.Headers, kafka.Header{
			Key:   "original_" + h.Key,
			Value: h.Value,
		})
	}

	// Produce to DLQ
	deliveryChan := make(chan kafka.Event, 1)
	if err := c.producer.Produce(dlqMsg, deliveryChan); err != nil {
		return fmt.Errorf("failed to produce to DLQ: %w", err)
	}

	// Wait for delivery confirmation
	select {
	case e := <-deliveryChan:
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				return fmt.Errorf("DLQ delivery failed: %w", ev.TopicPartition.Error)
			}
		case kafka.Error:
			return fmt.Errorf("DLQ delivery error: %w", ev)
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(defaultDLQDeliveryTimeout):
		return errors.New("DLQ delivery timeout")
	}

	return nil
}

// createMessageContext creates a context for message processing with correlation ID.
func (c *consumer) createMessageContext(ctx context.Context, msg *kafka.Message) context.Context {
	// Extract correlation ID from headers (check both watermill and standard formats)
	correlationID := ""
	for _, h := range msg.Headers {
		if h.Key == "correlation_id" || h.Key == "X-Correlation-ID" || h.Key == "correlation-id" {
			correlationID = string(h.Value)
			break
		}
	}

	// Generate correlation ID if not present (use message UUID if available in headers)
	if correlationID == "" {
		for _, h := range msg.Headers {
			if h.Key == "uuid" || h.Key == "message_id" {
				correlationID = string(h.Value)
				break
			}
		}
	}

	// Fallback to generating one from topic/partition/offset
	if correlationID == "" {
		correlationID = fmt.Sprintf("%s-%d-%d", *msg.TopicPartition.Topic, msg.TopicPartition.Partition, msg.TopicPartition.Offset)
	}

	// Add correlation ID to context
	return context.WithValue(ctx, "correlation_id", correlationID)
}
