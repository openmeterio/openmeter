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

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// Config contains the configuration for the consumer.
type EnvironmentConfig struct {
	Consumer *kafka.Consumer
	Producer *kafka.Producer

	ConsumerConfig config.ConsumerConfiguration

	Logger      *slog.Logger
	MetricMeter metric.Meter
	Tracer      trace.Tracer

	// PollTimeout is the maximum time to wait for messages when polling.
	// If 0, defaults to 100ms.
	PollTimeout time.Duration
}

func (c *EnvironmentConfig) Validate() error {
	if c.Consumer == nil {
		return errors.New("consumer is required")
	}

	if c.Producer == nil {
		return errors.New("producer is required for DLQ")
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

type Config struct {
	Environment EnvironmentConfig
	Topics      []string
	Handler     Handler
}

func (c *Config) Validate() error {
	var errs []error

	if c.Handler == nil {
		errs = append(errs, errors.New("handler is required"))
	}

	if err := c.Environment.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("environment config: %w", err))
	}

	if len(c.Topics) == 0 {
		errs = append(errs, errors.New("at least one topic is required"))
	}

	return errors.Join(errs...)
}

type Handler interface {
	Handle(ctx context.Context, msg *kafka.Message) error
	ExtractEventName(msg *kafka.Message) string
}

// Consumer is the interface for a librdkafka-based consumer that processes messages from Kafka topics.
// It uses one goroutine per assigned partition and provides retry, DLQ, and eventbus callback support.
type Consumer interface {
	// Run starts consuming messages. It uses a single polling goroutine that routes messages
	// to dedicated partition processing goroutines.
	Run(ctx context.Context) error

	// Close closes the consumer and releases resources.
	Close() error
}

// consumer is a librdkafka-based consumer that processes messages from Kafka topics.
// It uses one goroutine per assigned partition and provides retry, DLQ, and eventbus callback support.
type consumer struct {
	consumer    *kafka.Consumer
	producer    *kafka.Producer
	pollTimeout time.Duration
	dlqTopic    string

	consumerConfig config.ConsumerConfiguration
	logger         *slog.Logger

	handler Handler

	metrics consumerMetrics

	// State
	isRunning atomic.Bool
	isClosed  atomic.Bool
	wg        sync.WaitGroup // TODO: Use for children too!

	partitionWorkers map[PartitionKey]PartitionWorker
}

var _ partitionWorkerConsumer = (*consumer)(nil)

type consumerMetrics struct {
	messageProcessingCount metric.Int64Counter
	messageProcessingTime  metric.Int64Histogram
	dlqMessageCount        metric.Int64Counter
}

