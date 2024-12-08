package billingservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

func (s *Service) validateCustomerForUpdate(ctx context.Context, customerID customerentity.CustomerID) error {
	if err := customerID.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	cust, err := s.customerService.GetCustomer(ctx, customerentity.GetCustomerInput(customerID))
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
