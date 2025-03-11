package customerservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ customer.Service = (*Service)(nil)

func (s *Service) ListCustomers(ctx context.Context, input customer.ListCustomersInput) (pagination.PagedResponse[customer.Customer], error) {
	return s.adapter.ListCustomers(ctx, input)
}

func (s *Service) CreateCustomer(ctx context.Context, input customer.CreateCustomerInput) (*customer.Customer, error) {
	if err := s.requestValidatorRegistry.ValidateCreateCustomer(ctx, input); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	return s.adapter.CreateCustomer(ctx, input)
}

func (s *Service) DeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	if err := s.requestValidatorRegistry.ValidateDeleteCustomer(ctx, input); err != nil {
		return models.NewGenericValidationError(err)
	}

	return s.adapter.DeleteCustomer(ctx, input)
}

func (s *Service) GetCustomer(ctx context.Context, input customer.GetCustomerInput) (*customer.Customer, error) {
	return s.adapter.GetCustomer(ctx, input)
}

func (s *Service) UpdateCustomer(ctx context.Context, input customer.UpdateCustomerInput) (*customer.Customer, error) {
	if err := s.requestValidatorRegistry.ValidateUpdateCustomer(ctx, input); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	return s.adapter.UpdateCustomer(ctx, input)
}

func (s *Service) GetEntitlementValue(ctx context.Context, input customer.GetEntitlementValueInput) (entitlement.EntitlementValue, error) {
	cust, err := s.GetCustomer(ctx, customer.GetCustomerInput{
		Namespace: input.ID.Namespace,
		ID:        input.ID.ID,
	})
	if err != nil {
		return nil, err
	}

	if len(cust.UsageAttribution.SubjectKeys) != 1 {
		return nil, models.NewGenericConflictError(
			fmt.Errorf("customer %s has multiple subject keys", input.ID.ID),
		)
	}

	subjectKey := cust.UsageAttribution.SubjectKeys[0]

	val, err := s.entitlementConnector.GetEntitlementValue(ctx, input.ID.Namespace, subjectKey, input.FeatureKey, clock.Now())
	if err != nil {
		if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
			return entitlement.NoAccessValue{}, nil
		}

		return nil, err
	}

	return val, nil
}

func (s *Service) CustomerExists(ctx context.Context, customer customer.CustomerID) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.CustomerExists(ctx, customer)
	})
}