// New creates a new Consumer instance.
func New(cfg Config) (Consumer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	pollTimeout := cfg.Environment.PollTimeout
	if pollTimeout == 0 {
		pollTimeout = defaultPollTimeout
	}

	envCfg := cfg.Environment

	// Initialize metrics
	messageProcessingCount, err := envCfg.MetricMeter.Int64Counter(
		"consumer.message_processing_count",
		metric.WithDescription("Number of messages processed"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message processing count metric: %w", err)
	}

	messageProcessingTime, err := envCfg.MetricMeter.Int64Histogram(
		"consumer.message_processing_time_ms",
		metric.WithDescription("Time spent processing a message (including retries)"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message processing time metric: %w", err)
	}

	dlqMessageCount, err := envCfg.MetricMeter.Int64Counter(
		"consumer.dlq_message_count",
		metric.WithDescription("Number of messages sent to DLQ"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DLQ message count metric: %w", err)
	}

	c := &consumer{
		consumer:       envCfg.Consumer,
		producer:       envCfg.Producer,
		consumerConfig: envCfg.ConsumerConfig,
		logger:         envCfg.Logger,
		handler:        cfg.Handler,
		dlqTopic:       envCfg.ConsumerConfig.DLQ.Topic,
		pollTimeout:    pollTimeout,
		metrics: consumerMetrics{
			messageProcessingCount: messageProcessingCount,
			messageProcessingTime:  messageProcessingTime,
			dlqMessageCount:        dlqMessageCount,
		},
		partitionWorkers: make(map[PartitionKey]PartitionWorker, defaultPartitionWorkerMapSize),
	}

	topics := lo.Uniq(cfg.Topics)

	// Subscribe to topics:
	// TODO: maybe move it to the Run?
	if err := c.consumer.SubscribeTopics(topics, nil); err != nil {
		return nil, fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	c.logger.Info("consumer initialized", "topics", topics)

	return c, nil
}

// Run starts consuming messages. It uses a single polling goroutine that routes messages
// to dedicated partition processing goroutines.
func (c *consumer) Run(ctx context.Context) error {
	if !c.isRunning.CompareAndSwap(false, true) {
		return errors.New("consumer is already running")
	}

	defer c.isRunning.Store(false)

	c.logger.InfoContext(ctx, "starting consumer")

	// Start single polling goroutine
	c.wg.Add(1)
	go c.runPollingLoop(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	c.logger.InfoContext(ctx, "consumer context canceled, stopping all partition workers")
	c.stopWorkers(ctx, stopWorkersInput{
		keys:        lo.Keys(c.partitionWorkers),
		waitTimeout: defaultStopWorkersTimeout,
	})

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
			ev := c.consumer.Poll(int(c.pollTimeout.Milliseconds()))
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				// Route message to partition-specific channel
				// TODO: Once we have removed watermill, let's filter messages early based on type.
				c.handleMessage(ctx, e)
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
					// TODO: terminate the whole execution flow
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

	// Stop all partition workers (we are using context.Background() as the main context is already cancelled and we have a timeout set)
	err := c.stopWorkers(context.Background(), stopWorkersInput{
		keys:        lo.Keys(c.partitionWorkers),
		waitTimeout: defaultStopWorkersTimeout,
	})
	if err != nil {
		return fmt.Errorf("failed to stop workers: %w", err)
	}

	// Wait for processing to finish
	c.wg.Wait()

	// Close consumer
	if err := c.consumer.Close(); err != nil {
		return fmt.Errorf("failed to close consumer: %w", err)
	}

	return nil
}

func (c *consumer) CommitMessage(ctx context.Context, msg *kafka.Message) error {
	if _, err := c.consumer.CommitMessage(msg); err != nil {
		return fmt.Errorf("failed to commit message: %w", err)
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
	for _, partition := range partitions {
		key, err := PartitionKeyFromTopicPartition(partition)
		if err != nil {
			return fmt.Errorf("failed to map partition to key: %w", err)
		}

		err = c.startWorkerForPartition(ctx, key)
		if err != nil {
			// TODO: continue?!
			return fmt.Errorf("failed to start worker for partition: %w", err)
		}
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

	partitionsToStop, err := slicesx.MapWithErr(partitions, func(partition kafka.TopicPartition) (PartitionKey, error) {
		return PartitionKeyFromTopicPartition(partition)
	})
	if err != nil {
		return fmt.Errorf("failed to map partitions to keys: %w", err)
	}

	// Stop worker goroutines for revoked partitions (clean shutdown)
	err = c.stopWorkers(ctx, stopWorkersInput{
		keys:        partitionsToStop,
		waitTimeout: defaultStopWorkersTimeout,
	})
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to stop workers by partition key, continuing with unassignment", "error", err)
	}

	// Unassign partitions from consumer after workers have stopped
	if err := c.consumer.IncrementalUnassign(partitions); err != nil {
		return fmt.Errorf("failed to unassign partitions: %w", err)
	}

	return nil
}

// handleMessage routes a message to the appropriate partition channel.
// It skips routing if the partition worker is shutting down.
func (c *consumer) handleMessage(ctx context.Context, msg *kafka.Message) {
	key, err := PartitionKeyFromMessage(msg)
	if err != nil {
		c.logger.ErrorContext(ctx, "received message with nil topic, dropping", "error", err)
		return
	}

	worker, exists := c.partitionWorkers[key]
	if !exists || worker == nil {
		c.logger.ErrorContext(ctx, "no worker for partition, message may be from unassigned partition",
			"partition", key.String())
		return
	}

	worker.EnqueueMessage(ctx, msg)
}

type stopWorkersInput struct {
	keys        []PartitionKey
	waitTimeout time.Duration
}

func (o stopWorkersInput) Validate() error {
	var errs []error
	if len(o.keys) == 0 {
		errs = append(errs, errors.New("keys are required"))
	}

	if o.waitTimeout == 0 {
		errs = append(errs, errors.New("wait timeout is required"))
	}

	return errors.Join(errs...)
}

// stopWorkers stops the workers for the given partition keys, and returns the workers that received a shutdown signal.
// workers might not be stopped immediately, use the WaitForWorkersToFinish method to wait for them to finish.
func (c *consumer) stopWorkers(ctx context.Context, input stopWorkersInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("failed to validate stop workers input: %w", err)
	}

	workersToStop := make(map[PartitionKey]PartitionWorker, len(input.keys))
	for _, key := range input.keys {
		worker, exists := c.partitionWorkers[key]
		if !exists || worker == nil {
			c.logger.ErrorContext(ctx, "worker not found for partition - ignoring", "partition", key.String())
			continue
		}

		workersToStop[key] = worker
	}

	for key, worker := range workersToStop {
		worker.Shutdown()
		delete(c.partitionWorkers, key)
	}

	// Let's wait for the workers to actually stop (given the context of the worker is cancelled this should be almost instant)
	shutdownCtx, cancel := context.WithTimeout(ctx, input.waitTimeout)
	defer cancel()

	for key, worker := range workersToStop {
		select {
		case <-worker.IsDone():
			c.logger.DebugContext(ctx, "partition worker stopped cleanly", "partition", key.String())
		case <-shutdownCtx.Done():
			c.logger.WarnContext(ctx, "partition worker shutdown timeout, forcing stop", "partition", key.String())
			worker.ForceStop()
		}
	}

	return nil
}

func (c *consumer) startWorkerForPartition(ctx context.Context, key PartitionKey) error {
	// Skip if worker already exists
	if _, exists := c.partitionWorkers[key]; exists {
		c.logger.DebugContext(ctx, "partition worker already exists, skipping", "partition", key)
		return nil
	}

	// TODO: Use somethin like NewWorker....

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

	// Start goroutine for this partition
	c.wg.Add(1)
	go worker.Run(workerCtx)

	return nil
}

// sendToDLQ sends a failed message to the dead letter queue.
// DLQ is mandatory, so this function always sends messages to DLQ.
func (c *consumer) SendToDLQ(ctx context.Context, kafkaMsg *kafka.Message, processingErr error) error {
	if c.producer == nil {
		return errors.New("producer is required for DLQ")
	}

	if c.isClosed.Load() {
		// We are not sending to DLQ if the consumer is closed to prevent context cancellation errors appearing on the DLQ.
		return nil
	}

	// Use old message body for DLQ (just forward the failed message as-is)
	dlqMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &c.dlqTopic,
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
