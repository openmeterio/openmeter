package reconciler

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargesusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestFlatFeeChargeCollectionPeriodChangesEmitEmulatedReplacement(t *testing.T) {
	for _, tc := range []struct {
		name string
		add  func(*flatFeeChargeCollection, persistedstate.Item, targetstate.StateItem) error
	}{
		{
			name: "shrink",
			add: func(c *flatFeeChargeCollection, existing persistedstate.Item, target targetstate.StateItem) error {
				return c.AddShrink(target.UniqueID, existing, target)
			},
		},
		{
			name: "extend",
			add: func(c *flatFeeChargeCollection, existing persistedstate.Item, target targetstate.StateItem) error {
				return c.AddExtend(existing, target)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			collection := newFlatFeeChargeCollection(1)
			target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
			existing := newChargePatchTestFlatFeeItem(t, target, "flat-fee-charge")

			require.NoError(t, tc.add(collection, existing, target))

			assertEmulatedReplacement(t, collection.Patches(), "flat-fee-charge", func(intent charges.ChargeIntent) {
				flatFeeIntent, err := intent.AsFlatFeeIntent()
				require.NoError(t, err)
				require.Equal(t, target.GetServicePeriod(), flatFeeIntent.ServicePeriod)
				require.Equal(t, target.FullServicePeriod, flatFeeIntent.FullServicePeriod)
				require.Equal(t, target.BillingPeriod, flatFeeIntent.BillingPeriod)
				require.Equal(t, productcatalog.CreditOnlySettlementMode, flatFeeIntent.SettlementMode)
			})
		})
	}
}

func TestUsageBasedChargeCollectionShrinkEmitsEmulatedReplacement(t *testing.T) {
	collection := newUsageBasedChargeCollection(1)
	target := newChargePatchTestTarget(t, productcatalog.CreditThenInvoiceSettlementMode, newChargePatchTestUsageRateCard())
	existing := newChargePatchTestUsageBasedItem(t, target, "usage-based-charge", productcatalog.CreditThenInvoiceSettlementMode)

	require.NoError(t, collection.AddShrink(target.UniqueID, existing, target))

	assertEmulatedReplacement(t, collection.Patches(), "usage-based-charge", func(intent charges.ChargeIntent) {
		usageBasedIntent, err := intent.AsUsageBasedIntent()
		require.NoError(t, err)
		require.Equal(t, target.GetServicePeriod(), usageBasedIntent.ServicePeriod)
		require.Equal(t, target.FullServicePeriod, usageBasedIntent.FullServicePeriod)
		require.Equal(t, target.BillingPeriod, usageBasedIntent.BillingPeriod)
		require.Equal(t, productcatalog.CreditThenInvoiceSettlementMode, usageBasedIntent.SettlementMode)
	})
}

func TestUsageBasedCreditOnlyChargeCollectionExtendEmitsEmulatedReplacement(t *testing.T) {
	collection := newUsageBasedChargeCollection(1)
	target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestUsageRateCard())
	existing := newChargePatchTestUsageBasedItem(t, target, "usage-based-charge", productcatalog.CreditOnlySettlementMode)

	require.NoError(t, collection.AddExtend(existing, target))

	assertEmulatedReplacement(t, collection.Patches(), "usage-based-charge", func(intent charges.ChargeIntent) {
		usageBasedIntent, err := intent.AsUsageBasedIntent()
		require.NoError(t, err)
		require.Equal(t, target.GetServicePeriod(), usageBasedIntent.ServicePeriod)
		require.Equal(t, target.FullServicePeriod, usageBasedIntent.FullServicePeriod)
		require.Equal(t, target.BillingPeriod, usageBasedIntent.BillingPeriod)
		require.Equal(t, productcatalog.CreditOnlySettlementMode, usageBasedIntent.SettlementMode)
	})
}

func TestUsageBasedCreditThenInvoiceChargeCollectionExtendEmitsNativePatch(t *testing.T) {
	collection := newUsageBasedChargeCollection(1)
	target := newChargePatchTestTarget(t, productcatalog.CreditThenInvoiceSettlementMode, newChargePatchTestUsageRateCard())
	existing := newChargePatchTestUsageBasedItem(t, target, "usage-based-charge", productcatalog.CreditThenInvoiceSettlementMode)

	require.NoError(t, collection.AddExtend(existing, target))

	patches := collection.Patches()
	require.Empty(t, patches.Creates)
	require.Len(t, patches.PatchesByChargeID, 1)

	patch, ok := patches.PatchesByChargeID["usage-based-charge"]
	require.True(t, ok)

	extendPatch, ok := patch.(chargesmeta.PatchExtend)
	require.True(t, ok, "expected native extend patch, got %T", patch)
	require.Equal(t, target.GetServicePeriod().To, extendPatch.GetNewServicePeriodTo())
	require.Equal(t, target.FullServicePeriod.To, extendPatch.GetNewFullServicePeriodTo())
	require.Equal(t, target.BillingPeriod.To, extendPatch.GetNewBillingPeriodTo())
}

func assertEmulatedReplacement(t *testing.T, patches charges.ApplyPatchesInput, chargeID string, assertCreate func(charges.ChargeIntent)) {
	t.Helper()

	require.Len(t, patches.PatchesByChargeID, 1)
	require.Len(t, patches.Creates, 1)

	patch, ok := patches.PatchesByChargeID[chargeID]
	require.True(t, ok)

	_, ok = patch.(chargesmeta.PatchDelete)
	require.True(t, ok, "expected delete patch, got %T", patch)

	assertCreate(patches.Creates[0])
}

