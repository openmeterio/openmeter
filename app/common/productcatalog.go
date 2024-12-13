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
)

var ProductCatalog = wire.NewSet(
	wire.FieldsOf(new(config.Configuration), "ProductCatalog"),

	Feature,
	Plan,
)

var Feature = wire.NewSet(
	NewFeatureConnector,
)

var Plan = wire.NewSet(
	NewPlanService,
)

func NewFeatureConnector(logger *slog.Logger, db *entdb.Client, meterRepo meter.Repository) feature.FeatureConnector {
	// TODO: remove this check after enabled by default
	if db == nil {
		return nil
	}

	featureRepo := productcatalogpgadapter.NewPostgresFeatureRepo(db, logger)
	return feature.NewFeatureConnector(featureRepo, meterRepo)
}

func NewPlanService(
	logger *slog.Logger,
	db *entdb.Client,
	productCatalogConf config.ProductCatalogConfiguration,
	featureConnector feature.FeatureConnector,
) (plan.Service, error) {
	// TODO: remove this check after enabled by default
	if db == nil {
		return nil, nil
	}

	if !productCatalogConf.Enabled {
		return nil, nil
	}

	adapter, err := planadapter.New(planadapter.Config{
		Client: db,
		Logger: logger.With("subsystem", "productcatalog.plan"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize plan adapter: %w", err)
	}

	return planservice.New(planservice.Config{
		Feature: featureConnector,
		Adapter: adapter,
		Logger:  logger.With("subsystem", "productcatalog.plan"),
	})
}
