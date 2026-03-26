package features

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListFeatures() ListFeaturesHandler
	GetFeature() GetFeatureHandler
	CreateFeature() CreateFeatureHandler
	UpdateFeature() UpdateFeatureHandler
	DeleteFeature() DeleteFeatureHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	connector        feature.FeatureConnector
	meterService     meter.Service
	llmcostService   llmcost.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	connector feature.FeatureConnector,
	meterService meter.Service,
	llmcostService llmcost.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		connector:        connector,
		meterService:     meterService,
		llmcostService:   llmcostService,
		options:          options,
	}
}
