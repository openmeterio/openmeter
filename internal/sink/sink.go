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

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/openmeterio/openmeter/internal/dedupe"
	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
)

var namespaceTopicRegexp = regexp.MustCompile(`^om_([A-Za-z0-9]+(?:_[A-Za-z0-9]+)*)_events$`)
var defaultDeadletterTopicTemplate = "om_%s_events_deadletter"

type SinkMessage struct {
	Namespace    string
	KafkaMessage *kafka.Message
	Serialized   *serializer.CloudEventsKafkaPayload
	Error        *ProcessingError
}

type Sink struct {
	consumer         *kafka.Consumer
	producer         *kafka.Producer
	config           *SinkConfig
	running          bool
	buffer           SinkBuffer
	flushTimer       *time.Timer
	namespaceStore   *NamespaceStore
	namespaceRefetch *time.Timer
}

type SinkConfig struct {
	Context                 context.Context
	Logger                  *slog.Logger
	Storage                 Storage
	Deduplicator            dedupe.Deduplicator
	ConsumerKafkaConfig     kafka.ConfigMap
	ProducerKafkaConfig     kafka.ConfigMap
	MinCommitCount          int
	MaxCommitWait           time.Duration
	NamespaceRefetch        time.Duration
	DeadletterTopicTemplate string
}

func NewSink(config *SinkConfig) (*Sink, error) {
	// These are Kafka configs but also related to sink logic
	_ = config.ConsumerKafkaConfig.SetKey("session.timeout.ms", 6000)
	_ = config.ConsumerKafkaConfig.SetKey("enable.auto.commit", false)
	_ = config.ConsumerKafkaConfig.SetKey("enable.auto.offset.store", false)
	_ = config.ConsumerKafkaConfig.SetKey("go.application.rebalance.enable", true)

	consumer, err := kafka.NewConsumer(&config.ConsumerKafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %s", err)
	}

	producer, err := kafka.NewProducer(&config.ProducerKafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	if config.MinCommitCount == 0 {
		config.MinCommitCount = 1
	}
	if config.MaxCommitWait == 0 {
		config.MaxCommitWait = 1 * time.Second
	}
	if config.NamespaceRefetch == 0 {
		config.NamespaceRefetch = 15 * time.Second
	}
	if config.DeadletterTopicTemplate == "" {
		config.DeadletterTopicTemplate = defaultDeadletterTopicTemplate
	}

	sink := &Sink{
		consumer:       consumer,
		producer:       producer,
		config:         config,
		buffer:         NewSinkBuffer(),
		namespaceStore: NewNamespaceStore(),
	}

	return sink, nil
}

func (s *Sink) flush() error {
	logger := s.config.Logger.With("sink", "flush")
	messages := s.buffer.Dequeue()

	// Stop polling new messages from Kafka until we finalize batch
	s.clearFlushTimer()

	// Dedupe messages so if we have multiple messages in the same batch
	dedupedMessages := dedupeSinkMessages(messages)

	// 1. Sink to Storage
	if len(dedupedMessages) > 0 {
		logger.Debug("started to flush", "buffer size", len(dedupedMessages))

		deadletterMessages := []SinkMessage{}
		batchesPerNamespace := map[string][]SinkMessage{}

		// Insert per namespace
		for _, message := range dedupedMessages {
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
					return fmt.Errorf("unknown error type: %s", message.Error)
				}
			}

			batchesPerNamespace[message.Namespace] = append(batchesPerNamespace[message.Namespace], message)
		}

		for namespace, batch := range batchesPerNamespace {
			list := []*serializer.CloudEventsKafkaPayload{}
			for _, message := range batch {
				list = append(list, message.Serialized)
			}

			err := s.config.Storage.BatchInsert(s.config.Context, namespace, list)
			if err != nil {
				// Note: a single error in batch will make the whole batch fail
				if perr, ok := err.(*ProcessingError); ok {
					switch perr.ProcessingControl {
					case DEADLETTER:
						deadletterMessages = append(deadletterMessages, batch...)
					case DROP:
						continue
					default:
						return fmt.Errorf("unknown error type: %s", err)
					}
				}

				// Throwing and error means we will retry the whole batch again
				return fmt.Errorf("failed to sink to storage: %s", err)
			}
			logger.Debug("succeeded to sink to storage", "buffer size", len(messages))
		}

		if len(deadletterMessages) > 0 {
			s.deadLetter(deadletterMessages...)
		}
	}

	// 2. Commit Offset to Kafka
	// Least once guarantee, if offset commit fails we will potentially process the same messages again as they are not committed yet
	// If Redis write succeeds but Kafka commit fails we will reprocess messages without side effect as they will be dropped at duplicate check
	// If Redis write fails we will double write them to ClickHouse and we have to relay on ClickHouse's deduplication
	var offsets []kafka.TopicPartition
	for _, message := range messages {
		offsets = append(offsets, message.KafkaMessage.TopicPartition)
	}
	commitedOffsets, err := s.consumer.CommitOffsets(offsets)
	if err != nil {
		logger.Error("failed to commit offset to kafka", "err", err)
		// do not return error here as we want to write Redis even if commit fails as we already saved them in ClickHouse
	}
	logger.Debug("succeeded to commit offset to kafka", "offsets", commitedOffsets)

	// 3. Sink to Redis
	// Least once guarantee, if Redis write fails we will accept messages with same idempotency key in future and
	// we have to relay on ClickHouse's deduplication.
	if len(dedupedMessages) > 0 {
		dedupeItems := []dedupe.Item{}
		for _, message := range dedupedMessages {
			dedupeItems = append(dedupeItems, dedupe.Item{
				Namespace: message.Namespace,
				ID:        message.Serialized.Id,
				Source:    message.Serialized.Source,
			})
		}

		err := s.config.Deduplicator.Set(s.config.Context, dedupeItems...)
		if err != nil {
			logger.Error("failed to sink to redis", "err", err)
			return fmt.Errorf("failed to sink to redis: %s", err)
		}
		logger.Debug("succeeded to sink to redis", "buffer size", len(messages))
	}

	// 4. Reset states for next batch
	s.setFlushTimer()

	logger.Debug("succeeded to flush", "buffer size", len(messages))

	return nil
}

