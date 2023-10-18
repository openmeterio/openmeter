package sink_test

import (
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/sink"
)

func TestBuffer(t *testing.T) {
	buffer := sink.NewSinkBuffer()
	topic := "my-topic"

	sinkMessage1 := sink.SinkMessage{
		KafkaMessage: &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: 1,
				Offset:    1,
			},
		},
	}
	sinkMessage2 := sink.SinkMessage{
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
	assert.EqualValues(t, []sink.SinkMessage{sinkMessage1, sinkMessage2}, buffer.Dequeue())
}
