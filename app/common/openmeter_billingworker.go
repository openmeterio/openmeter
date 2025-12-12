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
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingworker "github.com/openmeterio/openmeter/openmeter/billing/worker"
	billingworkersubscription "github.com/openmeterio/openmeter/openmeter/billing/worker/subscription"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

var BillingWorker = wire.NewSet(
	App,
	Customer,
	Secret,

	BillingWorkerProvisionTopics,
	BillingWorkerSubscriber,

	Lockr,
	FFX,

	Subscription,
	ProductCatalog,
	Entitlement,
	BillingAdapter,
	BillingService,

	NewBillingWorkerOptions,
	NewBillingWorker,
	NewBillingSubscriptionHandler,
	BillingWorkerGroup,
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
	billingAdapter billing.Adapter,
	subscriptionServices SubscriptionServiceWithWorkflow,
	subsSyncHandler *billingworkersubscription.Handler,
	billingFsConfig config.BillingFeatureSwitchesConfiguration,
	logger *slog.Logger,
) billingworker.WorkerOptions {
	return billingworker.WorkerOptions{
		SystemEventsTopic: eventConfig.SystemEvents.Topic,

		Router:                         routerOptions,
		EventBus:                       eventBus,
		BillingService:                 billingService,
		BillingAdapter:                 billingAdapter,
		SubscriptionService:            subscriptionServices.Service,
		BillingSubscriptionSyncHandler: subsSyncHandler,
		Logger:                         logger,

		// Feature switches
		LockdownNamespaces: billingFsConfig.NamespaceLockdown,
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
