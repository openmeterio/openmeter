package transactions

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func testCurrencyReference(code currencyx.Code) currencies.CurrencyReference {
	if !code.IsCustom() {
		return currencies.NewCurrencyReference(code)
	}

	id := fmt.Sprintf("%x", sha256.Sum256([]byte(code)))[:26]
	reference, err := currencies.ParseCurrencyReference([]byte(fmt.Sprintf("custom|v1|%s|%s|2", code, id)))
	if err != nil {
		panic(err)
	}

	return reference
}

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
		AccountService: e.Deps.ResolversService,
		AccountCatalog: e.Deps.AccountService,
		BalanceQuerier: e.Deps.HistoricalLedger,
	}
}

func (e *transactionsTestEnv) resolve(t *testing.T, templates ...TransactionTemplate) []ledger.TransactionInput {
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

func (e *transactionsTestEnv) resolveAndCommit(t *testing.T, templates ...TransactionTemplate) []ledger.TransactionInput {
	t.Helper()

	inputs := e.resolve(t, templates...)
	e.commit(t, inputs...)
	return inputs
}

func (e *transactionsTestEnv) fundPriority(t *testing.T, priority int, amount int64) ledger.SubAccount {
	t.Helper()

	return e.fundPriorityWithCostBasis(t, priority, amount, nil, nil)
}

func (e *transactionsTestEnv) fundPriorityWithCostBasis(t *testing.T, priority int, amount int64, costBasis *alpacadecimal.Decimal, sourceChargeID *string) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       e.CurrencyReference(),
		CostBasis:      costBasis,
		CreditPriority: priority,
	})
	require.NoError(t, err)

	e.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:             e.Now(),
			Amount:         alpacadecimal.NewFromInt(amount),
			Currency:       e.CurrencyReference(),
			CostBasis:      costBasis,
			SourceChargeID: sourceChargeID,
			CreditPriority: &priority,
		},
		AuthorizeCustomerReceivablePaymentTemplate{
			At:             e.Now(),
			Amount:         alpacadecimal.NewFromInt(amount),
			Currency:       e.CurrencyReference(),
			CostBasis:      costBasis,
			SourceChargeID: sourceChargeID,
		},
		SettleCustomerReceivableFromPaymentTemplate{
			At:             e.Now(),
			Amount:         alpacadecimal.NewFromInt(amount),
			Currency:       e.CurrencyReference(),
			CostBasis:      costBasis,
			SourceChargeID: sourceChargeID,
		},
	)

	return subAccount
}
