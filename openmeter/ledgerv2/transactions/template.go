package transactions

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ledgerv2/historical"
)

type (
	guard    bool // private type guard
	Resolver interface {
		typeGuard() guard
	}
)

// TransactionTemplate is a template for transactions
type TransactionTemplate interface {
	Resolver

	// Resolve resolves the template's intent for a concrete customer
	Resolve(ctx context.Context, resolutionScope ResolutionScope, resolvers ResolverDependencies) (historical.TransactionInput, error)
}
