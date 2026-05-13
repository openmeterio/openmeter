package transactions

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ResolverDependencies are the dependencies required to resolve transactions
type ResolverDependencies struct {
	AccountService ledger.AccountResolver
	AccountCatalog ledger.AccountCatalog
	BalanceQuerier ledger.BalanceQuerier
}

// ResolutionScope is the scope for which we resolve the transaction templates
type ResolutionScope struct {
	CustomerID customer.CustomerID
	Namespace  string
}

func (s ResolutionScope) Validate() error {
	if s.CustomerID.Namespace != "" && s.Namespace != "" && s.CustomerID.Namespace != s.Namespace {
		return ledger.ErrResolutionScopeInvalid.WithAttrs(models.Attributes{
			"reason":                "namespace_mismatch",
			"customer_id_namespace": s.CustomerID.Namespace,
			"namespace":             s.Namespace,
		})
	}

	return nil
}

func (s ResolutionScope) validateForCustomerTransaction() error {
	if s.CustomerID.Namespace == "" {
		return ledger.ErrResolutionScopeInvalid.WithAttrs(models.Attributes{
			"reason": "customer_id_namespace_required",
		})
	}

	if s.CustomerID.ID == "" {
		return ledger.ErrResolutionScopeInvalid.WithAttrs(models.Attributes{
			"reason": "customer_id_required",
		})
	}

	return nil
}

func (s ResolutionScope) validateForOrgTransaction() error {
	if s.Namespace == "" {
		return ledger.ErrResolutionScopeInvalid.WithAttrs(models.Attributes{
			"reason": "namespace_required",
		})
	}

	return nil
}

// ResolveTransactions resolves a list of transaction templates into a list of transaction inputs
func ResolveTransactions(
	ctx context.Context,
	deps ResolverDependencies,
	scope ResolutionScope,
	templates ...TransactionTemplate,
) ([]ledger.TransactionInput, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}

	var inputs []ledger.TransactionInput

	for _, template := range templates {
		if err := template.Validate(); err != nil {
			return nil, err
		}

		switch typ := any(template).(type) {
		case CustomerTransactionTemplate:
			if err := scope.validateForCustomerTransaction(); err != nil {
				return nil, err
			}

			tx, err := typ.resolve(ctx, scope.CustomerID, deps)
			if err != nil {
				return nil, err
			}

			inputs, err = appendResolvedTemplateTransaction(inputs, tx, template, ledger.TransactionDirectionForward)
			if err != nil {
				return nil, err
			}
		case OrgTransactionTemplate:
			if err := scope.validateForOrgTransaction(); err != nil {
				return nil, err
			}

			tx, err := typ.resolve(ctx, scope.Namespace, deps)
			if err != nil {
				return nil, err
			}

			inputs, err = appendResolvedTemplateTransaction(inputs, tx, template, ledger.TransactionDirectionForward)
			if err != nil {
				return nil, err
			}
		default:
			return nil, ledger.ErrResolutionTemplateUnknown.WithAttrs(models.Attributes{
				"template_type": fmt.Sprintf("%T", typ),
			})
		}
	}

	return inputs, nil
}
