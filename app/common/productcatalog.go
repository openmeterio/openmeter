package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	productcatalogpgadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

var ProductCatalog = wire.NewSet(
	Feature,
	Plan,
)

var Feature = wire.NewSet(
	NewFeatureConnector,
)

var Plan = wire.NewSet(
	NewPlanService,
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

func NewPlanService(
	logger *slog.Logger,
	db *entdb.Client,
	productCatalogConf config.ProductCatalogConfiguration,
	featureConnector feature.FeatureConnector,
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
		Adapter:   adapter,
		Feature:   featureConnector,
		Logger:    logger.With("subsystem", "productcatalog.plan"),
		Publisher: publisher,
	})
}
