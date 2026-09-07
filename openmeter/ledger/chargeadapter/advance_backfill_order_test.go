package chargeadapter_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/pkg/clock"
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
