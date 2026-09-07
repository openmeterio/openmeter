package chargeadapter_test

import (
	"context"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func TestAdvanceBackfillStaleSelectionRollsBackPurchase(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)
	ctx := t.Context()

	// Given both purchases read the same uncovered 20, then the first backs 10.
	spendID := ulid.Make().String()
	env.createAdvanceExposureForSpend(t, alpacadecimal.NewFromInt(20), nil, &spendID)
	staleRoots, err := env.lineage.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
		Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency.Reference(),
	})
	require.NoError(t, err)
	require.Len(t, staleRoots, 1)
	staleRoots[0].OriginalTransactionGroupID = env.originalAdvanceGroups[staleRoots[0].RootRealizationID]
	costBasis := alpacadecimal.NewFromFloat(0.5)
	first := env.newExternalCharge(alpacadecimal.NewFromInt(10), costBasis)
	first.ID = ulid.Make().String()
	_, err = env.grantCredits(t, first)
	require.NoError(t, err)

	// When the second books 5 against its stale selection, persistence rejects
	// the closed segment rather than assigning that money to a different row.
	second := env.newExternalCharge(alpacadecimal.NewFromInt(5), costBasis)
	second.ID = ulid.Make().String()
	var rejectedGroupID string
	err = transaction.RunWithNoValue(ctx, enttx.NewCreator(env.DB), func(ctx context.Context) error {
		result, err := env.handler.OnCreditPurchaseInitiated(ctx, creditpurchase.CreditGrantInput{Charge: second, AdvanceLineages: staleRoots})
		if err != nil {
			return err
		}
		rejectedGroupID = result.TransactionGroupID
		return env.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{
			Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency,
			Amount: second.Intent.CreditAmount, BackingTransactionGroupID: result.TransactionGroupID,
			Allocations: result.BackfillAllocations,
		})
	})
	require.ErrorContains(t, err, "backfill allocation exceeds active eligible segment")

	// Then the rejected ledger group is gone and only the first purchase's 10
	// remains backed. Retrying with fresh lineage can safely back another 5.
	require.NotEmpty(t, rejectedGroupID)
	_, err = env.DB.LedgerTransactionGroup.Get(ctx, rejectedGroupID)
	require.True(t, entdb.IsNotFound(err), "rejected purchase group must roll back: %v", err)
	env.requireAccountSourceSpendBucketAmounts(t, env.CustomerAccounts.AccruedAccount.ID().ID, map[string]float64{
		sourceSpendChargeKey(&first.ID, &spendID): 10,
		sourceSpendChargeKey(nil, &spendID):       10,
	})
	_, err = env.grantCredits(t, second)
	require.NoError(t, err)
	env.requireAccountSourceSpendBucketAmounts(t, env.CustomerAccounts.AccruedAccount.ID().ID, map[string]float64{
		sourceSpendChargeKey(&first.ID, &spendID):  10,
		sourceSpendChargeKey(&second.ID, &spendID): 5,
		sourceSpendChargeKey(nil, &spendID):        5,
	})
}
