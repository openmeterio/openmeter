package transactions

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// ResolverDependencies are the dependencies required to resolve transactions
type ResolverDependencies struct {
	AccountService   ledger.AccountResolver
	DimensionService ledger.DimensionResolver
}

// ResolutionScope is the scope for which we resolve the transaction templates
type ResolutionScope struct {
	CustomerID customer.CustomerID
	Namespace  string
}

func (s ResolutionScope) Validate() error {
	if s.CustomerID.Namespace != "" && s.Namespace != "" && s.CustomerID.Namespace != s.Namespace {
		return fmt.Errorf("customer ID namespace and namespace must match")
	}

	return nil
}

func (s ResolutionScope) validateForCustomerTransaction() error {
	if s.CustomerID.Namespace == "" {
		return fmt.Errorf("customer ID namespace is required")
	}

	if s.CustomerID.ID == "" {
		return fmt.Errorf("customer ID is required")
	}

	return nil
}

func (s ResolutionScope) validateForOrgTransaction() error {
	if s.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	return nil
}

type (
	guard    bool // private type guard
	Resolver interface {
		typeGuard() guard
	}
)

// ResolveTransactions resolves a list of transaction templates into a list of transaction inputs
func ResolveTransactions(
	ctx context.Context,
	deps ResolverDependencies,
	scope ResolutionScope,
	templates ...Resolver,
) ([]ledger.TransactionInput, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}

	var inputs []ledger.TransactionInput

	for _, template := range templates {
		switch typ := any(template).(type) {
		case CustomerTransactionTemplate:
			if err := scope.validateForCustomerTransaction(); err != nil {
				return nil, err
			}

			tx, err := typ.resolve(ctx, scope.CustomerID, deps)
			if err != nil {
				return nil, err
			}

			inputs = append(inputs, tx)
		case OrgTransactionTemplate:
			if err := scope.validateForOrgTransaction(); err != nil {
				return nil, err
			}

			tx, err := typ.resolve(ctx, scope.Namespace, deps)
			if err != nil {
				return nil, err
			}

			inputs = append(inputs, tx)
		default:
			return nil, fmt.Errorf("unknown template type: %T", typ)
		}
	}

	return inputs, nil
}
