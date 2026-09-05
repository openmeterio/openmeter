package customerbalance

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

// TestCreditVoidCustomCurrency proves VoidCreditPurchase correctly reverses
// unused custom-currency FBO credit and that transaction history keeps reused
// display codes separated by their immutable managed currency identities.
func TestCreditVoidCustomCurrency(t *testing.T) {
	env := newTestEnv(t)
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	env.Currency = customCurrencyValue.GetCode()
	env.CustomCurrency = &customCurrencyValue
	otherCustomCurrency := currenciestestutils.NewCustomCurrency(t, "ACME", 2)

	// source_charge_id is a fixed char(26) column (ULID-sized); it must be a
	// real ULID or Postgres blank-pads it, breaking the exact string match
	// VoidCreditPurchase uses to locate the original issue transaction.
	chargeID := ulid.Make().String()
	amount := alpacadecimal.NewFromInt(100)

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
			Amount:         amount,
			Currency:       env.CurrencyReference(),
			SourceChargeID: &chargeID,
		},
	)
	require.NoError(t, err)
	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
	require.NoError(t, err)

	otherChargeID := ulid.Make().String()
	otherAmount := alpacadecimal.NewFromInt(50)
	otherInputs, err := transactions.ResolveTransactions(
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
			Amount:         otherAmount,
			Currency:       otherCustomCurrency.Reference(),
			SourceChargeID: &otherChargeID,
		},
	)
	require.NoError(t, err)
	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, otherInputs...))
	require.NoError(t, err)

	fbo := env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)
	require.True(t, env.SumBalance(t, fbo).Equal(amount))

	result, err := env.CreditVoidService.VoidCreditPurchase(t.Context(), creditvoid.VoidCreditPurchaseInput{
		CustomerID: env.CustomerID,
		ChargeID:   chargeID,
		Currency:   env.Currency,
	})
	require.NoError(t, err)
	require.True(t, result.Amount.Equal(amount), "voided amount: %s", result.Amount)
	otherResult, err := env.CreditVoidService.VoidCreditPurchase(t.Context(), creditvoid.VoidCreditPurchaseInput{
		CustomerID: env.CustomerID,
		ChargeID:   otherChargeID,
		Currency:   env.Currency,
	})
	require.NoError(t, err)
	require.True(t, otherResult.Amount.Equal(otherAmount), "other voided amount: %s", otherResult.Amount)

	require.True(t, env.SumBalance(t, fbo).Equal(alpacadecimal.Zero))

	voidedType := CreditTransactionTypeVoided
	history, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID:    env.CustomerID,
		Limit:         10,
		Type:          &voidedType,
		Currency:      &env.Currency,
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)
	require.Len(t, history.Items, 2)
	voidedByCurrencyID := make(map[string]CreditTransaction, len(history.Items))
	for _, item := range history.Items {
		require.Equal(t, CreditTransactionTypeVoided, item.Type)
		require.NotNil(t, item.CustomCurrencyID)
		voidedByCurrencyID[*item.CustomCurrencyID] = item
	}
	require.Equal(t, float64(-100), voidedByCurrencyID[customCurrencyValue.ID].Amount.InexactFloat64())
	require.Equal(t, float64(100), voidedByCurrencyID[customCurrencyValue.ID].Balance.Before.InexactFloat64())
	require.Equal(t, float64(0), voidedByCurrencyID[customCurrencyValue.ID].Balance.After.InexactFloat64())
	require.Equal(t, float64(-50), voidedByCurrencyID[otherCustomCurrency.ID].Amount.InexactFloat64())
	require.Equal(t, float64(50), voidedByCurrencyID[otherCustomCurrency.ID].Balance.Before.InexactFloat64())
	require.Equal(t, float64(0), voidedByCurrencyID[otherCustomCurrency.ID].Balance.After.InexactFloat64())
}
