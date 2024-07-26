package kafka

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const (
	PartitionKeyMetadataKey = "x-kafka-partition-key"
)

type Publisher struct {
	producer *kafka.Producer
}

var _ message.Publisher = (*Publisher)(nil)

func NewPublisher(producer *kafka.Producer) *Publisher {
	return &Publisher{producer: producer}
}

func (p *Publisher) Publish(topic string, messages ...*message.Message) error {
	for _, message := range messages {
		kafkaMessage := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(message.Payload),
			Headers:        make([]kafka.Header, 0, len(message.Metadata)),
		}

		for k, v := range message.Metadata {
			if k == PartitionKeyMetadataKey {
				continue
			}
			kafkaMessage.Headers = append(kafkaMessage.Headers, kafka.Header{
				Key:   k,
				Value: []byte(v),
			})
		}

		if partitionKey, ok := message.Metadata[PartitionKeyMetadataKey]; ok {
			kafkaMessage.Key = []byte(partitionKey)
		}

		if err := p.producer.Produce(kafkaMessage, nil); err != nil {
			return err
		}
	}

	return nil
}

func (p *Publisher) Close() error {
	p.producer.Close()
	return nil
}

func AddPartitionKeyFromSubject(watermillIn *message.Message, cloudEvent event.Event) (*message.Message, error) {
	if cloudEvent.Subject() != "" {
		watermillIn.Metadata[PartitionKeyMetadataKey] = cloudEvent.Subject()
	}
	return watermillIn, nil
}
