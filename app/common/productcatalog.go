package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/cost"
	costadapter "github.com/openmeterio/openmeter/openmeter/cost/adapter"
	costservice "github.com/openmeterio/openmeter/openmeter/cost/service"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcatalogpgadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	addonadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/adapter"
	addonservice "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/featureresolver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	planaddonadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/adapter"
	planaddonservice "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/service"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

var ProductCatalog = wire.NewSet(
	Feature,
	Cost,
	Plan,
	Addon,
	PlanAddon,
)

var Feature = wire.NewSet(
	NewFeatureConnector,
	NewFeatureResolver,
)

var Cost = wire.NewSet(
	NewCostService,
)

var Plan = wire.NewSet(
	NewPlanService,
)

var Addon = wire.NewSet(
	NewAddonService,
)

var PlanAddon = wire.NewSet(
	NewPlanAddonService,
)

func NewFeatureConnector(
	logger *slog.Logger,
	db *entdb.Client,
	meterService meter.Service,
	publisher eventbus.Publisher,
) feature.FeatureConnector {
	featureRepo := productcatalogpgadapter.NewPostgresFeatureRepo(db, logger)
	return feature.NewFeatureConnector(featureRepo, meterService, publisher)
}

var NewFeatureResolver = featureresolver.New

func NewCostService(
	featureConnector feature.FeatureConnector,
	meterService meter.Service,
	streamingConnector streaming.Connector,
	llmcostService llmcost.Service,
) (cost.Service, error) {
	adapter := costadapter.New(featureConnector, meterService, streamingConnector, llmcostService)

	return costservice.New(costservice.Config{
		Adapter: adapter,
	})
}

func NewPlanService(
	logger *slog.Logger,
	db *entdb.Client,
	featureResolver productcatalog.FeatureResolver,
	taxCodeService taxcode.Service,
	publisher eventbus.Publisher,
) (plan.Service, error) {
	adapter, err := planadapter.New(planadapter.Config{
		Client: db,
		Logger: logger.With("subsystem", "productcatalog.plan"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize plan adapter: %w", err)
	}

	return planservice.New(planservice.Config{
		Adapter:         adapter,
		FeatureResolver: featureResolver,
		TaxCode:         taxCodeService,
		Logger:          logger.With("subsystem", "productcatalog.plan"),
		Publisher:       publisher,
	})
}

func NewAddonService(
	logger *slog.Logger,
	db *entdb.Client,
	featureResolver productcatalog.FeatureResolver,
	taxCodeService taxcode.Service,
	publisher eventbus.Publisher,
) (addon.Service, error) {
	adapter, err := addonadapter.New(addonadapter.Config{
		Client: db,
		Logger: logger.With("subsystem", "productcatalog.addon"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize add-on adapter: %w", err)
	}

	return addonservice.New(addonservice.Config{
		Adapter:         adapter,
		FeatureResolver: featureResolver,
		TaxCode:         taxCodeService,
		Logger:          logger.With("subsystem", "productcatalog.addon"),
		Publisher:       publisher,
	})
}

func NewPlanAddonService(
	logger *slog.Logger,
	db *entdb.Client,
	planService plan.Service,
	addonService addon.Service,
	publisher eventbus.Publisher,
) (planaddon.Service, error) {
	adapter, err := planaddonadapter.New(planaddonadapter.Config{
		Client: db,
		Logger: logger.With("subsystem", "productcatalog.planaddon"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize add-on adapter: %w", err)
	}

	return planaddonservice.New(planaddonservice.Config{
		Adapter:   adapter,
		Plan:      planService,
		Addon:     addonService,
		Logger:    logger.With("subsystem", "productcatalog.addon"),
		Publisher: publisher,
	})
}
