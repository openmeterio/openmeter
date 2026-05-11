package collector

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

func TestCollectCustomerFBOUsesPriorityOrder(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	collector := newTestAccrualCollector(env)

	priorityTwo := fundPriority(t, env, 2, 50)
	priorityOne := fundPriority(t, env, 1, 30)

	sources, err := collector.collectCustomerFBO(t.Context(), env.CustomerID, env.Currency, alpacadecimal.NewFromInt(60), env.Now())
	require.NoError(t, err)
	require.Len(t, sources, 2)

	require.Equal(t, priorityOne.Address().SubAccountID(), sources[0].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(30).Equal(sources[0].Amount))
	require.Equal(t, priorityTwo.Address().SubAccountID(), sources[1].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(30).Equal(sources[1].Amount))
}

func TestCollectCustomerFBOUsesSubAccountIDTieBreaker(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	collector := newTestAccrualCollector(env)

	costBasisOne := alpacadecimal.NewFromInt(1)
	costBasisTwo := alpacadecimal.NewFromInt(2)

	first := fundPriorityWithCostBasis(t, env, 1, 10, &costBasisOne)
	second := fundPriorityWithCostBasis(t, env, 1, 10, &costBasisTwo)

	sources, err := collector.collectCustomerFBO(t.Context(), env.CustomerID, env.Currency, alpacadecimal.NewFromInt(20), env.Now())
	require.NoError(t, err)
	require.Len(t, sources, 2)

	expected := []ledger.SubAccount{first, second}
	if expected[0].Address().SubAccountID() > expected[1].Address().SubAccountID() {
		expected[0], expected[1] = expected[1], expected[0]
	}

	require.Equal(t, expected[0].Address().SubAccountID(), sources[0].Address.SubAccountID())
	require.Equal(t, expected[1].Address().SubAccountID(), sources[1].Address.SubAccountID())
}

func TestCollectCustomerFBOUsesAsOfBalance(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	collector := newTestAccrualCollector(env)

	source := fundPriority(t, env, 1, 50)
	bookFutureFBOCollection(t, env, 1, 30, env.Now().AddDate(0, 0, 1))

	currentSources, err := collector.collectCustomerFBO(t.Context(), env.CustomerID, env.Currency, alpacadecimal.NewFromInt(50), env.Now())
	require.NoError(t, err)
	require.Len(t, currentSources, 1)
	require.Equal(t, source.Address().SubAccountID(), currentSources[0].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(50).Equal(currentSources[0].Amount), "current amount: %s", currentSources[0].Amount)

	futureAsOf := env.Now().AddDate(0, 0, 1)
	futureSources, err := collector.collectCustomerFBO(t.Context(), env.CustomerID, env.Currency, alpacadecimal.NewFromInt(50), futureAsOf)
	require.NoError(t, err)
	require.Len(t, futureSources, 1)
	require.Equal(t, source.Address().SubAccountID(), futureSources[0].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(20).Equal(futureSources[0].Amount), "future amount: %s", futureSources[0].Amount)
}

func newTestAccrualCollector(env *ledgertestutils.IntegrationEnv) *accrualCollector {
	return &accrualCollector{
		ledger: env.Deps.HistoricalLedger,
		deps: transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
	}
}

func fundPriority(t *testing.T, env *ledgertestutils.IntegrationEnv, priority int, amount int64) ledger.SubAccount {
	t.Helper()

	return fundPriorityWithCostBasis(t, env, priority, amount, nil)
}

func fundPriorityWithCostBasis(
	t *testing.T,
	env *ledgertestutils.IntegrationEnv,
	priority int,
	amount int64,
	costBasis *alpacadecimal.Decimal,
) ledger.SubAccount {
	t.Helper()

	subAccount, err := env.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       env.Currency,
		CostBasis:      costBasis,
		CreditPriority: priority,
	})
	require.NoError(t, err)

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: env.CustomerID,
			Namespace:  env.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(amount),
			Currency:       env.Currency,
			CostBasis:      costBasis,
			CreditPriority: &priority,
		},
		transactions.AuthorizeCustomerReceivablePaymentTemplate{
			At:        env.Now(),
			Amount:    alpacadecimal.NewFromInt(amount),
			Currency:  env.Currency,
			CostBasis: costBasis,
		},
		transactions.SettleCustomerReceivableFromPaymentTemplate{
			At:        env.Now(),
			Amount:    alpacadecimal.NewFromInt(amount),
			Currency:  env.Currency,
			CostBasis: costBasis,
		},
	)
	require.NoError(t, err)

	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
	require.NoError(t, err)

	return subAccount
}

func bookFutureFBOCollection(t *testing.T, env *ledgertestutils.IntegrationEnv, priority int, amount int64, at time.Time) {
	t.Helper()

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: env.CustomerID,
			Namespace:  env.Namespace,
		},
		transactions.CoverCustomerReceivableTemplate{
			At:             at,
			Amount:         alpacadecimal.NewFromInt(amount),
			Currency:       env.Currency,
			CreditPriority: &priority,
		},
	)
	require.NoError(t, err)

	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
	require.NoError(t, err)
}