// deadLetter sends a message to the dead letter queue, useful permanent non-recoverable errors like json parsing
func (s *Sink) deadLetter(messages ...SinkMessage) error {
	logger := s.config.Logger.With("sink", "deadLetter")

	for _, message := range messages {
		topic := fmt.Sprintf(s.config.DeadletterTopicTemplate, message.Namespace)
		headers := message.KafkaMessage.Headers
		headers = append(headers, kafka.Header{Key: "error", Value: []byte(message.Error.Error())})

		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Timestamp:      message.KafkaMessage.Timestamp,
			Headers:        headers,
			Key:            message.KafkaMessage.Key,
			Value:          message.KafkaMessage.Value,
		}

		err := s.producer.Produce(msg, nil)
		if err != nil {
			return fmt.Errorf("producing kafka message to deadletter topic: %w", err)
		}
	}

	logger.Debug("succeeded to deadletter", "messages", len(messages))
	return nil
}

func (s *Sink) getNamespaces() (*NamespaceStore, error) {
	meters, err := s.config.Storage.GetMeters(s.config.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to get meters: %s", err)
	}

	namespaceStore := NewNamespaceStore()
	for _, meter := range meters {
		namespaceStore.AddMeter(meter.Namespace, meter)
	}

	return namespaceStore, nil
}

func (s *Sink) subscribeToNamespaces() error {
	ns, err := s.getNamespaces()
	if err != nil {
		return fmt.Errorf("failed to get namespaces: %s", err)
	}
	s.namespaceStore = ns

	// We always subscribe to all namespaces as consumer.SubscribeTopics replaces the current subscription.
	topics := getTopics(*s.namespaceStore)
	err = s.consumer.SubscribeTopics(topics, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topics: %s", err)
	}
	if err != nil {
		return fmt.Errorf("failed to subscribe to topics: %s", err)
	}

	return nil
}

// Periodically flush even if MinCommitCount threshold not reached yet but MaxCommitWait time elapsed
func (s *Sink) setFlushTimer() {
	logger := s.config.Logger.With("sink", "flush timer")

	flush := func() {
		logger.Debug("flush timer elapsed", "buffer size", s.buffer.Size())

		if s.buffer.Size() > 0 {
			err := s.flush()
			if err != nil {
				// TODO: should we panic?
				logger.Error("failed to flush", "err", err)
			}
		}
	}

	// Schedule flush
	s.flushTimer = time.AfterFunc(s.config.MaxCommitWait, flush)
}

func (s *Sink) clearFlushTimer() {
	if s.flushTimer != nil {
		s.flushTimer.Stop()
	}
}

// Run starts the Kafka consumer and sinks the events to Clickhouse
func (s *Sink) Run() error {
	logger := s.config.Logger.With("sink", "run")

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
	s.setFlushTimer()

	for s.running {
		select {
		case sig := <-sigchan:
			logger.Error("caught signal", "sig", sig)
			s.running = false
		default:
			ev := s.consumer.Poll(100)
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

				s.buffer.Add(sinkMessage)
				logger.Debug("event added to buffer", "event", kafkaCloudEvent)

				// Store message, this won't commit offset immediately just store it for the next manual commit
				_, err = s.consumer.StoreMessage(e)
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

			case kafka.AssignedPartitions:
				logger.Info("kafka assigned partitions", "event", e)
				err := s.consumer.Assign(e.Partitions)
				if err != nil {
					return fmt.Errorf("failed to assign partitions: %w", err)
				}
			case kafka.RevokedPartitions:
				logger.Info("kafka revoked partitions", "event", e)
				err := s.consumer.Unassign()
				if err != nil {
					return fmt.Errorf("failed to unassign partitions: %w", err)
				}
			case kafka.Error:
				// Errors should generally be considered
				// informational, the client will try to
				// automatically recover.
				logger.Error("kafka error", "code", e.Code(), "event", e)

				// But in this example we choose to terminate
				// the application if all brokers are down.
				if e.Code() == kafka.ErrAllBrokersDown {
					// TODO: should we panic? or what will restart polling?
					s.running = false
				}
			case kafka.OffsetsCommitted:
				// do nothing, this is an ack of the periodic offset commit
				logger.Debug("kafka offset committed", "offset", e.Offsets)
			default:
				logger.Debug("kafka ignored event", "event", e)
			}
		}
	}

	return nil
}

func (s *Sink) ParseMessage(e *kafka.Message) (string, *serializer.CloudEventsKafkaPayload, error) {
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
	isUnique, err := s.config.Deduplicator.IsUnique(s.config.Context, dedupe.Item{
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

	// Validation
	err = s.namespaceStore.validateEvent(s.config.Context, kafkaCloudEvent, namespace)
	return namespace, &kafkaCloudEvent, err
}

func (s *Sink) Close() error {
	if s.namespaceRefetch != nil {
		s.namespaceRefetch.Stop()
	}
	if s.flushTimer != nil {
		s.flushTimer.Stop()
	}
	return s.consumer.Close()
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
