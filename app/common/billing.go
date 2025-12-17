package common

import (
	"log/slog"

	"github.com/google/wire"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	billingcustomer "github.com/openmeterio/openmeter/openmeter/billing/validators/customer"
	billingsubscription "github.com/openmeterio/openmeter/openmeter/billing/validators/subscription"
	billingworkerautoadvance "github.com/openmeterio/openmeter/openmeter/billing/worker/advance"
	billingworkercollect "github.com/openmeterio/openmeter/openmeter/billing/worker/collect"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	subscriptionsyncadapter "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/reconciler"
	subscriptionsyncservice "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

var Billing = wire.NewSet(
	BillingService,
	BillingAdapter,
	wire.Bind(new(billing.CustomerOverrideService), new(billing.Service)),
)

func BillingAdapter(
	logger *slog.Logger,
	db *entdb.Client,
) (billing.Adapter, error) {
	return billingadapter.New(billingadapter.Config{
		Client: db,
		Logger: logger,
	})
}

func BillingService(
	logger *slog.Logger,
	appService app.Service,
	billingAdapter billing.Adapter,
	customerService customer.Service,
	featureConnector feature.FeatureConnector,
	meterService meter.Service,
	streamingConnector streaming.Connector,
	eventPublisher eventbus.Publisher,
	billingConfig config.BillingConfiguration,
	subscriptionServices SubscriptionServiceWithWorkflow,
	db *entdb.Client,
	fsConfig config.BillingFeatureSwitchesConfiguration,
	tracer trace.Tracer,
) (billing.Service, error) {
	service, err := billingservice.New(billingservice.Config{
		Adapter:                      billingAdapter,
		AppService:                   appService,
		CustomerService:              customerService,
		FeatureService:               featureConnector,
		Logger:                       logger,
		MeterService:                 meterService,
		StreamingConnector:           streamingConnector,
		Publisher:                    eventPublisher,
		AdvancementStrategy:          billingConfig.AdvancementStrategy,
		FSNamespaceLockdown:          fsConfig.NamespaceLockdown,
		MaxParallelQuantitySnapshots: billingConfig.MaxParallelQuantitySnapshots,
	})
	if err != nil {
		return nil, err
	}

	// To prevent circular dependencies, we create the validator here
	subscriptionSyncAdapter, err := NewBillingSubscriptionSyncAdapter(db)
	if err != nil {
		return nil, err
	}
	subscriptionSyncService, err := NewBillingSubscriptionSyncService(logger, subscriptionServices, service, subscriptionSyncAdapter, tracer)
	if err != nil {
		return nil, err
	}

	validator, err := billingcustomer.NewValidator(service, subscriptionSyncService, subscriptionServices.Service)
	if err != nil {
		return nil, err
	}

	customerService.RegisterRequestValidator(validator)

	subscriptionValidator, err := billingsubscription.NewValidator(service)
	if err != nil {
		return nil, err
	}

	if err = subscriptionServices.Service.RegisterHook(subscriptionValidator); err != nil {
		return nil, err
	}

	return service, nil
}

func NewBillingAutoAdvancer(logger *slog.Logger, service billing.Service) (*billingworkerautoadvance.AutoAdvancer, error) {
	return billingworkerautoadvance.NewAdvancer(billingworkerautoadvance.Config{
		BillingService: service,
		Logger:         logger,
	})
}

func NewBillingCollector(logger *slog.Logger, service billing.Service, fs config.BillingFeatureSwitchesConfiguration) (*billingworkercollect.InvoiceCollector, error) {
	return billingworkercollect.NewInvoiceCollector(billingworkercollect.Config{
		BillingService:   service,
		Logger:           logger,
		LockedNamespaces: fs.NamespaceLockdown,
	})
}

func NewBillingSubscriptionReconciler(logger *slog.Logger, subsServices SubscriptionServiceWithWorkflow, subscriptionSync subscriptionsync.Service, customerService customer.Service) (*reconciler.Reconciler, error) {
	return reconciler.NewReconciler(reconciler.ReconcilerConfig{
		SubscriptionService: subsServices.Service,
		SubscriptionSync:    subscriptionSync,
		Logger:              logger,
		CustomerService:     customerService,
	})
}

func NewBillingSubscriptionSyncAdapter(db *entdb.Client) (subscriptionsync.Adapter, error) {
	return subscriptionsyncadapter.New(subscriptionsyncadapter.Config{
		Client: db,
	})
}

func NewBillingSubscriptionSyncService(logger *slog.Logger, subsServices SubscriptionServiceWithWorkflow, billingService billing.Service, subscriptionSyncAdapter subscriptionsync.Adapter, tracer trace.Tracer) (subscriptionsync.Service, error) {
	return subscriptionsyncservice.New(subscriptionsyncservice.Config{
		SubscriptionService:     subsServices.Service,
		BillingService:          billingService,
		SubscriptionSyncAdapter: subscriptionSyncAdapter,
		Logger:                  logger,
		Tracer:                  tracer,
	})
}
