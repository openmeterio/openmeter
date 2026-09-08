package subscription_test

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestSubscriptionCustomCurrencyBackfillAcrossPeriods(t *testing.T) {
	// given: two funded monthly fees followed by two separately collected advances.
	clock.FreezeTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()
	f := setupCustomCurrencyFlatFeeSubscription(t)
	ctx := t.Context()
	for month := time.February; month <= time.May; month++ {
		at := time.Date(2026, month, 1, 0, 0, 0, 0, time.UTC)
		clock.FreezeTime(at)
		require.NoError(t, f.subscriptionSyncService.SyncByView(ctx, f.view, at))
		_, err := f.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: f.view.Customer.GetID()})
		require.NoError(t, err)
	}
	ids := listSubscriptionChargeIDs(t, f.testDeps, f.view.Subscription.ID)
	require.Len(t, ids, 4)
	chargeByMonth := map[time.Month]string{}
	for _, id := range ids {
		result, err := f.chargesService.GetByID(ctx, charges.GetByIDInput{ChargeID: chargesmeta.ChargeID{Namespace: f.view.Subscription.Namespace, ID: id}})
		require.NoError(t, err)
		charge, err := result.AsFlatFeeCharge()
		require.NoError(t, err)
		chargeByMonth[charge.Intent.GetEffectiveServicePeriod().From.Month()] = id
	}
	require.Len(t, chargeByMonth, 4)

	// when: a late purchase can cover all of April and only half of May.
	at := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(at)
	period := timeutil.ClosedPeriod{From: at, To: at}
	fiat, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)
	purchases, err := f.chargesService.Create(ctx, charges.CreateInput{
		Namespace: f.view.Subscription.Namespace,
		Intents: charges.ChargeIntents{charges.NewChargeIntent(creditpurchase.Intent{
			Intent: chargesmeta.Intent{ManagedBy: billing.ManuallyManagedLine, CustomerID: f.view.Customer.ID, Currency: f.currency},
			IntentMutableFields: creditpurchase.IntentMutableFields{
				IntentMutableFields: chargesmeta.IntentMutableFields{Name: "Monthly advance backfill", ServicePeriod: period, FullServicePeriod: period, BillingPeriod: period},
				CreditAmount:        decimal.NewFromInt(15), EffectiveAt: &at,
				Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus}),
			},
			CostBasis: creditpurchase.NewCostBasis(costbasis.NewIntent(costbasis.ManualIntent{FiatCurrency: fiat, Rate: decimal.NewFromFloat(0.5)})),
		})},
	})
	require.NoError(t, err)
	require.Len(t, purchases, 1)
	purchaseID, err := purchases[0].GetChargeID()
	require.NoError(t, err)
	for _, status := range []payment.Status{payment.StatusAuthorized, payment.StatusSettled} {
		_, err = f.chargesService.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{ChargeID: purchaseID, TargetPaymentState: status})
		require.NoError(t, err)
	}

	// then: ledger and lineage agree per period, not just in the customer total.
	expected := map[string]float64{chargeByMonth[time.April]: 10, chargeByMonth[time.May]: 5}
	buckets, err := f.ledgerDeps.HistoricalLedger.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: f.view.Subscription.Namespace,
		Filters: ledger.Filters{
			AccountID: lo.ToPtr(f.business.EarningsAccount.ID().ID), SourceChargeID: mo.Some(&purchaseID.ID),
			Route: ledger.RouteFilter{Currency: f.currency.Reference(), CostBasis: mo.Some(lo.ToPtr(decimal.NewFromFloat(0.5))), CostBasisCurrency: mo.Some(lo.ToPtr(currencyx.Code("USD")))},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySpendChargeID},
	})
	require.NoError(t, err)
	booked := map[string]float64{}
	for _, bucket := range buckets {
		booked[lo.FromPtr(bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID])] += bucket.SettledAmount.InexactFloat64()
	}
	require.Equal(t, expected, booked)
	roots, err := f.lineageService.LoadLineagesByCustomer(ctx, legacylineage.LoadLineagesByCustomerInput{Namespace: f.view.Subscription.Namespace, CustomerID: f.view.Customer.ID, Currency: f.currency.Reference()})
	require.NoError(t, err)
	recognized := map[string]float64{}
	uncovered := map[string]float64{}
	for _, root := range roots {
		if root.OriginKind != creditrealization.LineageOriginKindAdvance {
			continue
		}
		for _, segment := range root.Segments {
			switch segment.State {
			case creditrealization.LineageSegmentStateEarningsRecognized:
				require.Equal(t, creditrealization.LineageSegmentStateAdvanceBackfilled, lo.FromPtr(segment.SourceState))
				recognized[root.ChargeID] += segment.Amount.InexactFloat64()
			case creditrealization.LineageSegmentStateAdvanceUncovered:
				uncovered[root.ChargeID] += segment.Amount.InexactFloat64()
			default:
				t.Fatalf("unexpected advance state %q", segment.State)
			}
		}
	}
	require.Equal(t, expected, recognized)
	require.Equal(t, map[string]float64{chargeByMonth[time.May]: 5}, uncovered)
	requireCustomCurrencyAccountBalance(t, f, f.business.EarningsAccount, 35)
	beforeEntries, err := f.DBDeps.DBClient.LedgerEntry.Query().Count(ctx)
	require.NoError(t, err)
	_, err = f.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: f.view.Customer.GetID()})
	require.NoError(t, err)
	afterEntries, err := f.DBDeps.DBClient.LedgerEntry.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, beforeEntries, afterEntries)
	assertNoSubscriptionInvoices(t, f.testDeps, f.view.Customer.ID)
}
