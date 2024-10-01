package appcustomer

import (
	"context"

	appcustomerentity "github.com/openmeterio/openmeter/openmeter/appcustomer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type TxAdapter interface {
	AppCustomerAdapter
}

type Adapter interface {
	AppCustomerAdapter

	entutils.TxCreator
}

type AppCustomerAdapter interface {
	UpsertAppCustomer(ctx context.Context, input appcustomerentity.UpsertAppCustomerInput) error
}
