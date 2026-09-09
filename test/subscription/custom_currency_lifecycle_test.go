package subscription_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	pcsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type customCurrencyFlatFeeFixture struct {
	testDeps
	currency currencies.Currency
	view     subscription.SubscriptionView
	accounts ledger.CustomerAccounts
	business ledger.BusinessAccounts
}

// Build a subscription with a paid source so cancellation exercises recognized
// credit corrections, including the source's cost basis and currency identity.
func setupCustomCurrencyFlatFeeSubscription(t *testing.T) customCurrencyFlatFeeFixture {
	t.Helper()
	const ns = "test-namespace"
	deps := setup(t, setupConfig{enableCharges: true})
	t.Cleanup(func() { deps.cleanup(t) })
	provisionSubscriptionDefaultTaxCodes(t, deps, ns)
	_, err := deps.billingService.CreateProfile(t.Context(), minimalCreateProfileInputTemplate(deps.sandboxApp.GetID()))
	require.NoError(t, err)
	credits, err := deps.CurrencyService.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(ns, "CREDITS", "Credits", "CR"))
	require.NoError(t, err)
	customer := createUSDSubscriptionCustomer(t, deps, ns, "flat-fee-lifecycle")
	business, err := deps.ledgerDeps.ResolversService.EnsureBusinessAccounts(t.Context(), ns)
	require.NoError(t, err)
	accounts, err := deps.ledgerDeps.ResolversService.CreateCustomerAccounts(t.Context(), customer.GetID())
	require.NoError(t, err)
	at := clock.Now()
	period := timeutil.ClosedPeriod{From: at, To: at}
	fiat, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)
	grants, err := deps.chargesService.Create(t.Context(), charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{charges.NewChargeIntent(creditpurchase.Intent{
			Intent: chargesmeta.Intent{ManagedBy: billing.ManuallyManagedLine, CustomerID: customer.ID, Currency: credits},
			IntentMutableFields: creditpurchase.IntentMutableFields{
				IntentMutableFields: chargesmeta.IntentMutableFields{Name: "Paid credits", ServicePeriod: period, FullServicePeriod: period, BillingPeriod: period},
				CreditAmount:        decimal.NewFromInt(20), EffectiveAt: &at,
				Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus}),
			},
			CostBasis: creditpurchase.NewCostBasis(costbasis.NewIntent(costbasis.ManualIntent{FiatCurrency: fiat, Rate: decimal.NewFromFloat(0.5)})),
		})},
	})
	require.NoError(t, err)
	require.Len(t, grants, 1)
	grantID, err := grants[0].GetChargeID()
	require.NoError(t, err)
	for _, status := range []payment.Status{payment.StatusAuthorized, payment.StatusSettled} {
		_, err = deps.chargesService.HandleCreditPurchaseExternalPaymentStateTransition(t.Context(), charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{ChargeID: grantID, TargetPaymentState: status})
		require.NoError(t, err)
	}
	month := datetime.MustParseDuration(t, "P1M")
	createdPlan, err := deps.PlanService.CreatePlan(t.Context(), plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{Namespace: ns},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{Key: "flat-fee-lifecycle", Name: "Custom flat fee", Currency: credits.Reference(), SettlementMode: productcatalog.CreditOnlySettlementMode, BillingCadence: month, ProRatingConfig: productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices}},
			Phases: []productcatalog.Phase{{PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"}, RateCards: productcatalog.RateCards{&productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{Key: "flat-fee", Name: "Custom flat fee", Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: decimal.NewFromInt(10), PaymentTerm: productcatalog.InAdvancePaymentTerm})}, BillingCadence: &month,
			}}}},
		},
	})
	require.NoError(t, err)
	createdPlan, err = deps.PlanService.PublishPlan(t.Context(), plan.PublishPlanInput{NamespacedID: createdPlan.NamespacedID, EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: lo.ToPtr(at.Add(-time.Second))}})
	require.NoError(t, err)
	created, err := createCustomCurrencySubscription(t, deps, *createdPlan, customer.ID, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), productcatalog.CreditOnlySettlementMode, subscription.CostBasisModeDynamic)
	require.NoError(t, err)
	view, err := deps.subscriptionService.GetView(t.Context(), created.NamespacedID)
	require.NoError(t, err)
	return customCurrencyFlatFeeFixture{testDeps: deps, currency: credits, view: view, accounts: accounts, business: business}
}

