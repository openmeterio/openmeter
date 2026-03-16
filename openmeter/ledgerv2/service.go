package ledgerv2

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledgerv2/account"
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

// ResolverDependencies are the dependencies required to resolve transactions
type ResolverDependencies struct {
	AccountService account.Service
}

// ResolutionScope is the scope for which we resolve the transaction templates
type ResolutionScope struct {
	CustomerID customer.CustomerID
	Namespace  string
}

// TransactionTemplate is a template for transactions
type TransactionResolver interface {
	Resolve(ctx context.Context, resolutionScope ResolutionScope, resolvers ResolverDependencies) (historical.TransactionInput, error)
}

type TransactionGroupInput struct {
	Namespace    string
	Transactions []transactions.Resolver
}

type TransactionService interface {
	CommitTransactionGroup(ctx context.Context, group TransactionGroupInput) (historical.TransactionGroup, error)
}
