package kafka

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	watermillkafka "github.com/openmeterio/openmeter/internal/watermill/driver/kafka"
)

const (
	PartitionKeyMetadataKey = watermillkafka.PartitionKeyMetadataKey
)

type (
	Publisher = watermillkafka.Publisher
)

func NewPublisher(producer *kafka.Producer) *Publisher {
	return watermillkafka.NewPublisher(producer)
}

func AddPartitionKeyFromSubject(watermillIn *message.Message, cloudEvent event.Event) (*message.Message, error) {
	return watermillkafka.AddPartitionKeyFromSubject(watermillIn, cloudEvent)
}
