package productcatalog

import (
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/meter"
)

func NewFeatureConnector(
	db FeatureRepo,
	meterRepo meter.Repository,
) FeatureConnector {
	return productcatalog.NewFeatureConnector(db, meterRepo)
}
