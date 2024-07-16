package productcatalog

import "github.com/openmeterio/openmeter/internal/productcatalog"

type (
	CreateFeatureInputs                 = productcatalog.CreateFeatureInputs
	Feature                             = productcatalog.Feature
	FeatureConnector                    = productcatalog.FeatureConnector
	FeatureRepo                         = productcatalog.FeatureRepo
	FeatureInvalidFiltersError          = productcatalog.FeatureInvalidFiltersError
	FeatureInvalidMeterAggregationError = productcatalog.FeatureInvalidMeterAggregationError
	FeatureNotFoundError                = productcatalog.FeatureNotFoundError
	FeatureOrderBy                      = productcatalog.FeatureOrderBy
	FeatureWithNameAlreadyExistsError   = productcatalog.FeatureWithNameAlreadyExistsError
	ListFeaturesParams                  = productcatalog.ListFeaturesParams
)
