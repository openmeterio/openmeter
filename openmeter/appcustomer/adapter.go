package appcustomer

import (
	"context"

	appcustomerentity "github.com/openmeterio/openmeter/openmeter/appcustomer/entity"
	entcontext "github.com/openmeterio/openmeter/pkg/framework/entutils/context"
)

type TxAdapter interface {
	AppCustomerAdapter
}

type Adapter interface {
	AppCustomerAdapter

	DB() entcontext.DB
}

type AppCustomerAdapter interface {
	UpsertAppCustomer(ctx context.Context, input appcustomerentity.UpsertAppCustomerInput) error
}
