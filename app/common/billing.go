package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

var Billing = wire.NewSet(
	BillingService,
)

func BillingService(
	logger *slog.Logger,
	db *entdb.Client,
	appService app.Service,
	appStripeService appstripe.Service,
	billingConfig config.BillingConfiguration,
	customerService customer.Service,
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
		AppService:         appService,
		CustomerService:    customerService,
		FeatureService:     featureConnector,
		Logger:             logger,
		MeterRepo:          meterRepo,
		StreamingConnector: streamingConnector,
	})
}
