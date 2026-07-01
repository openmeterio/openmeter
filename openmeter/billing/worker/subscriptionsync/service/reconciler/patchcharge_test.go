package reconciler

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
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

func TestFlatFeeCreditOnlyChargeCollectionShrinkEmitsNativePatch(t *testing.T) {
	collection := newFlatFeeChargeCollection(1)
	target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
	existingServicePeriod := timeutil.ClosedPeriod{
		From: target.GetServicePeriod().From,
		To:   target.GetServicePeriod().To.AddDate(0, 1, 0),
	}
	existingIntent := newChargePatchTestExistingFlatFeeIntent(target)
	existingIntent.ServicePeriod = existingServicePeriod
	existing := newChargePatchTestFlatFeeItemWithIntent(t, target, "flat-fee-charge", existingIntent)

	require.NoError(t, collection.AddShrink(target.UniqueID, existing, target))

	patches := collection.Patches()
	require.Empty(t, patches.Creates)
	require.Len(t, patches.PatchesByChargeID, 1)

	patch, ok := patches.PatchesByChargeID["flat-fee-charge"]
	require.True(t, ok)

	shrinkPatch, ok := patch.(chargesmeta.PatchShrink)
	require.True(t, ok, "expected native shrink patch, got %T", patch)
	require.Equal(t, target.GetServicePeriod().To, shrinkPatch.GetNewServicePeriodTo())
	require.Equal(t, target.FullServicePeriod.To, shrinkPatch.GetNewFullServicePeriodTo())
	require.Equal(t, target.BillingPeriod.To, shrinkPatch.GetNewBillingPeriodTo())
	require.Equal(t, target.GetInvoiceAt(), shrinkPatch.GetNewInvoiceAt())
}

func TestFlatFeeCreditOnlyChargeCollectionExtendEmitsNativePatch(t *testing.T) {
	collection := newFlatFeeChargeCollection(1)
	target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
	existing := newChargePatchTestFlatFeeItem(t, target, "flat-fee-charge")

	require.NoError(t, collection.AddExtend(existing, target))

	patches := collection.Patches()
	require.Empty(t, patches.Creates)
	require.Len(t, patches.PatchesByChargeID, 1)

	patch, ok := patches.PatchesByChargeID["flat-fee-charge"]
	require.True(t, ok)

	extendPatch, ok := patch.(chargesmeta.PatchExtend)
	require.True(t, ok, "expected native extend patch, got %T", patch)
	require.Equal(t, target.GetServicePeriod().To, extendPatch.GetNewServicePeriodTo())
	require.Equal(t, target.FullServicePeriod.To, extendPatch.GetNewFullServicePeriodTo())
	require.Equal(t, target.BillingPeriod.To, extendPatch.GetNewBillingPeriodTo())
	require.Equal(t, target.GetInvoiceAt(), extendPatch.GetNewInvoiceAt())
}

func TestUsageBasedCreditOnlyChargeCollectionShrinkEmitsNativePatch(t *testing.T) {
	collection := newUsageBasedChargeCollection(1)
	target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestUsageRateCard())
	existingServicePeriod := timeutil.ClosedPeriod{
		From: target.GetServicePeriod().From,
		To:   target.GetServicePeriod().To.AddDate(0, 1, 0),
	}
	existing := newChargePatchTestUsageBasedItemWithServicePeriod(t, target, "usage-based-charge", productcatalog.CreditOnlySettlementMode, existingServicePeriod)

	require.NoError(t, collection.AddShrink(target.UniqueID, existing, target))

	patches := collection.Patches()
	require.Empty(t, patches.Creates)
	require.Len(t, patches.PatchesByChargeID, 1)

	patch, ok := patches.PatchesByChargeID["usage-based-charge"]
	require.True(t, ok)

	shrinkPatch, ok := patch.(chargesmeta.PatchShrink)
	require.True(t, ok, "expected native shrink patch, got %T", patch)
	require.Equal(t, target.GetServicePeriod().To, shrinkPatch.GetNewServicePeriodTo())
	require.Equal(t, target.FullServicePeriod.To, shrinkPatch.GetNewFullServicePeriodTo())
	require.Equal(t, target.BillingPeriod.To, shrinkPatch.GetNewBillingPeriodTo())
	require.Equal(t, target.GetInvoiceAt(), shrinkPatch.GetNewInvoiceAt())
}

