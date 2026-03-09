package featurecost

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	QueryFeatureCost() QueryFeatureCostHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	costService      cost.Service
	featureConnector feature.FeatureConnector
	meterService     meter.Service
	customerService  customer.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	costService cost.Service,
	featureConnector feature.FeatureConnector,
	meterService meter.Service,
	customerService customer.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		costService:      costService,
		featureConnector: featureConnector,
		meterService:     meterService,
		customerService:  customerService,
		options:          options,
	}
}