func TestSubscriptionCustomCurrencyRealizedCancellation(t *testing.T) {
	// given: an in-advance custom fee funded by paid credits and an old serialized event.
	clock.FreezeTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()
	f := setupCustomCurrencyFlatFeeSubscription(t)
	ctx := t.Context()
	raw, err := json.Marshal(f.view)
	require.NoError(t, err)
	var staleView subscription.SubscriptionView
	require.NoError(t, json.Unmarshal(raw, &staleView))
	horizon := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, f.subscriptionSyncService.SyncByView(ctx, f.view, horizon))
	ids := listSubscriptionChargeIDs(t, f.testDeps, f.view.Subscription.ID)
	require.GreaterOrEqual(t, len(ids), 2)
	clock.FreezeTime(f.view.Subscription.ActiveFrom)
	_, err = f.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: f.view.Customer.GetID()})
	require.NoError(t, err)
	requireCustomCurrencyAccountBalance(t, f, f.accounts.FBOAccount, 10)
	requireCustomCurrencyAccountBalance(t, f, f.business.EarningsAccount, 10)

	// when: cancellation halves the realized period, then the obsolete event is replayed.
	cancelAt := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(cancelAt)
	_, err = f.subscriptionService.Cancel(ctx, f.view.Subscription.NamespacedID, subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)})
	require.NoError(t, err)
	canceled, err := f.subscriptionService.GetView(ctx, f.view.Subscription.NamespacedID)
	require.NoError(t, err)
	event := subscription.NewCancelledEvent(ctx, canceled)
	require.NoError(t, f.subscriptionSyncService.HandleCancelledEvent(ctx, &event))

	// then: only half the original spend remains, with a single immutable correction.
	remaining := listSubscriptionChargeIDs(t, f.testDeps, f.view.Subscription.ID)
	require.Len(t, remaining, 1)
	require.Contains(t, ids, remaining[0])
	result, err := f.chargesService.GetByID(ctx, charges.GetByIDInput{ChargeID: chargesmeta.ChargeID{Namespace: f.view.Subscription.Namespace, ID: remaining[0]}})
	require.NoError(t, err)
	charge, err := result.AsFlatFeeCharge()
	require.NoError(t, err)
	require.Equal(t, cancelAt, charge.Intent.GetEffectiveServicePeriod().To)
	require.Equal(t, float64(5), charge.State.AmountAfterProration.InexactFloat64())
	requireCustomCurrencyAccountBalance(t, f, f.accounts.FBOAccount, 15)
	requireCustomCurrencyAccountBalance(t, f, f.business.EarningsAccount, 5)
	beforeLineage, err := f.lineageService.LoadLineagesByCustomer(ctx, legacylineage.LoadLineagesByCustomerInput{Namespace: f.view.Subscription.Namespace, CustomerID: f.view.Customer.ID, Currency: f.currency.Reference()})
	require.NoError(t, err)
	require.Empty(t, beforeLineage)
	query := ledger.BalanceBucketQuery{
		Namespace: f.view.Subscription.Namespace,
		Filters: ledger.Filters{
			AccountID:     lo.ToPtr(f.business.EarningsAccount.ID().ID),
			SpendChargeID: mo.Some(&remaining[0]),
			Route:         ledger.RouteFilter{Currency: f.currency.Reference()},
		},
		GroupBy: []string{ledger.BalanceBucketGroupByCollectionOriginID, ledger.BalanceBucketGroupBySourceChargeID},
	}
	beforeBuckets, err := f.ledgerDeps.HistoricalLedger.GetBalanceBuckets(ctx, query)
	require.NoError(t, err)
	require.Len(t, beforeBuckets, 1)
	require.NotEmpty(t, lo.FromPtr(beforeBuckets[0].GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID]))
	require.NotEmpty(t, lo.FromPtr(beforeBuckets[0].GroupByValues[ledger.BalanceBucketGroupBySourceChargeID]))
	require.Equal(t, float64(5), beforeBuckets[0].SettledAmount.InexactFloat64())
	beforeEntries, err := f.DBDeps.DBClient.LedgerEntry.Query().Count(ctx)
	require.NoError(t, err)
	require.NoError(t, f.subscriptionSyncService.HandleCancelledEvent(ctx, &event))
	require.NoError(t, f.subscriptionSyncService.SyncByView(ctx, staleView, horizon))
	require.Equal(t, remaining, listSubscriptionChargeIDs(t, f.testDeps, f.view.Subscription.ID))
	afterEntries, err := f.DBDeps.DBClient.LedgerEntry.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, beforeEntries, afterEntries)
	afterLineage, err := f.lineageService.LoadLineagesByCustomer(ctx, legacylineage.LoadLineagesByCustomerInput{Namespace: f.view.Subscription.Namespace, CustomerID: f.view.Customer.ID, Currency: f.currency.Reference()})
	require.NoError(t, err)
	require.Equal(t, beforeLineage, afterLineage)
	afterBuckets, err := f.ledgerDeps.HistoricalLedger.GetBalanceBuckets(ctx, query)
	require.NoError(t, err)
	require.Equal(t, beforeBuckets, afterBuckets)
	assertNoSubscriptionInvoices(t, f.testDeps, f.view.Customer.ID)
}

