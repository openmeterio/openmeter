// Package productcatalogdriver is deprecated. Use productcatalog/feature/httpdriver instead.
package productcatalogdriver

import (
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature/httpdriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

// FeatureHandler is an alias for httpdriver.FeatureHandler.
// Deprecated: Use httpdriver.FeatureHandler instead.
type FeatureHandler = httpdriver.FeatureHandler

// NewFeatureHandler creates a new feature HTTP handler.
// Deprecated: Use httpdriver.NewFeatureHandler instead.
func NewFeatureHandler(
	connector feature.Service,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	llmcostService llmcost.Service,
	options ...httptransport.HandlerOption,
) FeatureHandler {
	return httpdriver.NewFeatureHandler(connector, namespaceDecoder, llmcostService, options...)
}
