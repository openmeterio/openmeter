package appcustomer

import (
	"context"

	appcustomerentity "github.com/openmeterio/openmeter/openmeter/appcustomer/entity"
)

type Service interface {
	AppService
}

type AppService interface {
	UpsertAppCustomer(ctx context.Context, input appcustomerentity.UpsertAppCustomerInput) error
}