// Capture the same snapshot Cancel publishes, then let later workflow writes
// receive a newer root timestamp instead of hiding them behind a frozen clock.
type cancellationSnapshotHook struct {
	subscription.NoOpSubscriptionCommandHook
	event *subscription.CancelledEvent
}

func (h *cancellationSnapshotHook) AfterCancel(ctx context.Context, view subscription.SubscriptionView) error {
	h.event = lo.ToPtr(subscription.NewCancelledEvent(ctx, view))
	clock.FreezeTime(clock.Now().Add(time.Second))
	return nil
}

func TestSubscriptionCustomCurrencyPlanChangeReconcilesCancellation(t *testing.T) {
	// given: a realized fee and future charges on the subscription being replaced.
	clock.FreezeTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()
	f := setupCustomCurrencyFlatFeeSubscription(t)
	ctx := t.Context()
	require.NoError(t, f.subscriptionSyncService.SyncByView(ctx, f.view, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)))
	ids := listSubscriptionChargeIDs(t, f.testDeps, f.view.Subscription.ID)
	require.Len(t, ids, 2)
	clock.FreezeTime(f.view.Subscription.ActiveFrom)
	_, err := f.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: f.view.Customer.GetID()})
	require.NoError(t, err)
	requireCustomCurrencyAccountBalance(t, f, f.business.EarningsAccount, 10)
	hook := &cancellationSnapshotHook{}
	require.NoError(t, f.subscriptionService.RegisterHook(hook))
	month := datetime.MustParseDuration(t, "P1M")
	planInput := pcsubscription.PlanInput{}
	planInput.FromInput(&plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{Namespace: f.view.Subscription.Namespace},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{Name: "Replacement", Currency: f.currency.Reference(), SettlementMode: productcatalog.CreditOnlySettlementMode, BillingCadence: month},
			Phases: []productcatalog.Phase{{PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"}, RateCards: productcatalog.RateCards{&productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{Key: "flat-fee", Name: "Replacement fee", Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: decimal.NewFromInt(20), PaymentTerm: productcatalog.InAdvancePaymentTerm})}, BillingCadence: &month,
			}}}},
		},
	})

	// when: plan replacement annotates the old root after publishing cancellation.
	cancelAt := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(cancelAt)
	changed, err := f.pcSubscriptionService.Change(ctx, pcsubscription.ChangeSubscriptionRequest{
		ID: f.view.Subscription.NamespacedID,
		WorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}, Name: "Replacement", CostBasisMode: subscription.CostBasisModeDynamic,
		},
		PlanInput: planInput, SettlementMode: lo.ToPtr(productcatalog.CreditOnlySettlementMode),
	})
	require.NoError(t, err)
	require.NotNil(t, hook.event)
	require.True(t, changed.Current.UpdatedAt.After(hook.event.Subscription.UpdatedAt))
	require.Equal(t, changed.Next.Subscription.ID, lo.FromPtr(subscription.AnnotationParser.GetSupersedingSubscriptionID(changed.Current.Annotations)))
	require.NoError(t, f.subscriptionSyncService.HandleCancelledEvent(ctx, hook.event))

	// then: annotation-only freshness does not suppress the old fee's correction.
	remaining := listSubscriptionChargeIDs(t, f.testDeps, f.view.Subscription.ID)
	require.Len(t, remaining, 1)
	require.Contains(t, ids, remaining[0])
	result, err := f.chargesService.GetByID(ctx, charges.GetByIDInput{ChargeID: chargesmeta.ChargeID{Namespace: f.view.Subscription.Namespace, ID: remaining[0]}})
	require.NoError(t, err)
	charge, err := result.AsFlatFeeCharge()
	require.NoError(t, err)
	require.Equal(t, cancelAt, charge.Intent.GetEffectiveServicePeriod().To)
	require.Equal(t, float64(5), charge.State.AmountAfterProration.InexactFloat64())
	requireCustomCurrencyAccountBalance(t, f, f.accounts.FBOAccount, 15)
	requireCustomCurrencyAccountBalance(t, f, f.business.EarningsAccount, 5)
	beforeEntries, err := f.DBDeps.DBClient.LedgerEntry.Query().Count(ctx)
	require.NoError(t, err)
	require.NoError(t, f.subscriptionSyncService.HandleCancelledEvent(ctx, hook.event))
	afterEntries, err := f.DBDeps.DBClient.LedgerEntry.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, beforeEntries, afterEntries)
	require.Equal(t, remaining, listSubscriptionChargeIDs(t, f.testDeps, f.view.Subscription.ID))
}

