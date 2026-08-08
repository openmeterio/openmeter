package common

import (
	"context"
	"fmt"
	"log/slog"
	"syscall"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/wire"
	"github.com/oklog/run"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/subject"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

var BalanceWorker = wire.NewSet(
	BalanceWorkerProvisionTopics,
	BalanceWorkerSubscriber,
	NewEntitlementRegistry,
	NewBalanceWorkerOptions,
	NewBalanceWorker,
	BalanceWorkerGroup,
)

var BalanceWorkerAdapter = wire.NewSet(
	NewBalanceWorkerEntitlementRepo,

	wire.Bind(new(balanceworker.BalanceWorkerRepository), new(BalanceWorkerEntitlementRepo)),
)

type BalanceWorkerEntitlementRepo interface {
	entitlement.EntitlementRepo
	balanceworker.BalanceWorkerRepository
}

func NewBalanceWorkerEntitlementRepo(db *db.Client) BalanceWorkerEntitlementRepo {
	return entitlementadapter.NewPostgresEntitlementRepo(db)
}

func BalanceWorkerProvisionTopics(conf config.BalanceWorkerConfiguration, eventsConfig config.EventsConfiguration) []pkgkafka.TopicConfig {
	var provisionTopics []pkgkafka.TopicConfig

	if conf.DLQ.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, pkgkafka.TopicConfig{
			Name:          conf.DLQ.Topic,
			Partitions:    conf.DLQ.AutoProvision.Partitions,
			RetentionTime: pkgkafka.TimeDurationMilliSeconds(conf.DLQ.AutoProvision.Retention),
		})
	}

	if eventsConfig.BalanceWorkerEvents.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, pkgkafka.TopicConfig{
			Name:       eventsConfig.BalanceWorkerEvents.Topic,
			Partitions: eventsConfig.BalanceWorkerEvents.AutoProvision.Partitions,
		})
	}

	return provisionTopics
}

// no closer function: the subscriber is closed by the router/worker
func BalanceWorkerSubscriber(conf config.BalanceWorkerConfiguration, brokerOptions watermillkafka.BrokerOptions) (message.Subscriber, error) {
	subscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{
		Broker:            brokerOptions,
		ConsumerGroupName: conf.ConsumerGroupName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka subscriber: %w", err)
	}

	return subscriber, nil
}

func NewBalanceWorkerOptions(
	eventConfig config.EventsConfiguration,
	routerOptions router.Options,
	eventBus eventbus.Publisher,
	entitlements *registry.Entitlement,
	repo balanceworker.BalanceWorkerRepository,
	notificationService notification.Service,
	subjectService subject.Service,
	customerService customer.Service,
	logger *slog.Logger,
	balanceWorkerConfiguration config.BalanceWorkerConfiguration,
) balanceworker.WorkerOptions {
	return balanceworker.WorkerOptions{
		SystemEventsTopic:        eventConfig.SystemEvents.Topic,
		IngestEventsTopic:        eventConfig.IngestEvents.Topic,
		BalanceWorkerEventsTopic: eventConfig.BalanceWorkerEvents.Topic,
		Router:                   routerOptions,
		EventBus:                 eventBus,
		Entitlement:              entitlements,
		Repo:                     repo,
		Subject:                  subjectService,
		Customer:                 customerService,
		Logger:                   logger,
		MetricMeter:              routerOptions.MetricMeter,
		NotificationService:      notificationService,
		HighWatermarkCacheSize:   balanceWorkerConfiguration.StateStorage.HighWatermarkCache.LRUCacheSize,
	}
}

func NewBalanceWorker(workerOptions balanceworker.WorkerOptions) (*balanceworker.Worker, error) {
	worker, err := balanceworker.New(workerOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize worker: %w", err)
	}

	return worker, nil
}

func BalanceWorkerGroup(
	ctx context.Context,
	consumer balanceworker.Consumer,
	telemetryServer TelemetryServer,
) run.Group {
	var group run.Group

	group.Add(
		func() error { return telemetryServer.ListenAndServe() },
		func(err error) { _ = telemetryServer.Shutdown(ctx) },
	)

	group.Add(
		func() error { return consumer.Run(ctx) },
		func(err error) { _ = consumer.Close() },
	)

	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	return group
}

func BalanceWorkerConsumer(
	cfg config.BalanceWorkerConfiguration,
	worker *balanceworker.Worker,
	eventBusConfig config.EventsConfiguration,
	kafkaConsumerConfig pkgkafka.ConsumerConfig,
	meta Metadata,
	metricMeter metric.Meter,
	tracer trace.Tracer,
	logger *slog.Logger,
) (balanceworker.Consumer, func(), error) {
	if cfg.UseWatermill {
		consumer, err := balanceworker.NewWatermillConsumer(balanceworker.WatermillConsumerOptions{
			SystemEventsTopic:        eventBusConfig.SystemEvents.Topic,
			IngestEventsTopic:        eventBusConfig.IngestEvents.Topic,
			BalanceWorkerEventsTopic: eventBusConfig.BalanceWorkerEvents.Topic,

			Worker: worker,
			// Router
		})
		return consumer, func() {}, err
	}

	rdkafkaEnvConfig, cleanup, err := NewEventBusConsumerEnvironmentConfig(
		cfg.ConsumerConfiguration,
		kafkaConsumerConfig,
		meta,
		metricMeter,
		tracer,
		logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create event bus consumer environment config: %w", err)
	}

	// Let's construct a librdkafka consumer
	consumer, err := balanceworker.NewRDKafkaConsumer(balanceworker.RDKafkaConsumerOptions{
		SystemEventsTopic:        eventBusConfig.SystemEvents.Topic,
		IngestEventsTopic:        eventBusConfig.IngestEvents.Topic,
		BalanceWorkerEventsTopic: eventBusConfig.BalanceWorkerEvents.Topic,

		Worker:              worker,
		ConsumerEnvironment: rdkafkaEnvConfig,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create rdkafka consumer: %w", err)
	}

	return consumer, cleanup, nil
}
