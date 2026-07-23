package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestIssueCustomerReceivableTemplateValidate(t *testing.T) {
	err := (IssueCustomerReceivableTemplate{
		Amount:                 alpacadecimal.NewFromInt(-1),
		Currency:               currencyx.Code("AC|ME"),
		ExchangeSourceCurrency: lo.ToPtr(currencyx.Code("POINTS")),
		CostBasis:              lo.ToPtr(alpacadecimal.NewFromInt(-1)),
		CreditPriority:         lo.ToPtr(-1),
	}).Validate()
	require.True(t, models.IsGenericValidationError(err))
	require.ErrorContains(t, err, "amount must be positive")
	require.ErrorContains(t, err, "at is required")
	require.ErrorContains(t, err, "currency:")
	require.ErrorContains(t, err, "exchange source currency:")
	require.ErrorContains(t, err, "cost basis:")
	require.ErrorContains(t, err, "credit priority:")
}

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

func TestIssueCustomerReceivableTemplate_CustomPrecisionCommit(t *testing.T) {
	// given:
	// - a custom-currency amount already materialized to the registry precision
	// when:
	// - the issuance passes through the real historical ledger
	// then:
	// - the ledger preserves the amount instead of applying fiat precision
	env := newTransactionsTestEnv(t)
	env.Currency = currencyx.Code("ACME")
	amount, err := alpacadecimal.NewFromString("10.001")
	require.NoError(t, err)

	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   amount,
			Currency: env.Currency,
		},
	)

	require.Equal(t, float64(10.001), env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).InexactFloat64())
	require.Equal(t, float64(-10.001), env.SumBalance(t, env.ReceivableSubAccount(t)).InexactFloat64())
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