func TestSubscriptionCustomCurrencyStaleCancellation(t *testing.T) {
	for _, tc := range []struct {
		name            string
		cancelMonth     time.Month
		expectedCharges int
	}{
		{name: "continued", expectedCharges: 4},
		{name: "canceled again later", cancelMonth: time.April, expectedCharges: 3},
		{name: "canceled again for the same date", cancelMonth: time.February, expectedCharges: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: a cancellation event queued before continuation and a newer sync.
			clock.FreezeTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()
			f := setupCustomCurrencyFlatFeeSubscription(t)
			ctx := t.Context()
			clock.FreezeTime(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC))
			_, err := f.subscriptionService.Cancel(ctx, f.view.Subscription.NamespacedID, subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)})
			require.NoError(t, err)
			canceled, err := f.subscriptionService.GetView(ctx, f.view.Subscription.NamespacedID)
			require.NoError(t, err)
			event := subscription.NewCancelledEvent(ctx, canceled)
			clock.FreezeTime(time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC))
			_, err = f.subscriptionService.Continue(ctx, f.view.Subscription.NamespacedID)
			require.NoError(t, err)
			if tc.cancelMonth != 0 {
				clock.FreezeTime(time.Date(2026, tc.cancelMonth, 17, 0, 0, 0, 0, time.UTC))
				_, err = f.subscriptionService.Cancel(ctx, f.view.Subscription.NamespacedID, subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)})
				require.NoError(t, err)
			}
			current, err := f.subscriptionService.GetView(ctx, f.view.Subscription.NamespacedID)
			require.NoError(t, err)
			require.True(t, current.Subscription.UpdatedAt.After(event.Subscription.UpdatedAt))
			if tc.cancelMonth == time.February {
				require.Equal(t, event.Spec.ActiveTo, current.Spec.ActiveTo)
			}
			horizon := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
			require.NoError(t, f.subscriptionSyncService.SyncByView(ctx, current, horizon))
			ids := listSubscriptionChargeIDs(t, f.testDeps, f.view.Subscription.ID)
			require.Len(t, ids, tc.expectedCharges)
			states, err := f.subscriptionSyncService.GetSyncStates(ctx, []models.NamespacedID{f.view.Subscription.NamespacedID})
			require.NoError(t, err)
			require.Len(t, states, 1)
			beforeEntries, err := f.DBDeps.DBClient.LedgerEntry.Query().Count(ctx)
			require.NoError(t, err)

			// when: the obsolete cancellation is delivered and retried.
			clock.FreezeTime(clock.Now().Add(time.Minute))
			for range 2 {
				require.NoError(t, f.subscriptionSyncService.HandleCancelledEvent(ctx, &event))
				// then: the newer schedule and its accounting remain intact.
				require.Equal(t, ids, listSubscriptionChargeIDs(t, f.testDeps, f.view.Subscription.ID))
				afterEntries, err := f.DBDeps.DBClient.LedgerEntry.Query().Count(ctx)
				require.NoError(t, err)
				require.Equal(t, beforeEntries, afterEntries)
				afterStates, err := f.subscriptionSyncService.GetSyncStates(ctx, []models.NamespacedID{f.view.Subscription.NamespacedID})
				require.NoError(t, err)
				require.Len(t, afterStates, 1)
				if current.Spec.ActiveTo == nil {
					require.Equal(t, states, afterStates)
				} else {
					require.Equal(t, states[0].HasBillables, afterStates[0].HasBillables)
					require.Equal(t, clock.Now().UTC(), afterStates[0].SyncedAt)
					require.NotNil(t, afterStates[0].NextSyncAfter)
					require.False(t, afterStates[0].NextSyncAfter.Before(*current.Spec.ActiveTo))
				}
			}
		})
	}
}

