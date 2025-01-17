package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/ingestadapter"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

func NewKafkaIngestCollector(
	config config.KafkaIngestConfiguration,
	producer *kafka.Producer,
	topicResolver topicresolver.Resolver,
	topicProvisioner pkgkafka.TopicProvisioner,
) (*kafkaingest.Collector, error) {
	collector, err := kafkaingest.NewCollector(
		producer,
		serializer.NewJSONSerializer(),
		topicResolver,
		topicProvisioner,
		config.Partitions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kafka ingest: %w", err)
	}

	return collector, nil
}

func NewIngestCollector(
	conf config.Configuration,
	kafkaCollector *kafkaingest.Collector,
	logger *slog.Logger,
	meter metric.Meter,
) (ingest.Collector, func(), error) {
	collector, err := ingestadapter.WithMetrics(kafkaCollector, meter)
	if err != nil {
		return nil, nil, fmt.Errorf("init kafka ingest: %w", err)
	}

	if conf.Dedupe.Enabled {
		deduplicator, err := conf.Dedupe.NewDeduplicator()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize deduplicator: %w", err)
		}

		return ingest.DeduplicatingCollector{
				Collector:    collector,
				Deduplicator: deduplicator,
			}, func() {
				collector.Close()

				logger.Info("closing deduplicator")

				err := deduplicator.Close()
				if err != nil {
					logger.Error("failed to close deduplicator", "error", err)
				}
			}, nil
	}

	// Note: closing function is called by dedupe as well
	return collector, func() { collector.Close() }, nil
}

// TODO: create a separate file or package for each application instead

func NewServerPublisher(
	ctx context.Context,
	options watermillkafka.PublisherOptions,
	logger *slog.Logger,
) (message.Publisher, func(), error) {
	return NewPublisher(ctx, options, logger)
}

func ServerProvisionTopics(conf config.EventsConfiguration) []pkgkafka.TopicConfig {
	var provisionTopics []pkgkafka.TopicConfig

	if conf.SystemEvents.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, pkgkafka.TopicConfig{
			Name:       conf.SystemEvents.Topic,
			Partitions: conf.SystemEvents.AutoProvision.Partitions,
		})
	}

	return provisionTopics
}