func newChargePatchTestFlatFeeItem(t *testing.T, target targetstate.StateItem, id string) persistedstate.Item {
	t.Helper()

	charge := chargesflatfee.Charge{
		ChargeBase: chargesflatfee.ChargeBase{
			ManagedResource: newChargePatchTestManagedResource(target.Subscription.Namespace, id),
			Intent: chargesflatfee.Intent{
				Intent:                newChargePatchTestExistingIntent(target),
				InvoiceAt:             target.GetInvoiceAt(),
				SettlementMode:        target.Subscription.SettlementMode,
				PaymentTerm:           productcatalog.InAdvancePaymentTerm,
				ProRating:             target.Subscription.ProRatingConfig,
				AmountBeforeProration: alpacadecimal.NewFromInt(10),
			},
			Status: chargesflatfee.StatusActive,
			State: chargesflatfee.State{
				AmountAfterProration: alpacadecimal.NewFromInt(10),
			},
		},
	}

	item, err := persistedstate.NewChargeItemFromChargeType(chargesmeta.ChargeTypeFlatFee, nil, &charge)
	require.NoError(t, err)

	return item
}

func newChargePatchTestUsageBasedItem(t *testing.T, target targetstate.StateItem, id string, settlementMode productcatalog.SettlementMode) persistedstate.Item {
	t.Helper()

	charge := chargesusagebased.Charge{
		ChargeBase: chargesusagebased.ChargeBase{
			ManagedResource: newChargePatchTestManagedResource(target.Subscription.Namespace, id),
			Intent: chargesusagebased.Intent{
				Intent:         newChargePatchTestExistingIntent(target),
				InvoiceAt:      target.GetInvoiceAt(),
				SettlementMode: settlementMode,
				FeatureKey:     "feature-key",
				Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
			},
			Status: chargesusagebased.StatusActive,
			State: chargesusagebased.State{
				FeatureID:    "feature-id",
				RatingEngine: chargesusagebased.RatingEngineDelta,
			},
		},
	}

	item, err := persistedstate.NewChargeItemFromChargeType(chargesmeta.ChargeTypeUsageBased, &charge, nil)
	require.NoError(t, err)

	return item
}

func newChargePatchTestManagedResource(namespace, id string) chargesmeta.ManagedResource {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return chargesmeta.ManagedResource{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
		ID: id,
	}
}

func newChargePatchTestExistingIntent(target targetstate.StateItem) chargesmeta.Intent {
	return chargesmeta.Intent{
		Name:       "existing charge",
		ManagedBy:  billing.SubscriptionManagedLine,
		CustomerID: target.Subscription.CustomerId,
		Currency:   target.CurrencyCalculator.Currency,
		ServicePeriod: timeutil.ClosedPeriod{
			From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		FullServicePeriod: target.FullServicePeriod,
		BillingPeriod:     target.BillingPeriod,
		UniqueReferenceID: ptr("existing-charge"),
		Subscription: &chargesmeta.SubscriptionReference{
			SubscriptionID: target.Subscription.ID,
			PhaseID:        target.PhaseID,
			ItemID:         target.SubscriptionItem.ID,
		},
	}
}

func newChargePatchTestTarget(t *testing.T, settlementMode productcatalog.SettlementMode, rateCard productcatalog.RateCard) targetstate.StateItem {
	t.Helper()

	currencyCalculator, err := currencyx.Code("USD").Calculator()
	require.NoError(t, err)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}
	fullServicePeriod := timeutil.ClosedPeriod{
		From: servicePeriod.From,
		To:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
	billingPeriod := timeutil.ClosedPeriod{
		From: servicePeriod.From,
		To:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	itemKey := "item-key"
	phaseKey := "phase-key"

	return targetstate.StateItem{
		SubscriptionItemWithPeriods: targetstate.SubscriptionItemWithPeriods{
			SubscriptionItemView: subscription.SubscriptionItemView{
				SubscriptionItem: subscription.SubscriptionItem{
					NamespacedID: models.NamespacedID{
						Namespace: "namespace",
						ID:        "item-id",
					},
					Key:            itemKey,
					SubscriptionId: "subscription-id",
					PhaseId:        "phase-id",
					RateCard:       rateCard,
				},
				Spec: subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: phaseKey,
							ItemKey:  itemKey,
							RateCard: rateCard,
						},
					},
				},
			},
			UniqueID:          "target-unique-id",
			PhaseID:           "phase-id",
			PhaseKey:          phaseKey,
			ServicePeriod:     servicePeriod,
			FullServicePeriod: fullServicePeriod,
			BillingPeriod:     billingPeriod,
		},
		CurrencyCalculator: currencyCalculator,
		Subscription: subscription.Subscription{
			NamespacedID: models.NamespacedID{
				Namespace: "namespace",
				ID:        "subscription-id",
			},
			CustomerId:     "customer-id",
			Currency:       currencyx.Code("USD"),
			SettlementMode: settlementMode,
		},
	}
}

func newChargePatchTestUsageRateCard() productcatalog.RateCard {
	return &productcatalog.UsageBasedRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:        "usage-rate-card",
			Name:       "Usage Rate Card",
			FeatureKey: ptr("feature-key"),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
		},
		BillingCadence: datetime.NewISODuration(0, 1, 0, 0, 0, 0, 0),
	}
}

func newChargePatchTestFlatRateCard() productcatalog.RateCard {
	billingCadence := datetime.NewISODuration(0, 1, 0, 0, 0, 0, 0)
	return &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:  "flat-rate-card",
			Name: "Flat Rate Card",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
		BillingCadence: &billingCadence,
	}
}

func ptr[T any](v T) *T {
	return &v
}
