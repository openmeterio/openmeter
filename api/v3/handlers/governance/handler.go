package governance

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	QueryGovernanceAccess() QueryGovernanceAccessHandler
}

type handler struct {
	resolveNamespace   func(ctx context.Context) (string, error)
	customerService    customer.Service
	entitlementService entitlement.Service
	featureConnector   feature.FeatureConnector
	options            []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	customerService customer.Service,
	entitlementService entitlement.Service,
	featureConnector feature.FeatureConnector,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:   resolveNamespace,
		customerService:    customerService,
		entitlementService: entitlementService,
		featureConnector:   featureConnector,
		options:            options,
	}
}
