package kafka

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

type SubscriberOptions struct {
	Broker        BrokerConfiguration
	ConsumerGroup string
	Topics        []string
}

func (o *SubscriberOptions) Validate() error {
	if err := o.Broker.Validate(); err != nil {
		return fmt.Errorf("invalid kafka config: %w", err)
	}

	if o.ConsumerGroup == "" {
		return fmt.Errorf("consumer group is required")
	}
	return nil
}

func NewEventProcessorConfig(in SubscriberOptions) (*cqrs.EventGroupProcessorConfig, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	return &cqrs.EventGroupProcessorConfig{
		SubscriberConstructor: func(params cqrs.EventGroupProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			wmConfig := kafka.SubscriberConfig{
				Brokers:       []string{in.Broker.KafkaConfig.Broker},
				ConsumerGroup: in.ConsumerGroup,
				Tracer:        kafka.NewOTELSaramaTracer(),
			}

			saramaConfig, err := newSaramaConfig(in.Broker)
			if err != nil {
				return nil, err
			}

			// TODO: add groupname to meter name

			saramaConfig.ClientID = fmt.Sprintf("%s.%s", saramaConfig.ClientID, params.EventGroupName)
			// TODO: consumer group name?!

			wmConfig.OverwriteSaramaConfig = saramaConfig

			if err := wmConfig.Validate(); err != nil {
				return nil, err
			}

			return kafka.NewSubscriber(wmConfig, watermill.NewSlogLogger(in.Broker.Logger))
		},
		AckOnUnknownEvent: true,
		Marshaler:         marshaler.New(),
	}, nil
}
