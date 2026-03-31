package common

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"syscall"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/wire"
	"github.com/oklog/run"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/app/stripe/invoicesync"
	invoicesyncadapter "github.com/openmeterio/openmeter/openmeter/app/stripe/invoicesync/adapter"
	billingworker "github.com/openmeterio/openmeter/openmeter/billing/worker"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/secret"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
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
	Billing,
	LedgerStack,

	NewBillingWorkerOptions,
	NewBillingWorker,
	NewBillingSubscriptionSyncAdapter,
	NewBillingSubscriptionSyncService,
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
	billingRegistry BillingRegistry,
	subscriptionServices SubscriptionServiceWithWorkflow,
	subscriptionSyncService subscriptionsync.Service,
	billingFsConfig config.BillingFeatureSwitchesConfiguration,
	logger *slog.Logger,
) billingworker.WorkerOptions {
	return billingworker.WorkerOptions{
		SystemEventsTopic: eventConfig.SystemEvents.Topic,

		Router:                  routerOptions,
		EventBus:                eventBus,
		BillingService:          billingRegistry.Billing,
		ChargesService:          billingRegistry.ChargesServiceOrNil(),
		SubscriptionService:     subscriptionServices.Service,
		BillingSubscriptionSync: subscriptionSyncService,
		Logger:                  logger,

		// Feature switches
		LockdownNamespaces: billingFsConfig.NamespaceLockdown,
	}
}

func NewBillingWorker(
	workerOptions billingworker.WorkerOptions,
	appService app.Service,
	stripeAppService appstripe.Service,
	secretService secret.Service,
	billingRegistry BillingRegistry,
	publisher eventbus.Publisher,
	logger *slog.Logger,
	syncPlanAdapter *invoicesyncadapter.Adapter,
) (*billingworker.Worker, error) {
	worker, err := billingworker.New(workerOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize worker: %w", err)
	}

	// Create locker for advisory locks on sync plan execution
	syncPlanLocker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: logger.With("component", "stripe-sync-plan-locker"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sync plan locker: %w", err)
	}

	// Register sync plan handler for async Stripe invoice sync
	syncPlanHandler, err := invoicesync.NewHandler(invoicesync.HandlerConfig{
		Adapter:                syncPlanAdapter,
		AppService:             appService,
		BillingService:         billingRegistry.Billing,
		StripeAppService:       stripeAppService,
		SecretService:          secretService,
		StripeAppClientFactory: stripeclient.NewStripeAppClient,
		Publisher:              publisher,
		LockFunc: func(ctx context.Context, namespace, invoiceID string) error {
			key, err := lockr.NewKey("namespace", namespace, "invoice_sync", invoiceID)
			if err != nil {
				return fmt.Errorf("creating lock key: %w", err)
			}
			if err := syncPlanLocker.LockForTX(ctx, key); err != nil {
				if errors.Is(err, lockr.ErrLockTimeout) {
					return invoicesync.ErrSyncPlanLocked
				}
				return err
			}
			return nil
		},
		Logger: logger.With("component", "stripe-sync-plan"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sync plan handler: %w", err)
	}

	worker.AddHandler(grouphandler.NewGroupEventHandler(func(ctx context.Context, event *invoicesync.ExecuteSyncPlanEvent) error {
		return syncPlanHandler.Handle(ctx, event)
	}))

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
