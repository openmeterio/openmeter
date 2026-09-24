package subscription_test

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	pcsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestSubscriptionEditProratedCreditHistory(t *testing.T) {
	const namespace = "test-namespace"
	fundedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	startsAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	editAt := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(fundedAt)
	defer clock.UnFreeze()

	deps := setup(t, setupConfig{enableCharges: true})
	defer deps.cleanup(t)
	ctx := t.Context()
	provisionSubscriptionDefaultTaxCodes(t, deps, namespace)
	_, err := deps.billingService.CreateProfile(ctx, minimalCreateProfileInputTemplate(deps.sandboxApp.GetID()))
	require.NoError(t, err)
	customer := createUSDSubscriptionCustomer(t, deps, namespace, "prorated-credit-history")
	currency := currenciestestutils.NewFiatCurrency(t, "USD")
	_, err = deps.ledgerDeps.ResolversService.EnsureBusinessAccounts(ctx, namespace)
	require.NoError(t, err)
	_, err = deps.ledgerDeps.ResolversService.CreateCustomerAccounts(ctx, customer.GetID())
	require.NoError(t, err)

	// given: 100 credits and a custom subscription with one 18-credit monthly fee.
	period := timeutil.ClosedPeriod{From: fundedAt, To: fundedAt}
	funding, err := deps.chargesService.Create(ctx, charges.CreateInput{
		Namespace: namespace,
		Intents: charges.NewCreateChargeIntents(creditpurchase.Intent{
			Intent: chargesmeta.Intent{ManagedBy: billing.ManuallyManagedLine, CustomerID: customer.ID, Currency: currency},
			IntentMutableFields: creditpurchase.IntentMutableFields{
				IntentMutableFields: chargesmeta.IntentMutableFields{
					Name: "Funding", ServicePeriod: period, FullServicePeriod: period, BillingPeriod: period,
				},
				CreditAmount: decimal.NewFromInt(100),
				Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
			},
		}),
	})
	require.NoError(t, err)
	require.Len(t, funding, 1)
	fundingID, err := funding[0].GetChargeID()
	require.NoError(t, err)

	month := datetime.MustParseDuration(t, "P1M")
	rateCard := productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key: "platform-fee", Name: "Platform fee",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount: decimal.NewFromInt(18), PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
		BillingCadence: &month,
	}
	planInput := pcsubscription.PlanInput{}
	planInput.FromInput(&plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name: "Custom subscription", Currency: currency.Reference(),
				BillingCadence: month, SettlementMode: productcatalog.CreditOnlySettlementMode,
				ProRatingConfig: productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices},
			},
			Phases: []productcatalog.Phase{{
				PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
				RateCards: productcatalog.RateCards{&rateCard},
			}},
		},
	})
	created, err := deps.pcSubscriptionService.Create(ctx, pcsubscription.CreateSubscriptionRequest{
		WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
			Namespace: namespace, CustomerID: customer.ID,
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{Custom: &startsAt}, Name: "Custom subscription",
			},
		},
		PlanInput: planInput, SettlementMode: lo.ToPtr(productcatalog.CreditOnlySettlementMode),
	})
	require.NoError(t, err)
	view, err := deps.subscriptionService.GetView(ctx, created.NamespacedID)
	require.NoError(t, err)
	require.Nil(t, view.Subscription.PlanRef)
	require.Len(t, view.Phases, 1)
	require.Len(t, view.Phases[0].ItemsByKey, 1)
	originalItem := view.Phases[0].ItemsByKey[rateCard.Key()][0].SubscriptionItem

	clock.FreezeTime(startsAt)
	require.NoError(t, deps.subscriptionSyncService.SyncByView(ctx, view, startsAt))
	_, err = deps.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: customer.GetID()})
	require.NoError(t, err)

	// then: the initial history includes funding and the full in-advance fee.
	historyInput := customerbalance.ListCreditTransactionsInput{CustomerID: customer.GetID(), Limit: 20}
	before, err := deps.customerBalanceService.ListCreditTransactions(ctx, historyInput)
	require.NoError(t, err)
	require.Len(t, before.Items, 2)
	require.Equal(t, customerbalance.CreditTransactionTypeConsumed, before.Items[0].Type)
	require.Equal(t, float64(-18), before.Items[0].Amount.InexactFloat64())
	require.Equal(t, float64(100), before.Items[0].Balance.Before.InexactFloat64())
	require.Equal(t, float64(82), before.Items[0].Balance.After.InexactFloat64())
	require.Equal(t, originalItem.ID, before.Items[0].Annotations[ledger.AnnotationSubscriptionItemID])
	require.Equal(t, customerbalance.CreditTransactionTypeFunded, before.Items[1].Type)
	require.Equal(t, fundingID.ID, before.Items[1].ID.ID)
	require.Equal(t, float64(100), before.Items[1].Amount.InexactFloat64())
	require.Equal(t, float64(0), before.Items[1].Balance.Before.InexactFloat64())
	require.Equal(t, float64(100), before.Items[1].Balance.After.InexactFloat64())

	// when: a running edit replaces the fee halfway through February, prorating
	// both the old 18-credit fee and the new 36-credit fee through subscription sync.
	clock.FreezeTime(editAt)
	replacement := rateCard
	replacement.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount: decimal.NewFromInt(36), PaymentTerm: productcatalog.InAdvancePaymentTerm,
	})
	edited, err := deps.subscriptionWorkflowService.EditRunning(ctx, created.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{PhaseKey: "default", ItemKey: rateCard.Key()},
		patch.PatchAddItem{
			PhaseKey: "default", ItemKey: rateCard.Key(),
			CreateInput: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
						PhaseKey: "default", ItemKey: rateCard.Key(), RateCard: &replacement,
					},
				},
			},
		},
	}, subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)})
	require.NoError(t, err)
	versions := edited.Phases[0].ItemsByKey[rateCard.Key()]
	require.Len(t, versions, 2)
	require.NotEqual(t, originalItem.ID, versions[0].SubscriptionItem.ID, "the edit must exercise recreated subscription items")
	require.NoError(t, deps.subscriptionSyncService.SyncByView(ctx, edited, editAt))
	_, err = deps.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: customer.GetID()})
	require.NoError(t, err)

	// then: history retains the original rows and adds the refund and replacement
	// usage, so every visible movement adds up to the customer's settled balance.
	after, err := deps.customerBalanceService.ListCreditTransactions(ctx, historyInput)
	require.NoError(t, err)
	require.Len(t, after.Items, 4)
	require.Equal(t, before.Items, after.Items[2:])
	require.Equal(t, customerbalance.CreditTransactionTypeConsumed, after.Items[0].Type)
	require.Equal(t, float64(-18), after.Items[0].Amount.InexactFloat64())
	require.Equal(t, versions[1].SubscriptionItem.ID, after.Items[0].Annotations[ledger.AnnotationSubscriptionItemID])
	correction := after.Items[1]
	require.Equal(t, customerbalance.CreditTransactionTypeCorrection, correction.Type)
	require.Equal(t, float64(9), correction.Amount.InexactFloat64())
	require.Equal(t, startsAt, correction.BookedAt.UTC())
	require.Equal(t, editAt, correction.CreatedAt.UTC())
	require.Equal(t, "Platform fee", correction.Name)
	require.Equal(t, before.Items[0].Annotations[ledger.AnnotationChargeID], correction.Annotations[ledger.AnnotationChargeID])
	require.Equal(t, created.ID, correction.Annotations[ledger.AnnotationSubscriptionID])
	require.NotEmpty(t, correction.Annotations[ledger.AnnotationSubscriptionItemID])
	require.Equal(t, float64(82), correction.Balance.Before.InexactFloat64())
	require.Equal(t, float64(91), correction.Balance.After.InexactFloat64())
	require.Equal(t, float64(91), after.Items[0].Balance.Before.InexactFloat64())
	require.Equal(t, float64(73), after.Items[0].Balance.After.InexactFloat64())

	total := decimal.Zero
	for _, item := range after.Items {
		total = total.Add(item.Amount)
	}
	settled, err := deps.customerBalanceService.GetSettledBalance(ctx, customerbalance.GetBalanceServiceInput{
		CustomerID: customer.GetID(), Currency: currency.Reference(),
	})
	require.NoError(t, err)
	require.Equal(t, float64(73), settled.InexactFloat64())
	require.Equal(t, settled.InexactFloat64(), total.InexactFloat64())
}
