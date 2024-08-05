package productcatalogdriver

import (
	productcatalogdriver "github.com/openmeterio/openmeter/internal/productcatalog/driver"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateFeatureHandler = productcatalogdriver.CreateFeatureHandler
	DeleteFeatureHandler = productcatalogdriver.DeleteFeatureHandler
	GetFeatureHandler    = productcatalogdriver.GetFeatureHandler
	ListFeaturesHandler  = productcatalogdriver.ListFeaturesHandler
	FeatureHandler       = productcatalogdriver.FeatureHandler
)

func NewFeatureHandler(
	connector productcatalog.FeatureConnector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) FeatureHandler {
	return productcatalogdriver.NewFeatureHandler(connector, namespaceDecoder, options...)
}
