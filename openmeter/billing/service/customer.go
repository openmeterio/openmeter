package billingservice

import (
	"context"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

func (s *Service) validateCustomerForUpdate(ctx context.Context, customerID customerentity.CustomerID) error {
	if err := customerID.Validate(); err != nil {
		return billingentity.ValidationError{
			Err: err,
		}
	}

	cust, err := s.customerService.GetCustomer(ctx, customerentity.GetCustomerInput(customerID))
	if err != nil {
		return err
	}

	if cust.DeletedAt != nil {
		return billingentity.ValidationError{
			Err: billingentity.ErrCustomerDeleted,
		}
	}

	return nil
}
