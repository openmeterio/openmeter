package httpdriver

import (
	"github.com/openmeterio/openmeter/internal/productcatalog/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateFeatureHandler = httpdriver.CreateFeatureHandler
	DeleteFeatureHandler = httpdriver.DeleteFeatureHandler
	GetFeatureHandler    = httpdriver.GetFeatureHandler
	ListFeaturesHandler  = httpdriver.ListFeaturesHandler
	FeatureHandler       = httpdriver.FeatureHandler
)

func NewFeatureHandler(
	connector productcatalog.FeatureConnector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) FeatureHandler {
	return httpdriver.NewFeatureHandler(connector, namespaceDecoder, options...)
}
