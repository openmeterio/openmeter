package sink_test

import (
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/sink"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
)

func TestBuffer(t *testing.T) {
	buffer := sink.NewSinkBuffer()
	topic := "my-topic"

	sinkMessage1 := sinkmodels.SinkMessage{
		KafkaMessage: &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: 1,
				Offset:    1,
			},
		},
	}
	sinkMessage2 := sinkmodels.SinkMessage{
		KafkaMessage: &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: 1,
				Offset:    2,
			},
		},
	}

	// We call add with the same message twice but as it has the
	// same topic, partition and offset it should only be present in the buffer once.
	buffer.Add(sinkMessage1)
	buffer.Add(sinkMessage1)
	buffer.Add(sinkMessage2)

	assert.Equal(t, 2, buffer.Size())
	assert.ElementsMatch(t, []sinkmodels.SinkMessage{sinkMessage1, sinkMessage2}, buffer.Dequeue())
}

func TestBufferRemoveByPartitions(t *testing.T) {
	buffer := sink.NewSinkBuffer()
	topic := "my-topic"

	partition1 := kafka.TopicPartition{
		Topic:     &topic,
		Partition: 1,
		Offset:    1,
	}
	partition2 := kafka.TopicPartition{
		Topic:     &topic,
		Partition: 2,
		Offset:    1,
	}

	sinkMessage1 := sinkmodels.SinkMessage{
		KafkaMessage: &kafka.Message{
			TopicPartition: partition1,
		},
	}
	sinkMessage2 := sinkmodels.SinkMessage{
		KafkaMessage: &kafka.Message{
			TopicPartition: partition2,
		},
	}

	buffer.Add(sinkMessage1)
	buffer.Add(sinkMessage2)
	assert.Equal(t, 2, buffer.Size())

	buffer.RemoveByPartitions([]kafka.TopicPartition{partition2})
	assert.Equal(t, 1, buffer.Size())
}
