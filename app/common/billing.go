package common

import (
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	billingsubscription "github.com/openmeterio/openmeter/openmeter/billing/subscription"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
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
	appStripeService appstripe.Service,
	billingAdapter billing.Adapter,
	billingConfig config.BillingConfiguration,
	customerService customer.Service,
	featureConnector feature.FeatureConnector,
	meterRepo meter.Repository,
	streamingConnector streaming.Connector,
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
