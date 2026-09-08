package chargeadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func TestPartialBackfillKeepsOlderAdvanceAheadOfNewerAdvance(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)
	ctx := t.Context()
	start := env.Now()
	adapter, err := lineageadapter.New(lineageadapter.Config{Client: env.DB})
	require.NoError(t, err)

	// Given A=20 was collected before B=20.
	chargeA, chargeB := ulid.Make().String(), ulid.Make().String()
	env.createAdvanceExposureForSpend(t, alpacadecimal.NewFromInt(20), nil, &chargeA)
	clock.FreezeTime(start.Add(time.Hour))
	defer clock.UnFreeze()
	env.createAdvanceExposureForSpend(t, alpacadecimal.NewFromInt(20), nil, &chargeB)
	roots, err := env.lineage.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
		Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency.Reference(),
	})
	require.NoError(t, err)
	require.Len(t, roots, 2)
	require.Equal(t, chargeA, roots[0].ChargeID)
	require.Equal(t, chargeB, roots[1].ChargeID)

	// When a purchase backs 10 of A, its uncovered remainder becomes a new row,
	// created after B's original segment. Use the real grant and persistence path.
	clock.FreezeTime(start.Add(2 * time.Hour))
	defer clock.UnFreeze()
	costBasis := alpacadecimal.NewFromFloat(0.5)
	firstPurchase := env.newExternalCharge(alpacadecimal.NewFromInt(10), costBasis)
	firstPurchase.ID = ulid.Make().String()
	_, err = env.grantCredits(t, firstPurchase)
	require.NoError(t, err)

	// Then A's remaining 10 still precedes B's 20, even with reversed input IDs.
	uncovered := creditrealization.LineageSegmentStateAdvanceUncovered
	segments, err := adapter.ListActiveSegments(ctx, lineage.ListActiveSegmentsInput{
		LineageIDs: []string{roots[1].ID, roots[0].ID}, State: &uncovered,
	})
	require.NoError(t, err)
	require.Len(t, segments, 2)
	olderRemainder, err := env.DB.CreditRealizationLineageSegment.Get(ctx, segments[0].ID)
	require.NoError(t, err)
	newerOriginal, err := env.DB.CreditRealizationLineageSegment.Get(ctx, segments[1].ID)
	require.NoError(t, err)
	require.True(t, olderRemainder.CreatedAt.After(newerOriginal.CreatedAt), "A remainder is newer than B, but stays first")
	require.Equal(t, roots[0].ID, segments[0].LineageID)
	require.Equal(t, float64(10), segments[0].Amount.InexactFloat64())
	require.Equal(t, roots[1].ID, segments[1].LineageID)
	require.Equal(t, float64(20), segments[1].Amount.InexactFloat64())

	// The next purchase of 15 books A=10 and B=5 in both representations.
	clock.FreezeTime(start.Add(3 * time.Hour))
	defer clock.UnFreeze()
	secondPurchase := env.newExternalCharge(alpacadecimal.NewFromInt(15), costBasis)
	secondPurchase.ID = ulid.Make().String()
	result, err := env.grantCredits(t, secondPurchase)
	require.NoError(t, err)
	require.Len(t, result.BackfillAllocations, 2)
	require.Equal(t, segments[0].ID, result.BackfillAllocations[0].SegmentID)
	require.Equal(t, float64(10), result.BackfillAllocations[0].Amount.InexactFloat64())
	require.Equal(t, segments[1].ID, result.BackfillAllocations[1].SegmentID)
	require.Equal(t, float64(5), result.BackfillAllocations[1].Amount.InexactFloat64())
	env.requireAccountSourceSpendBucketAmounts(t, env.accruedSubAccount(t, costBasis).AccountID().ID, map[string]float64{
		sourceSpendChargeKey(&firstPurchase.ID, &chargeA):  10,
		sourceSpendChargeKey(&secondPurchase.ID, &chargeA): 10,
		sourceSpendChargeKey(&secondPurchase.ID, &chargeB): 5,
		sourceSpendChargeKey(nil, &chargeB):                15,
	})
}

