package sink

// TODO: make thread safe: buffer, messageCount, running, lastSink
// TODO: flush after MaxCommitWait time selapsed even if MinCommitCount threshold not reached yet

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/pkg/models"
)

var namespaceTopicRegexp = regexp.MustCompile(`^om_([A-Za-z0-9]+(?:_[A-Za-z0-9]+)*)_events$`)

type Sink struct {
	consumer *kafka.Consumer
	config   *SinkConfig
	state    *SinkState
}

type SinkState struct {
	running      bool
	messageCount int
	buffer       []serializer.CloudEventsKafkaPayload
	lastSink     time.Time
}

type SinkConfig struct {
	Context        context.Context
	SinkStore      SinkStore
	Storage        Storage
	Dedupe         Dedupe
	KafkaConfig    kafka.ConfigMap
	MinCommitCount int
	MaxCommitWait  time.Duration
	EventsTopics   []string
	Meters         []*models.Meter
}

func NewSink(config *SinkConfig) (*Sink, error) {
	// TODO: where to set these?
	// These are Kafka configs but also related to sink logic
	_ = config.KafkaConfig.SetKey("group.id", "om-ch-sink-v1")
	_ = config.KafkaConfig.SetKey("session.timeout.ms", 6000)
	_ = config.KafkaConfig.SetKey("auto.offset.reset", "latest")
	_ = config.KafkaConfig.SetKey("enable.auto.commit", false)
	_ = config.KafkaConfig.SetKey("enable.auto.offset.store", true)
	_ = config.KafkaConfig.SetKey("go.application.rebalance.enable", true)

	consumer, err := kafka.NewConsumer(&config.KafkaConfig)
	if err != nil {
		return nil, err
	}

	if config.MinCommitCount == 0 {
		config.MinCommitCount = 1
	}
	if config.MaxCommitWait == 0 {
		config.MaxCommitWait = 1 * time.Second
	}

	sink := &Sink{
		consumer: consumer,
		config:   config,
		state: &SinkState{
			messageCount: 0,
			buffer:       []serializer.CloudEventsKafkaPayload{},
			lastSink:     time.Now(),
		},
	}

	return sink, nil
}

func (s *Sink) flush() error {
	logger := slog.With("sink", "operation", "flush")
	logger.Debug("started to flush", "buffer size", len(s.state.buffer))

	// Stop polling new messages from Kafka until we finalize batch
	s.state.running = false

	// 1. Dedupe inside bufffer
	// We filter out duplicates in the case the same batch had multiple messages for the same key.
	// This is needed as although we check Redis for every single Kafka poll we only write to Redis in batches at the end of processing.
	dedupedBuffer := dedupeEventList(s.state.buffer)

	// 2. Sink to Storage
	if len(s.state.buffer) > 0 {
		// TODO: should we insert per namespace?
		// Check out how https://github.com/ClickHouse/clickhouse-kafka-connect does
		insertErr := s.config.Storage.BatchInsert(s.config.Context, "TODO", dedupedBuffer)
		if insertErr != nil {
			switch insertErr.ProcessingControl {
			case DEADLETTER:
				// TODO dead letter
			}
			// Note: a single error in batch will make the whole batch fail

			// Throwing and error means we will retry the whole batch again
			return fmt.Errorf("failed to sink to storage: %s", insertErr)
		}
		logger.Debug("succeeded to sink to storage", "buffer size", len(dedupedBuffer))
	}

	// 3. Commit Offset to Kafka
	// Least once guarantee, if offset commit fails we will potentially process the same messages again as they are not committed yet
	// If Redis write succeeds but Kafka commit fails we will reprocess messages without side effect as they will be dropped at duplicate check
	// If Redis write fails we will double write them to ClickHouse and we have to relay on ClickHouse's deduplication
	commitedOffsets, err := s.consumer.Commit()
	if err != nil {
		logger.Error("failed to commit offset to kafka", "err", err)
		// do not return error here as we want to write Redis even if commit fails as we already saved them in ClickHouse
	}
	logger.Debug("succeeded to commit offset to kafka", "offsets", commitedOffsets)

	// 4. Sink to Redis
	// Least once guarantee, if Redis write fails we will accept messages with same idempotency key in future and
	// we have to relay on ClickHouse's deduplication.
	if len(dedupedBuffer) > 0 {
		err := s.config.Dedupe.Set(s.config.Context, dedupedBuffer...)
		if err != nil {
			logger.Error("failed to sink to redis", "err", err, "rows", dedupedBuffer)
			return NewProcessingError(fmt.Sprintf("failed to sink to redis: %s", err), RETRY)
		}
		logger.Debug("succeeded to sink to redis", "buffer size", len(dedupedBuffer))
	}

	// 5. Reset states for next batch
	s.state.lastSink = time.Now()
	s.state.buffer = []serializer.CloudEventsKafkaPayload{}
	s.state.messageCount = 0
	s.state.running = true

	logger.Debug("succeeded to flush", "buffer size", len(dedupedBuffer))

	return nil
}

