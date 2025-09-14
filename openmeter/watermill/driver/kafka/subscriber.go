package kafka

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	// defaultMaxProcessinTime is the default maximum time a message is allowed to be processed before the
	// partition assignment is lost by the consumer. For now we just set it to a high enough value (default 1s)
	//
	// Later we can make this configurable if needed.
	defaultMaxProcessingTime = 5 * time.Minute
)

type SubscriberOptions struct {
	Broker            BrokerOptions
	ConsumerGroupName string
}

func (o *SubscriberOptions) Validate() error {
	if err := o.Broker.Validate(); err != nil {
		return err
	}

	if o.ConsumerGroupName == "" {
		return errors.New("consumer group name is required")
	}

	return nil
}

func NewSubscriber(in SubscriberOptions) (message.Subscriber, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	saramaConfig, err := in.Broker.createKafkaConfig("subscriber")
	if err != nil {
		return nil, err
	}

	saramaConfig.Consumer.MaxProcessingTime = defaultMaxProcessingTime

	wmConfig := kafka.SubscriberConfig{
		Brokers:               strings.Split(in.Broker.KafkaConfig.Broker, ","),
		OverwriteSaramaConfig: saramaConfig,
		ConsumerGroup:         in.ConsumerGroupName,
		ReconnectRetrySleep:   100 * time.Millisecond,
		Unmarshaler:           kafka.DefaultMarshaler{},
	}

	if err := wmConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid subscriber config: %w", err)
	}

	// Initialize Kafka subscriber
	return kafka.NewSubscriber(wmConfig, watermill.NewSlogLogger(in.Broker.Logger))
}
