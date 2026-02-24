package testutils

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

type AnyCustomerTemplate struct {
	TransactionInput *AnyTransactionInput
}

var _ transactions.CustomerTransactionTemplate = (*AnyCustomerTemplate)(nil)

func (a *AnyCustomerTemplate) Resolve(ctx context.Context, customerID customer.CustomerID, resolvers transactions.Resolvers) (ledger.TransactionInput, error) {
	return a.TransactionInput, nil
}

type AnyOrgTemplate struct {
	TransactionInput *AnyTransactionInput
}

var _ transactions.OrgTransactionTemplate = (*AnyOrgTemplate)(nil)

func (a *AnyOrgTemplate) Resolve(ctx context.Context, namespace string, resolvers transactions.Resolvers) (ledger.TransactionInput, error) {
	return a.TransactionInput, nil
}
