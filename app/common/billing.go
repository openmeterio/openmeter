package common

import (
	"context"
	"fmt"
	"log/slog"
	"syscall"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/wire"
	"github.com/oklog/run"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	billingworker "github.com/openmeterio/openmeter/openmeter/billing/worker"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

func BillingWorkerProvisionTopics(conf config.BillingConfiguration) []pkgkafka.TopicConfig {
	var provisionTopics []pkgkafka.TopicConfig

	if conf.Worker.DLQ.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, pkgkafka.TopicConfig{
			Name:          conf.Worker.DLQ.Topic,
			Partitions:    conf.Worker.DLQ.AutoProvision.Partitions,
			RetentionTime: pkgkafka.TimeDurationMilliSeconds(conf.Worker.DLQ.AutoProvision.Retention),
		})
	}

	return provisionTopics
}

// no closer function: the subscriber is closed by the router/worker
func BillingWorkerSubscriber(conf config.BillingConfiguration, brokerOptions watermillkafka.BrokerOptions) (message.Subscriber, error) {
	subscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{
		Broker:            brokerOptions,
		ConsumerGroupName: conf.Worker.ConsumerGroupName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka subscriber: %w", err)
	}

	return subscriber, nil
}

func NewBillingWorkerOptions(
	eventConfig config.EventsConfiguration,
	routerOptions router.Options,
	eventBus eventbus.Publisher,
	billingService billing.Service,
	logger *slog.Logger,
) billingworker.WorkerOptions {
	return billingworker.WorkerOptions{
		SystemEventsTopic: eventConfig.SystemEvents.Topic,

		Router:         routerOptions,
		EventBus:       eventBus,
		BillingService: billingService,
		Logger:         logger,
	}
}

func NewBillingWorker(workerOptions billingworker.WorkerOptions) (*billingworker.Worker, error) {
	worker, err := billingworker.New(workerOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize worker: %w", err)
	}

	return worker, nil
}

func BillingWorkerGroup(
	ctx context.Context,
	worker *billingworker.Worker,
	telemetryServer TelemetryServer,
) run.Group {
	var group run.Group

	group.Add(
		func() error { return telemetryServer.ListenAndServe() },
		func(err error) { _ = telemetryServer.Shutdown(ctx) },
	)

	group.Add(
		func() error { return worker.Run(ctx) },
		func(err error) { _ = worker.Close() },
	)

	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	return group
}

func BillingService(
	logger *slog.Logger,
	db *entdb.Client,
	billingConfig config.BillingConfiguration,
	customerService customer.Service,
	appService app.Service,
	featureConnector feature.FeatureConnector,
	meterRepo meter.Repository,
	streamingConnector streaming.Connector,
) (billing.Service, error) {
	if !billingConfig.Enabled {
		return nil, nil
	}

	adapter, err := billingadapter.New(billingadapter.Config{
		Client: db,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("creating billing adapter: %w", err)
	}

	return billingservice.New(billingservice.Config{
		Adapter:            adapter,
		CustomerService:    customerService,
		AppService:         appService,
		Logger:             logger,
		FeatureService:     featureConnector,
		MeterRepo:          meterRepo,
		StreamingConnector: streamingConnector,
	})
}

var BillingWorker = wire.NewSet(
	wire.FieldsOf(new(config.BillingWorkerConfiguration), "ConsumerConfiguration"),
	wire.FieldsOf(new(config.BillingConfiguration), "Worker"),

	BillingWorkerProvisionTopics,
	BillingWorkerSubscriber,

	NewCustomerService,
	BillingService,

	NewBillingWorkerOptions,
	NewBillingWorker,
	BillingWorkerGroup,
)
