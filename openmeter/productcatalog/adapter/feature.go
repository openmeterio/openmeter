// Package adapter is deprecated. Use productcatalog/feature/adapter instead.
package adapter

import (
	"log/slog"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	featureadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/feature/adapter"
)

// NewPostgresFeatureRepo creates a new feature adapter.
// Deprecated: Use featureadapter.NewPostgresFeatureRepo or featureadapter.New instead.
func NewPostgresFeatureRepo(db *entdb.Client, logger *slog.Logger) feature.Adapter {
	return featureadapter.NewPostgresFeatureRepo(db, logger)
}

// MapFeatureEntity maps a database feature entity to a feature model.
// Deprecated: Use featureadapter.MapFeatureEntity instead.
var MapFeatureEntity = featureadapter.MapFeatureEntity
