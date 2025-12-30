package consumer

import (
	"errors"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// PartitionKey is used as a key for partition maps.
// It contains only the topic (as a string) and partition ID.
type PartitionKey struct {
	Topic     string
	Partition int
}

// PartitionKeyFromTopicPartition creates a PartitionKey from a kafka.TopicPartition.
func PartitionKeyFromTopicPartition(tp kafka.TopicPartition) (PartitionKey, error) {
	if tp.Topic == nil {
		return PartitionKey{}, errors.New("topic is nil")
	}

	return PartitionKey{
		Topic:     *tp.Topic,
		Partition: int(tp.Partition),
	}, nil
}

// PartitionKeyFromMessage creates a PartitionKey from a kafka.Message.
func PartitionKeyFromMessage(msg *kafka.Message) (PartitionKey, error) {
	if msg.TopicPartition.Topic == nil {
		return PartitionKey{}, errors.New("message topic is nil")
	}
	return PartitionKey{
		Topic:     *msg.TopicPartition.Topic,
		Partition: int(msg.TopicPartition.Partition),
	}, nil
}

// String returns a string representation of the partition key.
func (k PartitionKey) String() string {
	return fmt.Sprintf("%s-%d", k.Topic, k.Partition)
}
