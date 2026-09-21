package subscription_test

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSubscriptionMigrationPreservesCollectionOrigins(t *testing.T) {
	// Given a paid, recognized February fee of 10 backed by a purchase of 20.
	clock.FreezeTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()
	f := setupCustomCurrencyFlatFeeSubscription(t)
	ctx := t.Context()
	clock.FreezeTime(f.view.Subscription.ActiveFrom)
	require.NoError(t, f.subscriptionSyncService.SyncByView(ctx, f.view, clock.Now()))

	_, err := f.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: f.view.Customer.GetID()})
	require.NoError(t, err)

	query := ledger.BalanceBucketQuery{
		Namespace: f.view.Subscription.Namespace,
		Filters: ledger.Filters{
			AccountID: lo.ToPtr(f.business.EarningsAccount.ID().ID),
			Route:     ledger.RouteFilter{Currency: f.currency.Reference()},
		},
		GroupBy: []string{ledger.BalanceBucketGroupByCollectionOriginID, ledger.BalanceBucketGroupBySourceChargeID, ledger.BalanceBucketGroupBySpendChargeID},
	}
	before, err := f.ledgerDeps.HistoricalLedger.GetBalanceBuckets(ctx, query)
	require.NoError(t, err)
	require.Len(t, before, 1)
	require.Equal(t, float64(10), before[0].SettledAmount.InexactFloat64())

	originalOrigin := lo.FromPtr(before[0].GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID])
	require.NotEmpty(t, originalOrigin)

	// When an in-place migration doubles the price halfway through February.
	clock.FreezeTime(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC))
	original, err := f.PlanService.GetPlan(ctx, plan.GetPlanInput{
		NamespacedID: models.NamespacedID{
			Namespace: f.view.Subscription.Namespace,
			ID:        f.view.Subscription.PlanRef.Id,
		},
	})
	require.NoError(t, err)

	target := original.AsProductCatalogPlan()
	require.NoError(t, target.Phases[0].RateCards[0].ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      decimal.NewFromInt(20),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})

		return meta, nil
	}))

	nextPlan, err := f.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{Namespace: original.Namespace},
		Plan:            target,
	})
	require.NoError(t, err)

	nextPlan, err = f.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
		NamespacedID: nextPlan.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(clock.Now().Add(-time.Second)),
		},
	})
	require.NoError(t, err)

	migrated := migrateAndSyncSubscription(t, f.testDeps, f.view, &plansubscription.Plan{
		Plan: nextPlan.AsProductCatalogPlan(),
		Ref:  &nextPlan.NamespacedID,
	})
	require.Equal(t, f.view.Subscription.ID, migrated.Subscription.ID)

	_, err = f.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: f.view.Customer.GetID()})
	require.NoError(t, err)

	// Then correction leaves 5 in the old origin and the new half-month fee gets its own origin.
	after, err := f.ledgerDeps.HistoricalLedger.GetBalanceBuckets(ctx, query)
	require.NoError(t, err)
	require.Len(t, after, 2)

	var newOrigin string

	for _, bucket := range after {
		require.Equal(t, before[0].GroupByValues[ledger.BalanceBucketGroupBySourceChargeID], bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID])

		origin := lo.FromPtr(bucket.GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID])
		if origin == originalOrigin {
			require.Equal(t, float64(5), bucket.SettledAmount.InexactFloat64())
			require.Equal(t, before[0].GroupByValues[ledger.BalanceBucketGroupBySpendChargeID], bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID])
		} else {
			newOrigin = origin
			require.NotEqual(t, before[0].GroupByValues[ledger.BalanceBucketGroupBySpendChargeID], bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID])
			require.Equal(t, float64(10), bucket.SettledAmount.InexactFloat64())
		}
	}

	require.NotEmpty(t, newOrigin)

	requireCustomCurrencyAccountBalance(t, f, f.accounts.FBOAccount, 5)

	// When cancellation halves the replacement's remaining service, only its origin decreases.
	clock.FreezeTime(time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC))
	_, err = f.subscriptionService.Cancel(ctx, f.view.Subscription.NamespacedID, subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)})
	require.NoError(t, err)

	canceled, err := f.subscriptionService.GetView(ctx, f.view.Subscription.NamespacedID)
	require.NoError(t, err)
	require.NoError(t, f.subscriptionSyncService.SyncByView(ctx, canceled, clock.Now()))

	final, err := f.ledgerDeps.HistoricalLedger.GetBalanceBuckets(ctx, query)
	require.NoError(t, err)
	require.Len(t, final, 2)

	remainingByOrigin := map[string]float64{}

	for _, bucket := range final {
		require.Equal(t, before[0].GroupByValues[ledger.BalanceBucketGroupBySourceChargeID], bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID])

		remainingByOrigin[lo.FromPtr(bucket.GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID])] += bucket.SettledAmount.InexactFloat64()
	}

	require.Equal(t, map[string]float64{
		originalOrigin: 5,
		newOrigin:      5,
	}, remainingByOrigin)

	requireCustomCurrencyAccountBalance(t, f, f.accounts.FBOAccount, 10)

	// Then retrying reconciliation neither posts entries nor changes the origin balances.
	entries, err := f.DBDeps.DBClient.LedgerEntry.Query().Count(ctx)
	require.NoError(t, err)
	require.NoError(t, f.subscriptionSyncService.SyncByView(ctx, canceled, clock.Now()))

	_, err = f.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: f.view.Customer.GetID()})
	require.NoError(t, err)

	retriedEntries, err := f.DBDeps.DBClient.LedgerEntry.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, entries, retriedEntries)

	retried, err := f.ledgerDeps.HistoricalLedger.GetBalanceBuckets(ctx, query)
	require.NoError(t, err)
	require.Equal(t, final, retried)
}
