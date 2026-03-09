// Package productcatalogdriver is deprecated. Use productcatalog/feature/httpdriver instead.
package productcatalogdriver

import (
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature/httpdriver"
)

// MapFeatureToResponse maps a feature to an API response.
// Deprecated: Use httpdriver.MapFeatureToResponse instead.
func MapFeatureToResponse(f feature.Feature) (api.Feature, error) {
	return httpdriver.MapFeatureToResponse(f)
}

// MapFeatureCreateInputsRequest maps an API request to feature create inputs.
// Deprecated: Use httpdriver.MapFeatureCreateInputsRequest instead.
func MapFeatureCreateInputsRequest(namespace string, f api.FeatureCreateInputs) (feature.CreateFeatureInputs, error) {
	return httpdriver.MapFeatureCreateInputsRequest(namespace, f)
}
