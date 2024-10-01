package appcustomerservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/appcustomer"
	appcustomerentity "github.com/openmeterio/openmeter/openmeter/appcustomer/entity"
)

var _ appcustomer.AppService = (*Service)(nil)

func (s *Service) UpsertAppCustomer(ctx context.Context, input appcustomerentity.UpsertAppCustomerInput) error {
	if err := input.Validate(); err != nil {
		return appcustomer.ValidationError{
			Err: err,
		}
	}

	return s.adapter.UpsertAppCustomer(ctx, input)
}
