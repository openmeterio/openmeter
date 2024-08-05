package productcatalogdriver

import productcatalogdriver "github.com/openmeterio/openmeter/internal/productcatalog/driver"

// requests
type (
	CreateFeatureHandlerRequest = productcatalogdriver.CreateFeatureHandlerRequest
	DeleteFeatureHandlerRequest = productcatalogdriver.DeleteFeatureHandlerRequest
	GetFeatureHandlerRequest    = productcatalogdriver.GetFeatureHandlerRequest
	ListFeaturesHandlerRequest  = productcatalogdriver.ListFeaturesHandlerRequest
)

// responses
type (
	CreateFeatureHandlerResponse = productcatalogdriver.CreateFeatureHandlerResponse
	DeleteFeatureHandlerResponse = productcatalogdriver.DeleteFeatureHandlerResponse
	GetFeatureHandlerResponse    = productcatalogdriver.GetFeatureHandlerResponse
	ListFeaturesHandlerResponse  = productcatalogdriver.ListFeaturesHandlerResponse
)

// params
type (
	DeleteFeatureHandlerParams = productcatalogdriver.DeleteFeatureHandlerParams
	GetFeatureHandlerParams    = productcatalogdriver.GetFeatureHandlerParams
	ListFeaturesHandlerParams  = productcatalogdriver.ListFeaturesHandlerParams
)
