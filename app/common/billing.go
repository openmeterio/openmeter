package common

import (
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	billingsubscription "github.com/openmeterio/openmeter/openmeter/billing/subscription"
	billingworkerautoadvance "github.com/openmeterio/openmeter/openmeter/billing/worker/advance"
	billingworkercollect "github.com/openmeterio/openmeter/openmeter/billing/worker/collect"
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
	meterRepo meter.Repository,
	streamingConnector streaming.Connector,
	eventPublisher eventbus.Publisher,
) (billing.Service, error) {
	if !billingConfig.Enabled {
		return nil, nil
	}

	return billingservice.New(billingservice.Config{
		Adapter:            billingAdapter,
		AppService:         appService,
		CustomerService:    customerService,
		FeatureService:     featureConnector,
		Logger:             logger,
		MeterRepo:          meterRepo,
		StreamingConnector: streamingConnector,
		Publisher:          eventPublisher,
	})
}

func BillingSubscriptionValidator(
	billingService billing.Service,
	billingConfig config.BillingConfiguration,
) (*billingsubscription.Validator, error) {
	if !billingConfig.Enabled {
		return nil, nil
	}

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
