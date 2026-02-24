package transactions

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type Resolvers struct {
	AccountService   ledger.AccountResolver
	DimensionService ledger.DimensionResolver
}

// CustomerTransactionTemplate is a template for customer scoped transactions
type CustomerTransactionTemplate interface {
	// Resolve resolves the template's intent for a concrete customer
	Resolve(ctx context.Context, customerID customer.CustomerID, resolvers Resolvers) (ledger.TransactionInput, error)
}

// OrgTransactionTemplate is a template for organization scoped transactions
type OrgTransactionTemplate interface {
	// Resolve resolves the template's intent for a given organization
	Resolve(ctx context.Context, namespace string, resolvers Resolvers) (ledger.TransactionInput, error)
}

func ResolveCustomerTransactionTemplate(template CustomerTransactionTemplate, ctx context.Context, customerID customer.CustomerID, resolvers Resolvers) (ledger.TransactionInput, error) {
	return template.Resolve(ctx, customerID, resolvers)
}
