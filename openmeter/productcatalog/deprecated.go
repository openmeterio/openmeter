package productcatalog

import (
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

// Deprecated: Use feature.CreateFeatureInputs
type CreateFeatureInputs = feature.CreateFeatureInputs

// Deprecated: Use feature.Feature
type FeatureConnector = feature.FeatureConnector

// Deprecated: Use feature.IncludeArchivedFeature
type IncludeArchivedFeature = feature.IncludeArchivedFeature

const (
	// Deprecated: Use feature.IncludeArchivedFeatureTrue
	IncludeArchivedFeatureTrue = feature.IncludeArchivedFeatureTrue
	// Deprecated: Use feature.IncludeArchivedFeatureFalse
	IncludeArchivedFeatureFalse = feature.IncludeArchivedFeatureFalse
)

// Deprecated: Use feature.FeatureOrderBy
type FeatureOrderBy = feature.FeatureOrderBy

const (
	// Deprecated: Use feature.FeatureOrderByCreatedAt
	FeatureOrderByCreatedAt = feature.FeatureOrderByCreatedAt
	// Deprecated: Use feature.FeatureOrderByUpdatedAt
	FeatureOrderByUpdatedAt = feature.FeatureOrderByUpdatedAt
)

// Deprecated: Use feature.ListFeaturesParams
type ListFeaturesParams = feature.ListFeaturesParams

// Deprecated: Use feature.FeatureRepo
type FeatureRepo = feature.FeatureRepo

// Deprecated: Use feature.NewFeatureConnector
func NewFeatureConnector(
	featureRepo FeatureRepo,
	meterRepo meter.Repository,
) FeatureConnector {
	return feature.NewFeatureConnector(featureRepo, meterRepo)
}

// Deprecated: Use feature.Feature
type FeatureNotFoundError = feature.FeatureNotFoundError

// Deprecated: Use feature.FeatureInvalidFiltersError
type FeatureInvalidFiltersError = feature.FeatureInvalidFiltersError

// Deprecated: Use feature.FeatureWithNameAlreadyExistsError
type FeatureWithNameAlreadyExistsError = feature.FeatureWithNameAlreadyExistsError

// Deprecated: Use feature.FeatureInvalidMeterAggregationError
type FeatureInvalidMeterAggregationError = feature.FeatureInvalidMeterAggregationError

// Deprecated: Use feature.MeterGroupByFilters
type MeterGroupByFilters = feature.MeterGroupByFilters

// Deprecated: Use feature.Feature
type Feature = feature.Feature
