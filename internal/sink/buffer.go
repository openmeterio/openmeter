package sink

import (
	"fmt"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type SinkBuffer struct {
	mu   sync.Mutex
	data map[string]SinkMessage
}

func NewSinkBuffer() *SinkBuffer {
	return &SinkBuffer{
		data: map[string]SinkMessage{},
	}
}

func (b *SinkBuffer) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.data)
}

func (b *SinkBuffer) Add(message SinkMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Unique identifier for each message (topic + partition + offset)
	key := message.KafkaMessage.String()
	b.data[key] = message
}

func (b *SinkBuffer) Dequeue() []SinkMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	list := []SinkMessage{}
	for key, message := range b.data {
		list = append(list, message)
		delete(b.data, key)
	}
	return list
}

// RemoveByRevokedPartitions removes messages from the buffer that are assigned to the revoked partitions.
func (b *SinkBuffer) RemoveByRevokedPartitions(revokedPartitions []kafka.TopicPartition) {
	b.mu.Lock()
	defer b.mu.Unlock()

	partitionMap := map[string]bool{}
	for _, topicPartition := range revokedPartitions {
		key := topicPartitionKey(topicPartition)
		partitionMap[key] = true
	}

	for key, message := range b.data {
		topicKey := topicPartitionKey(message.KafkaMessage.TopicPartition)

		if partitionMap[topicKey] {
			delete(b.data, key)
		}
	}
}

func topicPartitionKey(partition kafka.TopicPartition) string {
	var topic string
	if partition.Topic != nil {
		topic = *partition.Topic
	}
	return partitionKey(topic, partition.Partition)
}

func partitionKey(topic string, partition int32) string {
	return fmt.Sprintf("%s-%d", topic, partition)
}
