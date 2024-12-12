package common

import (
	"log/slog"

	"github.com/openmeterio/openmeter/app/config"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	productcatalogpgadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

func NewFeatureConnector(logger *slog.Logger, db *entdb.Client, appsConfig config.AppsConfiguration, meterRepo meter.Repository) feature.FeatureConnector {
	featureRepo := productcatalogpgadapter.NewPostgresFeatureRepo(db, logger)
	return feature.NewFeatureConnector(featureRepo, meterRepo)
}
