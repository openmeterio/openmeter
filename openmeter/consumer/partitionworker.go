package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type partitionWorkerConsumer interface {
	// SendToDLQ sends a message to the DLQ.
	SendToDLQ(ctx context.Context, msg *kafka.Message, processingErr error) error

	// CommitMessage commits a message.
	CommitMessage(ctx context.Context, msg *kafka.Message) error
}

type PartitionWorker interface {
	Shutdown()
	IsDone() <-chan struct{}
	ForceStop()
	EnqueueMessage(ctx context.Context, msg *kafka.Message)
}

type NewPartitionWorkerOptions struct {
	Key PartitionKey

	Handler Handler
	Config  config.ConsumerConfiguration

	Logger  *slog.Logger
	Tracer  trace.Tracer
	Metrics *consumerMetrics
}

func (o NewPartitionWorkerOptions) Validate() error {
	var errs []error
	if o.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if o.Tracer == nil {
		errs = append(errs, errors.New("tracer is required"))
	}

	if o.Metrics == nil {
		errs = append(errs, errors.New("metrics is required"))
	}

	if lo.IsEmpty(o.Key) {
		errs = append(errs, errors.New("key is required"))
	}

	return errors.Join(errs...)
}

// partitionWorker represents a worker goroutine for a specific partition.
type partitionWorker struct {
	key      PartitionKey
	cancel   context.CancelFunc
	done     chan struct{}
	msgChan  chan *kafka.Message
	shutdown atomic.Bool // Signals that shutdown has been initiated

	// Dependencies
	logger   *slog.Logger
	tracer   trace.Tracer
	metrics  *consumerMetrics
	consumer partitionWorkerConsumer
	handler  Handler
	config   config.ConsumerConfiguration
}

func NewPartitionWorker(opts NewPartitionWorkerOptions) (PartitionWorker, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	return &partitionWorker{
		key:     opts.Key,
		done:    make(chan struct{}),
		msgChan: make(chan *kafka.Message, defaultMsgChanBufferSize),

		logger:  opts.Logger,
		tracer:  opts.Tracer,
		metrics: opts.Metrics,
		handler: opts.Handler,
		config:  opts.Config,
	}, nil
}

func (w *partitionWorker) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	w.cancel = cancel
	go w.handleMessages(ctx)
}

// runPartitionWorker runs a worker goroutine for a specific partition.
// It receives messages from the partition channel and processes them.
func (w *partitionWorker) handleMessages(ctx context.Context) {
	defer close(w.done)

	for {
		select {
		case <-ctx.Done():
			w.logger.DebugContext(ctx, "partition worker context canceled")
			return
		case msg, ok := <-w.msgChan:
			if !ok {
				// Channel closed
				if w.shutdown.Load() {
					w.logger.DebugContext(ctx, "partition channel closed after shutdown, worker stopping")
				} else {
					w.logger.ErrorContext(ctx, "partition channel closed unexpectedly")
				}
				return
			}

			// Verify message belongs to this partition
			if msg.TopicPartition.Topic == nil {
				w.logger.ErrorContext(ctx, "received message with nil topic, skipping")
				continue
			}

			if *msg.TopicPartition.Topic != w.key.Topic {
				w.logger.ErrorContext(ctx, "message topic mismatch, skipping",
					"expected", w.key.Topic,
					"got", *msg.TopicPartition.Topic)
				continue
			}

			if int(msg.TopicPartition.Partition) != w.key.Partition {
				w.logger.ErrorContext(ctx, "message partition mismatch, skipping",
					"expected", w.key.Partition,
					"got", msg.TopicPartition.Partition)
				continue
			}

			// Process message
			if err := w.handleMessage(ctx, msg); err != nil {
				w.logger.ErrorContext(ctx, "failed to handle message", "error", err)
			}
		}
	}
}

func (w *partitionWorker) Shutdown() {
	w.shutdown.Store(true)
	w.cancel()
}

func (w *partitionWorker) ForceStop() {
	safeCloseChannel(w.msgChan, w.logger)
}

func (w *partitionWorker) IsDone() <-chan struct{} {
	return w.done
}

func (w *partitionWorker) EnqueueMessage(ctx context.Context, msg *kafka.Message) {
	// Try non-blocking send first
	select {
	case w.msgChan <- msg:
		// Message routed successfully
		return
	default:
		// Channel is full, log warning and block
		w.logger.WarnContext(ctx, "partition channel full, waiting for handler to process messages",
			"partition", msg.TopicPartition.String())
	}

	// Block on send (wait for handler to catch up)
	select {
	case w.msgChan <- msg:
		// Message routed successfully
	case <-ctx.Done():
	}
}

