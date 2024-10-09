package kafka

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"

	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

type PublisherOptions struct {
	Broker           BrokerOptions
	ProvisionTopics  []pkgkafka.TopicConfig
	TopicProvisioner pkgkafka.TopicProvisioner
}

func (o *PublisherOptions) Validate() error {
	if err := o.Broker.Validate(); err != nil {
		return fmt.Errorf("invalid broker options: %w", err)
	}

	return nil
}

func NewPublisher(ctx context.Context, in PublisherOptions) (*kafka.Publisher, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	saramaConfig, err := in.Broker.createKafkaConfig("publisher")
	if err != nil {
		return nil, err
	}

	wmConfig := kafka.PublisherConfig{
		Brokers:   []string{in.Broker.KafkaConfig.Broker},
		Marshaler: marshalerWithPartitionKey{},
		Tracer:    kafka.NewOTELSaramaTracer(), // This relies on the global trace provider
	}

	wmConfig.OverwriteSaramaConfig = saramaConfig

	if err := wmConfig.Validate(); err != nil {
		return nil, err
	}

	if err = in.TopicProvisioner.Provision(ctx, in.ProvisionTopics...); err != nil {
		return nil, err
	}

	return kafka.NewPublisher(wmConfig, watermill.NewSlogLogger(in.Broker.Logger))
}
