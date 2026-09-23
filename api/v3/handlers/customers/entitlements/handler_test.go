package customersentitlements

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

const (
	testNamespace     = "test-ns"
	testCustomerID    = "01K5A4V2X8Q9Z7M3N6P1R4S8T2"
	testEntitlementID = "01K5A4V2X8Q9Z7M3N6P1R4S8T3"
)

type fakeService struct {
	history     func(ctx context.Context, input entitlement.GetCustomerEntitlementHistoryInput) (entitlement.CustomerEntitlementHistory, error)
	reset       func(ctx context.Context, input entitlement.ResetCustomerEntitlementUsageInput) error
	createGrant func(ctx context.Context, input entitlement.CreateCustomerEntitlementGrantInput) (grant.Grant, error)
}

func (f fakeService) CreateCustomerEntitlement(context.Context, entitlement.CreateCustomerEntitlementInput) (*entitlement.Entitlement, error) {
	return nil, errors.New("not implemented")
}

func (f fakeService) OverrideCustomerEntitlement(context.Context, entitlement.OverrideCustomerEntitlementInput) (*entitlement.Entitlement, error) {
	return nil, errors.New("not implemented")
}

func (f fakeService) GetCustomerEntitlementHistory(ctx context.Context, input entitlement.GetCustomerEntitlementHistoryInput) (entitlement.CustomerEntitlementHistory, error) {
	return f.history(ctx, input)
}

func (f fakeService) GetCustomerEntitlement(context.Context, entitlement.GetCustomerEntitlementInput) (*entitlement.Entitlement, error) {
	return nil, errors.New("not implemented")
}

func (f fakeService) ListCustomerEntitlements(context.Context, entitlement.ListCustomerEntitlementsInput) (pagination.Result[entitlement.Entitlement], error) {
	return pagination.Result[entitlement.Entitlement]{}, errors.New("not implemented")
}

func (f fakeService) ResetCustomerEntitlementUsage(ctx context.Context, input entitlement.ResetCustomerEntitlementUsageInput) error {
	return f.reset(ctx, input)
}

func (f fakeService) DeleteCustomerEntitlement(context.Context, entitlement.DeleteCustomerEntitlementInput) error {
	return errors.New("not implemented")
}

func (f fakeService) ListCustomerEntitlementGrants(context.Context, entitlement.ListCustomerEntitlementGrantsInput) (pagination.Result[grant.Grant], error) {
	return pagination.Result[grant.Grant]{}, errors.New("not implemented")
}

func (f fakeService) CreateCustomerEntitlementGrant(ctx context.Context, input entitlement.CreateCustomerEntitlementGrantInput) (grant.Grant, error) {
	return f.createGrant(ctx, input)
}
