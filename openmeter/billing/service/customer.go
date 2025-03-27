package billingservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

func (s *Service) validateCustomerForUpdate(ctx context.Context, customerID customer.CustomerID) error {
	if err := customerID.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	cust, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customerID,
	})
	if err != nil {
		return err
	}

	if cust.DeletedAt != nil {
		return billing.ValidationError{
			Err: billing.ErrCustomerDeleted,
		}
	}

	return nil
}
