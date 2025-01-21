package customerservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ customer.Service = (*Service)(nil)

func (s *Service) ListCustomers(ctx context.Context, input customerentity.ListCustomersInput) (pagination.PagedResponse[customerentity.Customer], error) {
	return s.adapter.ListCustomers(ctx, input)
}

func (s *Service) CreateCustomer(ctx context.Context, input customerentity.CreateCustomerInput) (*customerentity.Customer, error) {
	return s.adapter.CreateCustomer(ctx, input)
}

func (s *Service) DeleteCustomer(ctx context.Context, input customerentity.DeleteCustomerInput) error {
	return s.adapter.DeleteCustomer(ctx, input)
}

func (s *Service) GetCustomer(ctx context.Context, input customerentity.GetCustomerInput) (*customerentity.Customer, error) {
	return s.adapter.GetCustomer(ctx, input)
}

func (s *Service) UpdateCustomer(ctx context.Context, input customerentity.UpdateCustomerInput) (*customerentity.Customer, error) {
	return s.adapter.UpdateCustomer(ctx, input)
}

func (s *Service) GetEntitlementValue(ctx context.Context, input customerentity.GetEntitlementValueInput) (entitlement.EntitlementValue, error) {
	cust, err := s.GetCustomer(ctx, customerentity.GetCustomerInput{
		Namespace: input.ID.Namespace,
		ID:        input.ID.ID,
	})
	if err != nil {
		return nil, err
	}

	if len(cust.UsageAttribution.SubjectKeys) != 1 {
		return nil, &models.GenericConflictError{
			Inner: fmt.Errorf("customer %s has multiple subject keys", input.ID.ID),
		}
	}

	subjectKey := cust.UsageAttribution.SubjectKeys[0]

	return s.entitlementConnector.GetEntitlementValue(ctx, input.ID.Namespace, subjectKey, input.FeatureKey, clock.Now())
}
