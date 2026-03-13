package transactions

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// CustomerTransactionTemplate is a template for customer scoped transactions
type CustomerTransactionTemplate interface {
	Resolver

	// Resolve resolves the template's intent for a concrete customer
	resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error)
}

// OrgTransactionTemplate is a template for organization scoped transactions
type OrgTransactionTemplate interface {
	Resolver

	// Resolve resolves the template's intent for a given organization
	resolve(ctx context.Context, namespace string, resolvers ResolverDependencies) (ledger.TransactionInput, error)
}
