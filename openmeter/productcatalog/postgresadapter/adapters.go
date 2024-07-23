package postgresadapter

import (
	"log/slog"

	"github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter"
	"github.com/openmeterio/openmeter/openmeter/entdb"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func NewPostgresFeatureDBAdapter(db *entdb.DBClient, logger *slog.Logger) productcatalog.FeatureRepo {
	return postgresadapter.NewPostgresFeatureRepo(db, logger)
}
