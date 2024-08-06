package kafka

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
)

type PublisherOptions struct {
	Broker          BrokerConfiguration
	ProvisionTopics []AutoProvisionTopic
}

func (o *PublisherOptions) Validate() error {
	if err := o.Broker.Validate(); err != nil {
		return fmt.Errorf("invalid kafka config: %w", err)
	}

	return nil
}

func NewPublisher(ctx context.Context, in PublisherOptions) (*kafka.Publisher, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	wmConfig := kafka.PublisherConfig{
		Brokers:   []string{in.Broker.KafkaConfig.Broker},
		Marshaler: marshalerWithPartitionKey{},
		Tracer:    kafka.NewOTELSaramaTracer(),
	}

	saramaConfig, err := newSaramaConfig(in.Broker)
	if err != nil {
		return nil, err
	}

	wmConfig.OverwriteSaramaConfig = saramaConfig

	if err := wmConfig.Validate(); err != nil {
		return nil, err
	}

	if err := provisionTopics(ctx, in.Broker.Logger, in.Broker.KafkaConfig.CreateKafkaConfig(), in.ProvisionTopics); err != nil {
		return nil, err
	}

	return kafka.NewPublisher(wmConfig, watermill.NewSlogLogger(in.Broker.Logger))
}