func TestSubscriptionCustomCurrencyScheduledDeletion(t *testing.T) {
	// given: future custom charges already exist for a scheduled subscription.
	clock.FreezeTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()
	f := setupCustomCurrencyFlatFeeSubscription(t)
	ctx := t.Context()
	horizon := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, f.subscriptionSyncService.SyncByView(ctx, f.view, horizon))
	ids := listSubscriptionChargeIDs(t, f.testDeps, f.view.Subscription.ID)
	require.NotEmpty(t, ids)
	// when: deletion and its event are processed twice, then an older view arrives.
	require.NoError(t, f.subscriptionService.Delete(ctx, f.view.Subscription.NamespacedID))
	event := subscription.NewDeletedEvent(ctx, f.view)
	require.NoError(t, f.subscriptionSyncService.HandleDeletedEvent(ctx, &event))
	beforeEntries, err := f.DBDeps.DBClient.LedgerEntry.Query().Count(ctx)
	require.NoError(t, err)
	require.NoError(t, f.subscriptionSyncService.HandleDeletedEvent(ctx, &event))
	require.NoError(t, f.subscriptionSyncService.SyncByView(ctx, f.view, horizon))
	// then: no future charges are resurrected and no accounting effects are duplicated.
	require.Empty(t, listSubscriptionChargeIDs(t, f.testDeps, f.view.Subscription.ID))
	states, err := f.subscriptionSyncService.GetSyncStates(ctx, []models.NamespacedID{f.view.Subscription.NamespacedID})
	require.NoError(t, err)
	require.Len(t, states, 1)
	require.False(t, states[0].HasBillables)
	for _, id := range ids {
		result, err := f.chargesService.GetByID(ctx, charges.GetByIDInput{ChargeID: chargesmeta.ChargeID{Namespace: f.view.Subscription.Namespace, ID: id}})
		require.NoError(t, err)
		charge, err := result.AsFlatFeeCharge()
		require.NoError(t, err)
		require.Equal(t, flatfee.StatusDeleted, charge.Status)
	}
	afterEntries, err := f.DBDeps.DBClient.LedgerEntry.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, beforeEntries, afterEntries)
	requireCustomCurrencyAccountBalance(t, f, f.accounts.FBOAccount, 20)
	assertNoSubscriptionInvoices(t, f.testDeps, f.view.Customer.ID)
}

func requireCustomCurrencyAccountBalance(t *testing.T, f customCurrencyFlatFeeFixture, account ledger.Account, expected float64) {
	t.Helper()
	balance, err := f.ledgerDeps.HistoricalLedger.GetAccountBalance(t.Context(), account, ledger.RouteFilter{Currency: f.currency.Reference(), CostBasis: mo.Some(lo.ToPtr(decimal.NewFromFloat(0.5))), CostBasisCurrency: mo.Some(lo.ToPtr(currencyx.Code("USD")))}, ledger.BalanceQuery{})
	require.NoError(t, err)
	require.Equal(t, expected, balance.InexactFloat64())
}
