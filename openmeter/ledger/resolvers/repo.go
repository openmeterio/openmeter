package resolvers

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// Repo manages the linking table that maps customers to their ledger accounts.
type Repo interface {
	entutils.TxCreator

	CreateCustomerAccount(ctx context.Context, input CreateCustomerAccountInput) error
	GetCustomerAccountIDs(ctx context.Context, customerID customer.CustomerID) (map[ledger.AccountType]string, error)
}

type CreateCustomerAccountInput struct {
	CustomerID  customer.CustomerID
	AccountType ledger.AccountType
	AccountID   string
}
