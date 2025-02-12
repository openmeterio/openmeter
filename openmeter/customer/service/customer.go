package customerservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
)

var _ customer.Service = (*Service)(nil)

func (s *Service) ListCustomers(ctx context.Context, input customer.ListCustomersInput) (pagination.PagedResponse[customer.Customer], error) {
	return s.adapter.ListCustomers(ctx, input)
}

func (s *Service) CreateCustomer(ctx context.Context, input customer.CreateCustomerInput) (*customer.Customer, error) {
	return s.adapter.CreateCustomer(ctx, input)
}

func (s *Service) DeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	return s.adapter.DeleteCustomer(ctx, input)
}

func (s *Service) GetCustomer(ctx context.Context, input customer.GetCustomerInput) (*customer.Customer, error) {
	return s.adapter.GetCustomer(ctx, input)
}

func (s *Service) FindCustomer(ctx context.Context, namespace string, customerRef ref.IDOrKey) (*customer.Customer, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}

	if err := customerRef.Validate(); err != nil {
		return nil, err
	}

	if customerRef.ID != "" {
		return s.GetCustomer(ctx, customer.GetCustomerInput{
			Namespace: namespace,
			ID:        customerRef.ID,
		})
	}
	custs, err := s.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace:      namespace,
		IncludeDeleted: false,
		Key:            lo.ToPtr(customerRef.Key),
		Page:           pagination.NewPage(1, 1),
	})
	if err != nil {
		return nil, err
	}

	if custs.TotalCount == 0 {
		return nil, customer.NotFoundError{CustomerID: customer.CustomerID{
			Namespace: namespace,
			ID:        customerRef.Key,
		}}
	}

	if custs.TotalCount == 1 {
		return &custs.Items[0], nil
	}

	return nil, &models.GenericConflictError{Inner: fmt.Errorf("multiple (%d) customers found with key %s while expecting one", custs.TotalCount, customerRef.Key)}
}

func (s *Service) UpdateCustomer(ctx context.Context, input customer.UpdateCustomerInput) (*customer.Customer, error) {
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
		return nil, &models.GenericConflictError{
			Inner: fmt.Errorf("customer %s has multiple subject keys", input.ID.ID),
		}
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
