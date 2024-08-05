package kafka

import (
	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/cloudevents/sdk-go/v2/event"
)

const (
	PartitionKeyMetadataKey = "x-kafka-partition-key"
)

type marshalerWithPartitionKey struct {
	kafka.DefaultMarshaler
}

func (m marshalerWithPartitionKey) Marshal(topic string, msg *message.Message) (*sarama.ProducerMessage, error) {
	kafkaMsg, err := m.DefaultMarshaler.Marshal(topic, msg)
	if err != nil {
		return nil, err
	}

	partitionKey := msg.Metadata.Get(PartitionKeyMetadataKey)
	if partitionKey != "" {
		kafkaMsg.Key = sarama.ByteEncoder(partitionKey)
	}

	return kafkaMsg, nil
}

// AddPartitionKeyFromSubject adds partition key to the message based on the CloudEvent subject.
func AddPartitionKeyFromSubject(watermillIn *message.Message, cloudEvent event.Event) (*message.Message, error) {
	if cloudEvent.Subject() != "" {
		watermillIn.Metadata[PartitionKeyMetadataKey] = cloudEvent.Subject()
	}
	return watermillIn, nil
}