func TestUsageBasedCreditThenInvoiceChargeCollectionShrinkEmitsNativePatch(t *testing.T) {
	collection := newUsageBasedChargeCollection(1)
	target := newChargePatchTestTarget(t, productcatalog.CreditThenInvoiceSettlementMode, newChargePatchTestUsageRateCard())
	existingServicePeriod := timeutil.ClosedPeriod{
		From: target.GetServicePeriod().From,
		To:   target.GetServicePeriod().To.AddDate(0, 1, 0),
	}
	existing := newChargePatchTestUsageBasedItemWithServicePeriod(t, target, "usage-based-charge", productcatalog.CreditThenInvoiceSettlementMode, existingServicePeriod)

	require.NoError(t, collection.AddShrink(target.UniqueID, existing, target))

	patches := collection.Patches()
	require.Empty(t, patches.Creates)
	require.Len(t, patches.PatchesByChargeID, 1)

	patch, ok := patches.PatchesByChargeID["usage-based-charge"]
	require.True(t, ok)

	shrinkPatch, ok := patch.(chargesmeta.PatchShrink)
	require.True(t, ok, "expected native shrink patch, got %T", patch)
	require.Equal(t, target.GetServicePeriod().To, shrinkPatch.GetNewServicePeriodTo())
	require.Equal(t, target.FullServicePeriod.To, shrinkPatch.GetNewFullServicePeriodTo())
	require.Equal(t, target.BillingPeriod.To, shrinkPatch.GetNewBillingPeriodTo())
	require.Equal(t, target.GetInvoiceAt(), shrinkPatch.GetNewInvoiceAt())
}

func TestUsageBasedCreditOnlyChargeCollectionExtendEmitsNativePatch(t *testing.T) {
	collection := newUsageBasedChargeCollection(1)
	target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestUsageRateCard())
	existing := newChargePatchTestUsageBasedItem(t, target, "usage-based-charge", productcatalog.CreditOnlySettlementMode)

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
	require.Equal(t, target.GetInvoiceAt(), extendPatch.GetNewInvoiceAt())
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
	require.Equal(t, target.GetInvoiceAt(), extendPatch.GetNewInvoiceAt())
}

func newChargePatchTestFlatFeeItem(t *testing.T, target targetstate.StateItem, id string) persistedstate.Item {
	t.Helper()

	existingIntent := newChargePatchTestExistingFlatFeeIntent(target)
	return newChargePatchTestFlatFeeItemWithIntent(t, target, id, existingIntent)
}

