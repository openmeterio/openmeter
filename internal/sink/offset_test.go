package sink_test

import (
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/sink"
)

func TestOffsetStore(t *testing.T) {
	store := sink.NewOffsetStore()
	topic := "my-topic"
	metadata := ""

	topicPartition1 := kafka.TopicPartition{
		Topic:     &topic,
		Partition: 1,
		Offset:    2,
		Metadata:  &metadata,
	}
	topicPartition2 := kafka.TopicPartition{
		Topic:     &topic,
		Partition: 1,
		Offset:    1,
		Metadata:  &metadata,
	}
	topicPartition3 := kafka.TopicPartition{
		Topic:     &topic,
		Partition: 2,
		Offset:    100,
		Metadata:  &metadata,
	}

	assignedPartitions := []kafka.TopicPartition{topicPartition1, topicPartition2, topicPartition3}

	store.Add(topicPartition1)
	store.Add(topicPartition2)
	store.Add(topicPartition2)
	store.Add(topicPartition3)

	assert.ElementsMatch(t, []kafka.TopicPartition{
		{
			Topic:     &topic,
			Partition: 1,
			Offset:    3, // next offset on partition 0
			Metadata:  &metadata,
		},
		{
			Topic:     &topic,
			Partition: 2,
			Offset:    101, // next offset on partition 1
			Metadata:  &metadata,
		},
	}, store.Get(assignedPartitions))
}

func TestOffsetStoreSkipNonAssignedPartitions(t *testing.T) {
	store := sink.NewOffsetStore()
	topic := "my-topic"
	metadata := ""

	topicPartition1 := kafka.TopicPartition{
		Topic:     &topic,
		Partition: 1,
		Offset:    1,
		Metadata:  &metadata,
	}
	topicPartition2 := kafka.TopicPartition{
		Topic:     &topic,
		Partition: 2,
		Offset:    100,
		Metadata:  &metadata,
	}

	assignedPartitions := []kafka.TopicPartition{topicPartition1}

	store.Add(topicPartition1)
	store.Add(topicPartition2)

	assert.ElementsMatch(t, []kafka.TopicPartition{
		{
			Topic:     &topic,
			Partition: 1,
			Offset:    2, // next offset on partition 0
			Metadata:  &metadata,
		},
	}, store.Get(assignedPartitions))
}
