package features

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListFeatures() ListFeaturesHandler
	GetFeature() GetFeatureHandler
	CreateFeature() CreateFeatureHandler
	DeleteFeature() DeleteFeatureHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	connector        feature.FeatureConnector
	llmcostService   llmcost.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	connector feature.FeatureConnector,
	llmcostService llmcost.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		connector:        connector,
		llmcostService:   llmcostService,
		options:          options,
	}
}
