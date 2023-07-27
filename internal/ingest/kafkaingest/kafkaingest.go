package kafkaingest

import (
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Collector is a receiver of events that handles sending those events to a downstream Kafka broker.
type Collector struct {
	Producer *kafka.Producer
	// NamespacedTopicTemplate needs to contain at least one string parameter passed to fmt.Sprintf.
	// For example: "om_%s_events"
	NamespacedTopicTemplate string
	Schema                  Schema
}

// Schema serializes events.
type Schema interface {
	SerializeKey(topic string, ev event.Event) ([]byte, error)
	SerializeValue(topic string, ev event.Event) ([]byte, error)
}

// Receive receives an event produces a message in a Kafka topic.
func (s Collector) Receive(ev event.Event, namespace string) error {
	topic := fmt.Sprintf(s.NamespacedTopicTemplate, namespace)
	key, err := s.Schema.SerializeKey(topic, ev)
	if err != nil {
		return fmt.Errorf("serialize event key: %w", err)
	}

	value, err := s.Schema.SerializeValue(topic, ev)
	if err != nil {
		return fmt.Errorf("serialize event value: %w", err)
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Timestamp:      ev.Time(),
		Headers: []kafka.Header{
			{Key: "specversion", Value: []byte(ev.SpecVersion())},
		},
		Key:   key,
		Value: value,
	}

	err = s.Producer.Produce(msg, nil)
	if err != nil {
		return fmt.Errorf("producing kafka message: %w", err)
	}

	return nil
}
