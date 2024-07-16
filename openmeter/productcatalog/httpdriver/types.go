package httpdriver

import "github.com/openmeterio/openmeter/internal/productcatalog/httpdriver"

// requests
type (
	CreateFeatureHandlerRequest = httpdriver.CreateFeatureHandlerRequest
	DeleteFeatureHandlerRequest = httpdriver.DeleteFeatureHandlerRequest
	GetFeatureHandlerRequest    = httpdriver.GetFeatureHandlerRequest
	ListFeaturesHandlerRequest  = httpdriver.ListFeaturesHandlerRequest
)

// responses
type (
	CreateFeatureHandlerResponse = httpdriver.CreateFeatureHandlerResponse
	DeleteFeatureHandlerResponse = httpdriver.DeleteFeatureHandlerResponse
	GetFeatureHandlerResponse    = httpdriver.GetFeatureHandlerResponse
	ListFeaturesHandlerResponse  = httpdriver.ListFeaturesHandlerResponse
)

// params
type (
	DeleteFeatureHandlerParams = httpdriver.DeleteFeatureHandlerParams
	GetFeatureHandlerParams    = httpdriver.GetFeatureHandlerParams
	ListFeaturesHandlerParams  = httpdriver.ListFeaturesHandlerParams
)