func newChargePatchTestFlatFeeItemWithIntent(t *testing.T, target targetstate.StateItem, id string, existingIntent chargesflatfee.Intent) persistedstate.Item {
	t.Helper()

	charge := chargesflatfee.Charge{
		ChargeBase: chargesflatfee.ChargeBase{
			ManagedResource: newChargePatchTestManagedResource(target.Subscription.Namespace, id),
			Intent: chargesflatfee.Intent{
				Intent: existingIntent.Intent,
				IntentMutableFields: chargesflatfee.IntentMutableFields{
					IntentMutableFields:   existingIntent.IntentMutableFields.IntentMutableFields,
					InvoiceAt:             target.GetInvoiceAt(),
					PaymentTerm:           productcatalog.InAdvancePaymentTerm,
					ProRating:             target.Subscription.ProRatingConfig,
					AmountBeforeProration: alpacadecimal.NewFromInt(10),
				},
				SettlementMode: target.Subscription.SettlementMode,
			}.AsOverridableIntent(),
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

func newChargePatchTestExistingFlatFeeIntent(target targetstate.StateItem) chargesflatfee.Intent {
	existingIntent := newChargePatchTestExistingIntent(target)

	return chargesflatfee.Intent{
		Intent: existingIntent.Intent,
		IntentMutableFields: chargesflatfee.IntentMutableFields{
			IntentMutableFields: existingIntent.IntentMutableFields.IntentMutableFields,
			InvoiceAt:           target.GetInvoiceAt(),
			PaymentTerm:         productcatalog.InAdvancePaymentTerm,
			ProRating:           target.Subscription.ProRatingConfig,
		},
		SettlementMode: target.Subscription.SettlementMode,
	}
}

func newChargePatchTestUsageBasedItem(t *testing.T, target targetstate.StateItem, id string, settlementMode productcatalog.SettlementMode) persistedstate.Item {
	t.Helper()

	return newChargePatchTestUsageBasedItemWithFullServicePeriod(t, target, id, settlementMode, target.FullServicePeriod)
}

func newChargePatchTestUsageBasedItemWithFullServicePeriod(t *testing.T, target targetstate.StateItem, id string, settlementMode productcatalog.SettlementMode, fullServicePeriod timeutil.ClosedPeriod) persistedstate.Item {
	t.Helper()

	intent := newChargePatchTestExistingIntent(target)
	intent.FullServicePeriod = fullServicePeriod

	return newChargePatchTestUsageBasedItemWithIntent(t, target, id, settlementMode, intent)
}

func newChargePatchTestUsageBasedItemWithServicePeriod(t *testing.T, target targetstate.StateItem, id string, settlementMode productcatalog.SettlementMode, servicePeriod timeutil.ClosedPeriod) persistedstate.Item {
	t.Helper()

	intent := newChargePatchTestExistingIntent(target)
	intent.ServicePeriod = servicePeriod

	return newChargePatchTestUsageBasedItemWithIntent(t, target, id, settlementMode, intent)
}

func newChargePatchTestUsageBasedItemWithIntent(t *testing.T, target targetstate.StateItem, id string, settlementMode productcatalog.SettlementMode, intent chargesusagebased.Intent) persistedstate.Item {
	t.Helper()

	charge := chargesusagebased.Charge{
		ChargeBase: chargesusagebased.ChargeBase{
			ManagedResource: newChargePatchTestManagedResource(target.Subscription.Namespace, id),
			Intent: chargesusagebased.Intent{
				Intent: intent.Intent,
				IntentMutableFields: chargesusagebased.IntentMutableFields{
					IntentMutableFields: intent.IntentMutableFields.IntentMutableFields,
					InvoiceAt:           target.GetInvoiceAt(),
					Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
				},
				SettlementMode: settlementMode,
				FeatureKey:     "feature-key",
			}.AsOverridableIntent(),
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

func newChargePatchTestExistingIntent(target targetstate.StateItem) chargesusagebased.Intent {
	return chargesusagebased.Intent{
		Intent: chargesmeta.Intent{
			ManagedBy:         billing.SubscriptionManagedLine,
			CustomerID:        target.Subscription.CustomerId,
			Currency:          target.CurrencyCalculator.CurrencyCode(),
			UniqueReferenceID: ptr("existing-charge"),
			TaxConfig: productcatalog.TaxCodeConfig{
				TaxCodeID: "tax-code-id",
			},
			Subscription: &chargesmeta.SubscriptionReference{
				SubscriptionID: target.Subscription.ID,
				PhaseID:        target.PhaseID,
				ItemID:         target.SubscriptionItem.ID,
			},
		},
		IntentMutableFields: chargesusagebased.IntentMutableFields{
			IntentMutableFields: chargesmeta.IntentMutableFields{
				Name: "existing charge",
				ServicePeriod: timeutil.ClosedPeriod{
					From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				},
				FullServicePeriod: target.FullServicePeriod,
				BillingPeriod:     target.BillingPeriod,
			},
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
			TaxConfig: &productcatalog.TaxConfig{
				TaxCodeID: ptr("tax-code-id"),
			},
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
			TaxConfig: &productcatalog.TaxConfig{
				TaxCodeID: ptr("tax-code-id"),
			},
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
