package kafka

import (
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/cloudevents/sdk-go/v2/event"

	watermillkafka "github.com/openmeterio/openmeter/internal/watermill/driver/kafka"
)

const (
	PartitionKeyMetadataKey = watermillkafka.PartitionKeyMetadataKey
)

type (
	PublisherOptions   = watermillkafka.PublisherOptions
	AutoProvisionTopic = watermillkafka.AutoProvisionTopic
)

func NewPublisher(in PublisherOptions) (*kafka.Publisher, error) {
	return watermillkafka.NewPublisher(in)
}

func AddPartitionKeyFromSubject(watermillIn *message.Message, cloudEvent event.Event) (*message.Message, error) {
	return watermillkafka.AddPartitionKeyFromSubject(watermillIn, cloudEvent)
}
