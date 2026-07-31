package customerbalance

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

// TestCreditVoidCustomCurrency proves VoidCreditPurchase correctly reverses an
// unused custom-currency FBO credit. VoidCreditPurchase is route-preserving:
// it corrects the original IssueCustomerReceivableTemplate transaction found
// by SourceChargeID rather than reconstructing a route from scratch, so it
// does not need its own CustomCurrency field to work for a single managed
// custom currency instance.
func TestCreditVoidCustomCurrency(t *testing.T) {
	env := newTestEnv(t)
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	env.Currency = customCurrencyValue.GetCode()
	env.CustomCurrency = &customCurrencyValue

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

	fbo := env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)
	require.True(t, env.SumBalance(t, fbo).Equal(amount))

	result, err := env.CreditVoidService.VoidCreditPurchase(t.Context(), creditvoid.VoidCreditPurchaseInput{
		CustomerID: env.CustomerID,
		ChargeID:   chargeID,
		Currency:   env.Currency,
	})
	require.NoError(t, err)
	require.True(t, result.Amount.Equal(amount), "voided amount: %s", result.Amount)

	require.True(t, env.SumBalance(t, fbo).Equal(alpacadecimal.Zero))
}
