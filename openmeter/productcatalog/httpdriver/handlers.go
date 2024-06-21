package httpdriver

import (
	"github.com/openmeterio/openmeter/internal/productcatalog/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type CreateFeatureHandler = httpdriver.CreateFeatureHandler
type DeleteFeatureHandler = httpdriver.DeleteFeatureHandler
type GetFeatureHandler = httpdriver.GetFeatureHandler
type ListFeaturesHandler = httpdriver.ListFeaturesHandler
type FeatureHandler = httpdriver.FeatureHandler

func NewFeatureHandler(
	connector productcatalog.FeatureConnector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) FeatureHandler {
	return httpdriver.NewFeatureHandler(connector, namespaceDecoder, options...)
}
