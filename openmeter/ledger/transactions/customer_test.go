package transactions

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestIssueCustomerReceivableTemplateValidate(t *testing.T) {
	err := (IssueCustomerReceivableTemplate{
		Amount:            alpacadecimal.NewFromInt(-1),
		Currency:          currencies.NewCurrencyReference(currencyx.Code("AC|ME")),
		CostBasisCurrency: lo.ToPtr(currencyx.Code("POINTS")),
		CostBasis:         lo.ToPtr(alpacadecimal.NewFromInt(-1)),
		CreditPriority:    lo.ToPtr(-1),
	}).Validate()
	require.True(t, models.IsGenericValidationError(err))
	require.ErrorContains(t, err, "amount must be positive")
	require.ErrorContains(t, err, "at is required")
	require.ErrorContains(t, err, "currency:")
	require.ErrorContains(t, err, "cost basis currency:")
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
			Currency:       env.CurrencyReference(),
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
			Currency: env.CurrencyReference(),
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
			Currency: env.CurrencyReference(),
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
			Currency: env.CurrencyReference(),
		},
	)

	inputs := env.resolveAndCommit(
		t,
		AuthorizeCustomerReceivablePaymentTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.CurrencyReference(),
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccountWithStatus(t, ledger.TransactionAuthorizationStatusAuthorized)).Equal(alpacadecimal.NewFromInt(-40)))
	require.True(t, env.SumBalance(t, env.WashSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).Equal(alpacadecimal.NewFromInt(40)))
}

// TestAuthorizeCustomerReceivablePaymentTemplateValidate_RequiresCostBasisCurrencyForKnownCostBasis
// pins the cost-basis "financial fact" invariant (see the Prepaid Credit Cost
// Basis design doc): a resolved cost basis is only meaningful together with
// the fiat it was quoted against, so any template touching a cost-basis
// bucket must supply that fiat whenever the currency is custom and the cost
// basis is known — the same rule IssueCustomerReceivableTemplate enforces.
// Catching this at Validate() keeps the failure local and cheap instead of
// surfacing as a route-lookup mismatch deep inside resolve().
func TestAuthorizeCustomerReceivablePaymentTemplateValidate_RequiresCostBasisCurrencyForKnownCostBasis(t *testing.T) {
	costBasis := alpacadecimal.NewFromFloat(0.25)

	authorizeErr := (AuthorizeCustomerReceivablePaymentTemplate{
		At:        time.Now(),
		Amount:    alpacadecimal.NewFromInt(40),
		Currency:  currencies.NewCurrencyReference(currencyx.Code("ACME")),
		CostBasis: &costBasis,
	}).Validate()
	require.Error(t, authorizeErr)
	require.ErrorContains(t, authorizeErr, "cost basis currency:")

	settleErr := (SettleCustomerReceivableFromPaymentTemplate{
		At:        time.Now(),
		Amount:    alpacadecimal.NewFromInt(40),
		Currency:  currencies.NewCurrencyReference(currencyx.Code("ACME")),
		CostBasis: &costBasis,
	}).Validate()
	require.Error(t, settleErr)
	require.ErrorContains(t, settleErr, "cost basis currency:")
}

// TestAuthorizeAndSettleCustomerReceivableTemplates_CarryCostBasisCurrency
// covers the positive lifecycle for a cost-basis-tracked custom-currency
// receivable: AuthorizeCustomerReceivablePaymentTemplate and
// SettleCustomerReceivableFromPaymentTemplate now carry CostBasisCurrency,
// the same field IssueCustomerReceivableTemplate and
// AttributeCustomerAdvanceReceivableCostBasisTemplate already had. Per the
// Prepaid Credit Cost Basis design doc, a resolved cost basis is a snapshot
// taken once and never recomputed, so authorize/settle must reuse the exact
// fiat pairing recorded at issuance to land on the same sub-account rather
// than a disconnected one.
func TestAuthorizeAndSettleCustomerReceivableTemplates_CarryCostBasisCurrency(t *testing.T) {
	env := newTransactionsTestEnv(t)
	env.Currency = currencyx.Code("ACME")
	costBasisCurrency := currencyx.Code("USD")
	costBasis := alpacadecimal.NewFromFloat(0.25)

	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:                env.Now(),
			Amount:            alpacadecimal.NewFromInt(40),
			Currency:          env.CurrencyReference(),
			CostBasisCurrency: &costBasisCurrency,
			CostBasis:         &costBasis,
		},
	)

	openReceivable, err := env.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       env.CurrencyReference(),
		CostBasisCurrency:              &costBasisCurrency,
		CostBasis:                      &costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	require.NoError(t, err)
	require.Equal(t, float64(-40), env.SumBalance(t, openReceivable).InexactFloat64())

	// when:
	// - authorizing the payment for the same currency/cost-basis, passing
	//   the same CostBasisCurrency the issuance snapshot recorded
	// then:
	// - resolution succeeds and moves the balance into the authorized route
	env.resolveAndCommit(
		t,
		AuthorizeCustomerReceivablePaymentTemplate{
			At:                env.Now(),
			Amount:            alpacadecimal.NewFromInt(40),
			Currency:          env.CurrencyReference(),
			CostBasisCurrency: &costBasisCurrency,
			CostBasis:         &costBasis,
		},
	)

	authorizedReceivable, err := env.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       env.CurrencyReference(),
		CostBasisCurrency:              &costBasisCurrency,
		CostBasis:                      &costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusAuthorized,
	})
	require.NoError(t, err)
	require.Equal(t, float64(0), env.SumBalance(t, openReceivable).InexactFloat64())
	require.Equal(t, float64(-40), env.SumBalance(t, authorizedReceivable).InexactFloat64())

	// when:
	// - settling the authorized payment, again passing the same
	//   CostBasisCurrency
	// then:
	// - resolution succeeds and clears the authorized route against wash
	env.resolveAndCommit(
		t,
		SettleCustomerReceivableFromPaymentTemplate{
			At:                env.Now(),
			Amount:            alpacadecimal.NewFromInt(40),
			Currency:          env.CurrencyReference(),
			CostBasisCurrency: &costBasisCurrency,
			CostBasis:         &costBasis,
		},
	)

	wash, err := env.BusinessAccounts.WashAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:          env.CurrencyReference(),
		CostBasisCurrency: &costBasisCurrency,
		CostBasis:         &costBasis,
	})
	require.NoError(t, err)
	require.Equal(t, float64(0), env.SumBalance(t, authorizedReceivable).InexactFloat64())
	require.Equal(t, float64(-40), env.SumBalance(t, wash).InexactFloat64())
}

func TestCoverCustomerReceivableTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)

	priority := 3
	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(45),
			Currency:       env.CurrencyReference(),
			CreditPriority: &priority,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		CoverCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(45),
			Currency:       env.CurrencyReference(),
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
			Currency: env.CurrencyReference(),
		},
		AuthorizeCustomerReceivablePaymentTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.CurrencyReference(),
		},
	)

	inputs := env.resolveAndCommit(
		t,
		SettleCustomerReceivableFromPaymentTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.CurrencyReference(),
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
			Currency:      env.CurrencyReference(),
			SpendChargeID: &spendChargeID,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		AttributeCustomerAdvanceReceivableCostBasisTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(40),
			Currency:       env.CurrencyReference(),
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
				Currency: currencies.NewCurrencyReference(env.Currency),
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
