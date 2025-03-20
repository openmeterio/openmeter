package common

import (
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	billingcustomer "github.com/openmeterio/openmeter/openmeter/billing/validators/customer"
	billingsubscription "github.com/openmeterio/openmeter/openmeter/billing/validators/subscription"
	billingworkerautoadvance "github.com/openmeterio/openmeter/openmeter/billing/worker/advance"
	billingworkercollect "github.com/openmeterio/openmeter/openmeter/billing/worker/collect"
	billingworkersubscription "github.com/openmeterio/openmeter/openmeter/billing/worker/subscription"
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
	db *entdb.Client,
	appService app.Service,
	billingAdapter billing.Adapter,
	billingConfig config.BillingConfiguration,
	customerService customer.Service,
	featureConnector feature.FeatureConnector,
	meterService meter.Service,
	streamingConnector streaming.Connector,
	eventPublisher eventbus.Publisher,
	advancementStrategy billing.AdvancementStrategy,
	subscriptionServices SubscriptionServiceWithWorkflow,
	subscriptionSync *billingworkersubscription.Handler,
) (billing.Service, error) {
	service, err := billingservice.New(billingservice.Config{
		Adapter:             billingAdapter,
		AppService:          appService,
		CustomerService:     customerService,
		FeatureService:      featureConnector,
		Logger:              logger,
		MeterService:        meterService,
		StreamingConnector:  streamingConnector,
		Publisher:           eventPublisher,
		AdvancementStrategy: advancementStrategy,
	})
	if err != nil {
		return nil, err
	}

	handler, err := NewBillingSubscriptionHandler(logger, subscriptionServices, service, billingAdapter)
	if err != nil {
		return nil, err
	}

	validator, err := billingcustomer.NewValidator(service, handler, subscriptionServices.Service)
	if err != nil {
		return nil, err
	}

	customerService.RegisterRequestValidator(validator)

	return service, nil
}

func BillingSubscriptionValidator(
	billingService billing.Service,
	billingConfig config.BillingConfiguration,
) (*billingsubscription.Validator, error) {
	return billingsubscription.NewValidator(billingService)
}

func NewBillingAutoAdvancer(logger *slog.Logger, service billing.Service) (*billingworkerautoadvance.AutoAdvancer, error) {
	return billingworkerautoadvance.NewAdvancer(billingworkerautoadvance.Config{
		BillingService: service,
		Logger:         logger,
	})
}

func NewBillingCollector(logger *slog.Logger, service billing.Service) (*billingworkercollect.InvoiceCollector, error) {
	return billingworkercollect.NewInvoiceCollector(billingworkercollect.Config{
		BillingService: service,
		Logger:         logger,
	})
}

func NewBillingSubscriptionReconciler(logger *slog.Logger, subsServices SubscriptionServiceWithWorkflow, subscriptionSync *billingworkersubscription.Handler) (*billingworkersubscription.Reconciler, error) {
	return billingworkersubscription.NewReconciler(billingworkersubscription.ReconcilerConfig{
		SubscriptionService: subsServices.Service,
		SubscriptionSync:    subscriptionSync,
		Logger:              logger,
	})
}

func NewBillingSubscriptionHandler(logger *slog.Logger, subsServices SubscriptionServiceWithWorkflow, billingService billing.Service, billingAdapter billing.Adapter) (*billingworkersubscription.Handler, error) {
	return billingworkersubscription.New(billingworkersubscription.Config{
		SubscriptionService: subsServices.Service,
		BillingService:      billingService,
		TxCreator:           billingAdapter,
		Logger:              logger,
	})
}
