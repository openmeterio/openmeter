package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/config"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/noop"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

// TODO: make this global? or more generic?
type WatermillClientID string

func NewPublisher(
	ctx context.Context,
	conf config.Configuration,
	clientID WatermillClientID,
	logger *slog.Logger,
	metricMeter metric.Meter,
) (message.Publisher, func(), error) {
	if !conf.Events.Enabled {
		return &noop.Publisher{}, func() {}, nil
	}

	provisionTopics := []watermillkafka.AutoProvisionTopic{}
	if conf.Events.SystemEvents.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, watermillkafka.AutoProvisionTopic{
			Topic:         conf.Events.SystemEvents.Topic,
			NumPartitions: int32(conf.Events.SystemEvents.AutoProvision.Partitions),
		})
	}

	publisher, err := watermillkafka.NewPublisher(ctx, watermillkafka.PublisherOptions{
		Broker: watermillkafka.BrokerOptions{
			KafkaConfig:  conf.Ingest.Kafka.KafkaConfiguration,
			ClientID:     string(clientID),
			Logger:       logger,
			MetricMeter:  metricMeter,
			DebugLogging: conf.Telemetry.Log.Level == slog.LevelDebug,
		},
		ProvisionTopics: provisionTopics,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize event publisher: %w", err)
	}

	return publisher, func() {
		// TODO: isn't this logged by the publisher itself?
		logger.Info("closing event publisher")

		if err = publisher.Close(); err != nil {
			logger.Error("failed to close event publisher", "error", err)
		}
	}, nil
}

func NewEventBusPublisher(
	publisher message.Publisher,
	conf config.Configuration, // TODO: limit configuration
	logger *slog.Logger,
) (eventbus.Publisher, error) {
	eventBusPublisher, err := eventbus.New(eventbus.Options{
		Publisher:              publisher,
		Config:                 conf.Events,
		Logger:                 logger,
		MarshalerTransformFunc: watermillkafka.AddPartitionKeyFromSubject,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize event bus publisher: %w", err)
	}

	return eventBusPublisher, nil
}
