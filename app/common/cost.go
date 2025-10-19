package common

import (
	"log/slog"
	"time"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/cost/modelcost"
	costservice "github.com/openmeterio/openmeter/openmeter/cost/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

var Cost = wire.NewSet(
	NewCostService,
)

func NewCostService(
	logger *slog.Logger,
	billingService billing.Service,
	customerService customer.Service,
	featureConnector feature.FeatureConnector,
	meterService meter.Service,
	streamingConnector streaming.Connector,
) (cost.Service, error) {
	service, err := costservice.New(costservice.Config{
		BillingService:     billingService,
		CustomerService:    customerService,
		FeatureService:     featureConnector,
		MeterService:       meterService,
		StreamingConnector: streamingConnector,
	})
	if err != nil {
		return nil, err
	}

	return service, nil
}

func NewModelCostProvider(
	logger *slog.Logger,
) (*modelcost.ModelCostProvider, error) {
	logger = logger.WithGroup("cost-provider")

	return modelcost.NewModelCostProvider(
		modelcost.CostProviderConfig{
			Logger:  logger,
			Timeout: 3 * time.Second,
		},
	)
}
