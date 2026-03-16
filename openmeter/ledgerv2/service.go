package ledgerv2

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledgerv2/historical"
	"github.com/openmeterio/openmeter/openmeter/ledgerv2/transactions"
)

type Service struct {
	AccountProvisioningService
	TransactionService
}

type AccountProvisioningService interface {
	CreateCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (CustomerAccounts, error)
}

type TransactionGroupInput struct {
	Namespace    string
	Transactions []transactions.Resolver
}

type TransactionService interface {
	CommitTransactionGroup(ctx context.Context, group TransactionGroupInput) (historical.TransactionGroup, error)
}
