package common

import (
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/consumer"
	notificationconsumer "github.com/openmeterio/openmeter/openmeter/notification/consumer"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
	"go.opentelemetry.io/otel/metric"
)

func NotificationServiceProvisionTopics(conf config.NotificationConfiguration) []pkgkafka.TopicConfig {
	var provisionTopics []pkgkafka.TopicConfig

	if conf.Consumer.DLQ.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, pkgkafka.TopicConfig{
			Name:          conf.Consumer.DLQ.Topic,
			Partitions:    conf.Consumer.DLQ.AutoProvision.Partitions,
			RetentionTime: pkgkafka.TimeDurationMilliSeconds(conf.Consumer.DLQ.AutoProvision.Retention),
		})
	}

	return provisionTopics
}

// no closer function: the subscriber is closed by the router/worker
func NotificationConsumerSubscriber(conf config.NotificationConfiguration, brokerOptions watermillkafka.BrokerOptions) (message.Subscriber, error) {
	subscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{
		Broker:            brokerOptions,
		ConsumerGroupName: conf.Consumer.ConsumerGroupName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka subscriber: %w", err)
	}

	return subscriber, nil
}

func NotificationConsumerOptions(
	eventConfig config.EventsConfiguration,
	conf config.NotificationConfiguration,
	meter metric.Meter,
	logger *slog.Logger,
	wmSubscriber message.Subscriber,
	messagePublisher message.Publisher,
	eventPublisher eventbus.Publisher,
	notificationService notification.Service,
) consumer.Options {
	return consumer.Options{
		SystemEventsTopic: eventConfig.SystemEvents.Topic,
		Router: router.Options{
			Subscriber:  wmSubscriber,
			Publisher:   messagePublisher,
			Logger:      logger,
			MetricMeter: meter,

			Config: conf.Consumer,
		},
		Marshaler: eventPublisher.Marshaler(),

		Notification: notificationService,

		Logger: logger,
	}
}

func NewNotificationConsumer(
	opts notificationconsumer.Options,
) (*notificationconsumer.Consumer, error) {
	return notificationconsumer.New(opts)
}

func NewNotificationBalanceThresholdEventHandler(
	notificationService notification.Service,
	logger *slog.Logger,
) (*notificationconsumer.BalanceThresholdEventHandler, error) {
	return notificationconsumer.NewBalanceThresholdEventHandler(notificationconsumer.BalanceThresholdEventHandlerOptions{
		Notification: notificationService,
		Logger:       logger,
	})
}
