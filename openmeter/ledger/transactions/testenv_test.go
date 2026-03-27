package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
)

type transactionsTestEnv struct {
	*ledgertestutils.IntegrationEnv
}

func newTransactionsTestEnv(t *testing.T) *transactionsTestEnv {
	return &transactionsTestEnv{
		IntegrationEnv: ledgertestutils.NewIntegrationEnv(t, "transactions"),
	}
}

func (e *transactionsTestEnv) resolverDeps() ResolverDependencies {
	return ResolverDependencies{
		AccountService:    e.Deps.ResolversService,
		SubAccountService: e.Deps.AccountService,
	}
}

func (e *transactionsTestEnv) resolve(t *testing.T, templates ...Resolver) []ledger.TransactionInput {
	t.Helper()

	inputs, err := ResolveTransactions(
		t.Context(),
		e.resolverDeps(),
		ResolutionScope{
			CustomerID: e.CustomerID,
			Namespace:  e.Namespace,
		},
		templates...,
	)
	require.NoError(t, err)

	return inputs
}

func (e *transactionsTestEnv) commit(t *testing.T, inputs ...ledger.TransactionInput) {
	t.Helper()

	_, err := e.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}

func (e *transactionsTestEnv) resolveAndCommit(t *testing.T, templates ...Resolver) []ledger.TransactionInput {
	t.Helper()

	inputs := e.resolve(t, templates...)
	e.commit(t, inputs...)
	return inputs
}

func (e *transactionsTestEnv) fundPriority(t *testing.T, priority int, amount int64) ledger.SubAccount {
	t.Helper()

	subAccount := e.FBOSubAccount(t, priority)

	e.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:             e.Now(),
			Amount:         alpacadecimal.NewFromInt(amount),
			Currency:       e.Currency,
			CreditPriority: &priority,
		},
		FundCustomerReceivableTemplate{
			At:       e.Now(),
			Amount:   alpacadecimal.NewFromInt(amount),
			Currency: e.Currency,
		},
		SettleCustomerReceivablePaymentTemplate{
			At:       e.Now(),
			Amount:   alpacadecimal.NewFromInt(amount),
			Currency: e.Currency,
		},
	)

	return subAccount
}
