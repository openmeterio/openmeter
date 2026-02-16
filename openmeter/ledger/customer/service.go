package customer

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type Service interface {
	GetCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error)
}