// handleMessage processes a single Kafka message with retry, timeout, and DLQ support.
func (w *partitionWorker) handleMessage(ctx context.Context, msg *kafka.Message) error {
	start := time.Now()

	// Get event type for metrics (extract from Kafka message)
	eventType := w.handler.ExtractEventName(msg)
	meterAttributeCEType := attribute.String("message.event_type", eventType)
	if eventType == "" {
		meterAttributeCEType = attribute.String("message.event_type", "UNKNOWN")
	}

	// Create span for tracing
	span := tracex.StartWithNoValue(ctx, w.tracer, "consumer.message_processing", trace.WithAttributes(
		meterAttributeCEType,
		attribute.String("kafka.topic", *msg.TopicPartition.Topic),
		attribute.Int("kafka.partition", int(msg.TopicPartition.Partition)),
		attribute.Int64("kafka.offset", int64(msg.TopicPartition.Offset)),
	))

	var processingErr error
	processingErr = span.Wrap(func(ctx context.Context) error {
		// Apply timeout if configured
		if w.config.ProcessingTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, w.config.ProcessingTimeout)
			defer cancel()
		}

		// Process with retry
		return w.processWithRetry(ctx, msg)
	})

	// Record metrics
	if processingErr != nil {
		w.metrics.messageProcessingCount.Add(ctx, 1, metric.WithAttributes(
			meterAttributeCEType,
			attribute.String("status", "failed"),
		))

		// Send to DLQ (mandatory)
		if err := w.consumer.SendToDLQ(ctx, msg, processingErr); err != nil {
			w.logger.ErrorContext(ctx, "failed to send message to DLQ", "error", err)
		}
		w.metrics.dlqMessageCount.Add(ctx, 1, metric.WithAttributes(meterAttributeCEType))

	} else {
		w.metrics.messageProcessingCount.Add(ctx, 1, metric.WithAttributes(
			meterAttributeCEType,
			attribute.String("status", "success"),
		))
	}

	// Commit after sending to DLQ to avoid reprocessing
	if err := w.consumer.CommitMessage(ctx, msg); err != nil {
		w.logger.ErrorContext(ctx, "failed to commit message after DLQ", "error", err)
	}

	w.metrics.messageProcessingTime.Record(ctx, time.Since(start).Milliseconds(), metric.WithAttributes(
		meterAttributeCEType,
		attribute.String("status", lo.Ternary(processingErr != nil, "failed", "success")),
	))

	return processingErr
}

// processWithRetry processes a message with retry logic.
func (w *partitionWorker) processWithRetry(ctx context.Context, kafkaMsg *kafka.Message) error {
	maxRetries := w.config.Retry.MaxRetries
	if maxRetries == 0 {
		// No retries, process once
		return w.processMessage(ctx, kafkaMsg)
	}

	attempt := 0

	// Create retry context with MaxElapsedTime if configured
	retryCtx := ctx
	if w.config.Retry.MaxElapsedTime > 0 {
		var cancel context.CancelFunc
		retryCtx, cancel = context.WithTimeout(ctx, w.config.Retry.MaxElapsedTime)
		defer cancel()
	}

	retryOptions := []retry.Option{
		retry.Attempts(uint(maxRetries + 1)), // +1 for initial attempt
		retry.Delay(w.config.Retry.InitialInterval),
		retry.MaxDelay(w.config.Retry.MaxInterval),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.Context(retryCtx),
		retry.OnRetry(func(n uint, err error) {
			attempt = int(n)
			w.logger.WarnContext(ctx, "retrying message processing",
				"attempt", n+1,
				"max_retries", maxRetries,
				"error", err,
			)
		}),
	}

	err := retry.Do(
		func() error {
			return w.processMessage(ctx, kafkaMsg)
		},
		retryOptions...,
	)
	if err != nil {
		return fmt.Errorf("failed after %d attempts: %w", attempt+1, err)
	}

	return nil
}

// processMessage processes a single message through the handler chain.
func (w *partitionWorker) processMessage(ctx context.Context, kafkaMsg *kafka.Message) error {
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

			w.logger.ErrorContext(ctx, "panic recovered in message processing",
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
	return w.handler.Handle(ctx, kafkaMsg)
}
