package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestConvertCustomerReceivableCurrencyTemplate_CustomToFiat(t *testing.T) {
	env := newTransactionsTestEnv(t)

	sourceCurrency := currencyx.Code("ACME")
	targetCurrency := currencyx.Code("USD")
	costBasis := alpacadecimal.RequireFromString("0.5")

	inputs := env.resolveAndCommit(
		t,
		ConvertCustomerReceivableCurrencyTemplate{
			At:             env.Now(),
			SourceAmount:   alpacadecimal.NewFromInt(100),
			CostBasis:      costBasis,
			SourceCurrency: sourceCurrency,
			TargetCurrency: targetCurrency,
		},
	)
	require.Len(t, inputs, 1)

	sourceReceivable, err := env.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       sourceCurrency,
		Source:                         &targetCurrency,
		CostBasis:                      &costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	require.NoError(t, err)
	targetReceivable, err := env.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       targetCurrency,
		CostBasis:                      &costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	require.NoError(t, err)
	sourceBrokerage, err := env.BusinessAccounts.BrokerageAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:  sourceCurrency,
		Source:    &targetCurrency,
		CostBasis: &costBasis,
	})
	require.NoError(t, err)
	targetBrokerage, err := env.BusinessAccounts.BrokerageAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:  targetCurrency,
		CostBasis: &costBasis,
	})
	require.NoError(t, err)

	require.Equal(t, &targetCurrency, sourceReceivable.Route().Source)
	require.Nil(t, targetReceivable.Route().Source)
	require.Equal(t, &targetCurrency, sourceBrokerage.Route().Source)
	require.Nil(t, targetBrokerage.Route().Source)
	require.Equal(t, float64(100), env.SumBalance(t, sourceReceivable).InexactFloat64())
	require.Equal(t, float64(-50), env.SumBalance(t, targetReceivable).InexactFloat64())
	require.Equal(t, float64(-100), env.SumBalance(t, sourceBrokerage).InexactFloat64())
	require.Equal(t, float64(50), env.SumBalance(t, targetBrokerage).InexactFloat64())
}