// deadLetter sends a message to the dead letter queue, useful permanent non-recoverable errors like json parsing
func (s *Sink) deadLetter(message string) error {
	// TODO: implement
	slog.Debug("todo: dead letter", "message", message)
	return nil
}

// Run starts the Kafka consumer and sinks the events to Clickhouse
func (s *Sink) Run() error {
	logger := slog.With("sink", "run")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	err := s.consumer.SubscribeTopics(s.config.EventsTopics, nil)
	if err != nil {
		return err
	}

	s.state.lastSink = time.Now()
	s.state.running = true

	for s.state.running {
		select {
		case sig := <-sigchan:
			logger.Error("caught signal", "sig", sig)
			s.state.running = false
		default:
			ev := s.consumer.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				kafkaMessage := string(e.Value)
				s.state.messageCount++

				// Parse Kafka Event
				var kafkaCloudEvent serializer.CloudEventsKafkaPayload
				err := json.Unmarshal(e.Value, &kafkaCloudEvent)
				if err != nil {
					logger.Error("faield to json parse kafka message", "err", err, "message", kafkaMessage)
					err := s.deadLetter(kafkaMessage)
					if err != nil {
						logger.Error("failed to dead letter message", "err", err, "message", kafkaMessage)
						// Stop processing, non-recoverable error
						return err
					}
				} else {
					// Dedupe, this stores key in store which means if sink fails and restarts it will not process the same message again
					isUnique, err := s.config.Dedupe.IsUnique(s.config.Context, kafkaCloudEvent)
					if err != nil {
						logger.Error("failed to check uniqueness of kafka message", "err", err, "kafkaCloudEvent", kafkaCloudEvent)
						// Stop processing, non-recoverable error
						return err
					}
					if !isUnique {
						logger.Debug("skipping non unique message", "kafkaCloudEvent", kafkaCloudEvent)
					} else {
						namespace, err := getNamespace(*e.TopicPartition.Topic)
						if err != nil {
							return NewProcessingError(fmt.Sprintf("failed to get namespace from topic: %s", *e.TopicPartition.Topic), DROP)
						}
						validateErr := s.config.SinkStore.validateEvent(s.config.Context, kafkaCloudEvent, namespace)
						if validateErr != nil {
							switch validateErr.ProcessingControl {
							case DEADLETTER:
								logger.Error("failed to parse kafka message to sink entry", "err", err, "kafkaCloudEvent", kafkaCloudEvent)
								err := s.deadLetter(kafkaMessage)
								if err != nil {
									logger.Error("faield to dead letter message", "err", err, "message", kafkaMessage)
									// Stop processing, non-recoverable error
									return err
								}
							}
						}

						s.state.buffer = append(s.state.buffer, kafkaCloudEvent)
						logger.Debug("event added to buffer", "event", kafkaCloudEvent)
					}
				}

				// TODO: currently we relay on `enable.auto.offset.store` to store offsets for commit
				// As we already manage commit manually it would be ideal to manage what goes into offset store to
				// This code should work in theory but at commit we get `Local: No offset stored` which means it is not stored
				// // Store message, this won't commit offset immediately just store it for the next manual commit
				// _, err = s.consumer.StoreMessage(e)
				// if err != nil {
				// 	slog.Error("cannot store kafka message for upcoming offset commit", "err", err, "event", ev)
				// 	// Stop processing, non-recoverable error
				// 	return err
				// }

				// Flush buffer and commit messages
				if s.state.messageCount >= s.config.MinCommitCount {
					err = s.flush()
					if err != nil {
						slog.Error("faield to flush", "err", err)
						// Stop processing, non-recoverable error
						return err
					}
				}
			case kafka.AssignedPartitions:
				slog.Info("kafka assigned partitions", "event", e)
				err := s.consumer.Assign(e.Partitions)
				if err != nil {
					return err
				}
			case kafka.RevokedPartitions:
				slog.Info("kafka revoked partitions", "event", e)
				err := s.consumer.Unassign()
				if err != nil {
					return err
				}
			case kafka.Error:
				// Errors should generally be considered
				// informational, the client will try to
				// automatically recover.
				slog.Error("kafka error", "code", e.Code(), "event", e)

				// But in this example we choose to terminate
				// the application if all brokers are down.
				if e.Code() == kafka.ErrAllBrokersDown {
					// TODO: should we panic? or what will restart polling?
					s.state.running = false
				}
			case kafka.OffsetsCommitted:
				// do nothing, this is an ack of the periodic offset commit
				slog.Debug("kafka offset committed", "offset", e.Offsets)
			default:
				slog.Debug("kafka ignored event", "event", e)
			}
		}
	}

	return s.Close()
}

func (s *Sink) Close() error {
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

// dedupeEventList removes duplicates from a list of events
func dedupeEventList(events []serializer.CloudEventsKafkaPayload) []serializer.CloudEventsKafkaPayload {
	keys := make(map[string]bool)
	list := []serializer.CloudEventsKafkaPayload{}

	for _, event := range events {
		key := event.GetKey()
		if _, value := keys[key]; !value {
			keys[key] = true
			list = append(list, event)
		}
	}
	return list
}
