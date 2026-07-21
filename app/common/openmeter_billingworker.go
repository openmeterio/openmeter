package common

import (
	"context"
	"fmt"
	"log/slog"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/wire"
	"github.com/oklog/run"
	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingworker "github.com/openmeterio/openmeter/openmeter/billing/worker"
	billingworkercollect "github.com/openmeterio/openmeter/openmeter/billing/worker/collect"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/invoicemetrics"
	billingworkermetricsadapter "github.com/openmeterio/openmeter/openmeter/billing/worker/invoicemetrics/adapter"
	billingworkermetricsservice "github.com/openmeterio/openmeter/openmeter/billing/worker/invoicemetrics/service"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

var BillingWorker = wire.NewSet(
	App,
	Customer,
	Secret,
	Currency,

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
	NewBillingCollector,
	NewBillingWorkerInvoiceMetricsAdapter,
	NewBillingWorkerInvoiceMetricsService,
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
	billingCollector *billingworkercollect.InvoiceCollector,
	billingFsConfig config.BillingFeatureSwitchesConfiguration,
	logger *slog.Logger,
) billingworker.WorkerOptions {
	return billingworker.WorkerOptions{
		SystemEventsTopic: eventConfig.SystemEvents.Topic,

		Router:                  routerOptions,
		EventBus:                eventBus,
		BillingService:          billingRegistry.Billing,
		BillingCollector:        billingCollector,
		ChargesService:          billingRegistry.ChargesServiceOrNil(),
		SubscriptionService:     subscriptionServices.Service,
		BillingSubscriptionSync: subscriptionSyncService,
		Logger:                  logger,

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

func NewBillingWorkerInvoiceMetricsAdapter(db *entdb.Client, billingAdapter billing.Adapter) (invoicemetrics.Adapter, error) {
	return billingworkermetricsadapter.New(billingworkermetricsadapter.Config{
		Client:         db,
		BillingAdapter: billingAdapter,
	})
}

func NewBillingWorkerInvoiceMetricsService(
	adapter invoicemetrics.Adapter,
	meter otelmetric.Meter,
	logger *slog.Logger,
	billingFsConfig config.BillingFeatureSwitchesConfiguration,
) (invoicemetrics.Service, error) {
	return billingworkermetricsservice.New(billingworkermetricsservice.Config{
		Adapter:            adapter,
		Meter:              meter,
		Logger:             logger,
		ReportInterval:     time.Minute,
		OverdueThreshold:   10 * time.Minute,
		QueryTimeout:       30 * time.Second,
		ExcludedNamespaces: billingFsConfig.NamespaceLockdown,
	})
}

func BillingWorkerGroup(
	ctx context.Context,
	worker *billingworker.Worker,
	invoiceMetrics invoicemetrics.Service,
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

	group.Add(
		func() error { return invoiceMetrics.Start(ctx) },
		func(err error) { invoiceMetrics.Stop() },
	)

	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	return group
}
