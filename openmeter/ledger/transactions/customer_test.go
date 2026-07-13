package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestIssueCustomerReceivableTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)

	priority := 7
	inputs := env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(50),
			Currency:       env.Currency,
			CreditPriority: &priority,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, priority)).Equal(alpacadecimal.NewFromInt(50)))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-50)))
}

func TestIssueCustomerReceivableTemplate_DefaultPriority(t *testing.T) {
	env := newTransactionsTestEnv(t)

	inputs := env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(15),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).Equal(alpacadecimal.NewFromInt(15)))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-15)))
}

func TestIssueCustomerReceivableTemplate_CustomSource(t *testing.T) {
	env := newTransactionsTestEnv(t)
	env.Currency = currencyx.Code("ACME")
	source := currencyx.Code("USD")
	costBasis := alpacadecimal.RequireFromString("0.5")
	priority := 7

	inputs := env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(50),
			Currency:       env.Currency,
			Source:         &source,
			CostBasis:      &costBasis,
			CreditPriority: &priority,
		},
	)
	require.Len(t, inputs, 1)

	fbo, err := env.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       env.Currency,
		Source:         &source,
		CostBasis:      &costBasis,
		CreditPriority: priority,
	})
	require.NoError(t, err)
	receivable, err := env.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       env.Currency,
		Source:                         &source,
		CostBasis:                      &costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	require.NoError(t, err)

	require.Equal(t, &source, fbo.Route().Source)
	require.Equal(t, &source, receivable.Route().Source)
	require.Equal(t, float64(50), env.SumBalance(t, fbo).InexactFloat64())
	require.Equal(t, float64(-50), env.SumBalance(t, receivable).InexactFloat64())
}

func TestAuthorizeCustomerReceivablePaymentTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)

	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.Currency,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		AuthorizeCustomerReceivablePaymentTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccountWithStatus(t, ledger.TransactionAuthorizationStatusAuthorized)).Equal(alpacadecimal.NewFromInt(-40)))
	require.True(t, env.SumBalance(t, env.WashSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).Equal(alpacadecimal.NewFromInt(40)))
}

func TestCoverCustomerReceivableTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)

	priority := 3
	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(45),
			Currency:       env.Currency,
			CreditPriority: &priority,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		CoverCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(45),
			Currency:       env.Currency,
			CreditPriority: &priority,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, priority)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
}

func TestSettleCustomerReceivableFromPaymentTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)

	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.Currency,
		},
		AuthorizeCustomerReceivablePaymentTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.Currency,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		SettleCustomerReceivableFromPaymentTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccountWithStatus(t, ledger.TransactionAuthorizationStatusAuthorized)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.WashSubAccount(t)).Equal(alpacadecimal.NewFromInt(-40)))
}

func TestAttributeCustomerAdvanceReceivableCostBasisTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)
	purchasedCostBasis := alpacadecimal.NewFromInt(1)
	sourceChargeID := testChargeID(1)
	spendChargeID := testChargeID(2)

	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:            env.Now(),
			Amount:        alpacadecimal.NewFromInt(40),
			Currency:      env.Currency,
			SpendChargeID: &spendChargeID,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		AttributeCustomerAdvanceReceivableCostBasisTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(40),
			Currency:       env.Currency,
			CostBasis:      &purchasedCostBasis,
			SourceChargeID: &sourceChargeID,
			SpendChargeID:  &spendChargeID,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &purchasedCostBasis)).Equal(alpacadecimal.NewFromInt(-40)))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
	requireReceivableBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceChargeID, &spendChargeID): -40,
	})
}

func requireReceivableBalanceBuckets(t *testing.T, env *transactionsTestEnv, expected map[string]float64) {
	t.Helper()

	receivableAccount, ok := env.CustomerAccounts.ReceivableAccount.(accountIdentifier)
	require.True(t, ok)
	receivableAccountID := receivableAccount.ID().ID

	buckets, err := env.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
		Namespace: env.Namespace,
		Filters: ledger.Filters{
			AccountID: &receivableAccountID,
			Route: ledger.RouteFilter{
				Currency: env.Currency,
			},
		},
		GroupBy: []string{
			ledger.BalanceBucketGroupBySourceChargeID,
			ledger.BalanceBucketGroupBySpendChargeID,
		},
	})
	require.NoError(t, err)

	actual := make(map[string]float64, len(buckets))
	for _, bucket := range buckets {
		if bucket.SettledAmount.IsZero() {
			continue
		}

		actual[sourceSpendChargeKey(
			bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID],
			bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID],
		)] = bucket.SettledAmount.InexactFloat64()
	}
	require.Equal(t, expected, actual)
}