func TestAdvanceBackfillSortsSuppliedRootsByCollectionTimeThenID(t *testing.T) {
	for _, sameTime := range []bool{false, true} {
		name := "collection time wins over ID"
		if sameTime {
			name = "ID breaks equal collection times"
		}
		t.Run(name, func(t *testing.T) {
			env := newCreditPurchaseHandlerTestEnv(t)
			ctx := t.Context()
			start := env.Now()
			chargeA, chargeB := ulid.Make().String(), ulid.Make().String()
			bTime := start.Add(time.Hour)
			if sameTime {
				bTime = start
			}

			// Given B has the smaller root ID, but A has the earlier collection
			// timestamp unless the two timestamps tie.
			clock.FreezeTime(bTime)
			defer clock.UnFreeze()
			env.createAdvanceExposureForSpend(t, alpacadecimal.NewFromInt(20), nil, &chargeB)
			clock.FreezeTime(start)
			defer clock.UnFreeze()
			env.createAdvanceExposureForSpend(t, alpacadecimal.NewFromInt(20), nil, &chargeA)
			roots, err := env.lineage.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
				Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency.Reference(),
			})
			require.NoError(t, err)
			require.Len(t, roots, 2)
			byCharge := lo.KeyBy(roots, func(root lineage.Lineage) string { return root.ChargeID })
			require.Less(t, byCharge[chargeB].ID, byCharge[chargeA].ID)
			require.True(t, byCharge[chargeA].CreatedAt.Equal(start))
			require.True(t, byCharge[chargeB].CreatedAt.Equal(bTime))
			for i := range roots {
				roots[i].OriginalTransactionGroupID = env.originalAdvanceGroups[roots[i].RootRealizationID]
			}
			slices.Reverse(roots)
			originalOrder := lo.Map(roots, func(root lineage.Lineage, _ int) string { return root.ID })

			// When the ledger receives reverse-ordered roots for a purchase of 25.
			clock.FreezeTime(start.Add(2 * time.Hour))
			defer clock.UnFreeze()
			costBasis := alpacadecimal.NewFromFloat(0.5)
			purchase := env.newExternalCharge(alpacadecimal.NewFromInt(25), costBasis)
			result, err := transaction.Run(ctx, enttx.NewCreator(env.DB), func(ctx context.Context) (creditpurchase.CreditGrantResult, error) {
				result, err := env.handler.OnCreditPurchaseInitiated(ctx, creditpurchase.CreditGrantInput{Charge: purchase, AdvanceLineages: roots})
				if err != nil {
					return result, err
				}
				err = env.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{
					Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency,
					Amount: purchase.Intent.CreditAmount, BackingTransactionGroupID: result.TransactionGroupID,
					Allocations: result.BackfillAllocations,
				})
				return result, err
			})
			require.NoError(t, err)

			// Then the canonical first occurrence receives 20 and the other 5,
			// regardless of caller order. Equal timestamps use the smaller root ID.
			first, second := chargeA, chargeB
			if sameTime {
				first, second = chargeB, chargeA
			}
			require.Len(t, result.BackfillAllocations, 2)
			require.Equal(t, byCharge[first].Segments[0].ID, result.BackfillAllocations[0].SegmentID)
			require.Equal(t, float64(20), result.BackfillAllocations[0].Amount.InexactFloat64())
			require.Equal(t, byCharge[second].Segments[0].ID, result.BackfillAllocations[1].SegmentID)
			require.Equal(t, float64(5), result.BackfillAllocations[1].Amount.InexactFloat64())
			require.Equal(t, originalOrder, lo.Map(roots, func(root lineage.Lineage, _ int) string { return root.ID }))
			env.requireAccountSourceSpendBucketAmounts(t, env.CustomerAccounts.AccruedAccount.ID().ID, map[string]float64{
				sourceSpendChargeKey(&purchase.ID, &first):  20,
				sourceSpendChargeKey(&purchase.ID, &second): 5,
				sourceSpendChargeKey(nil, &second):          15,
			})
		})
	}
}
