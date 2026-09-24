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
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFlatFeeIntentsMatchIgnoringPeriodsAndSubscriptionReference(t *testing.T) {
	base := newFlatFeeComparisonTestIntent(t)

	tests := []struct {
		name          string
		update        func(existing *chargesflatfee.Intent, target *chargesflatfee.Intent)
		expectedMatch bool
	}{
		{
			name:          "unchanged",
			expectedMatch: true,
		},
		{
			name: "physical subscription reference changed",
			update: func(existing *chargesflatfee.Intent, _ *chargesflatfee.Intent) {
				existing.Subscription.PhaseID = "old-phase-id"
				existing.Subscription.ItemID = "old-item-id"
			},
			expectedMatch: true,
		},
		{
			name: "subscription plan attribution changed",
			update: func(existing *chargesflatfee.Intent, target *chargesflatfee.Intent) {
				existing.SubscriptionPlan = &chargesmeta.SubscriptionPlan{Key: "old", Version: 1}
				target.SubscriptionPlan = &chargesmeta.SubscriptionPlan{Key: "new", Version: 2}
			},
			expectedMatch: true,
		},
		{
			name: "disabled proration mode is defaulted on persistence read",
			update: func(existing *chargesflatfee.Intent, target *chargesflatfee.Intent) {
				existing.ProRating = productcatalog.ProRatingConfig{Enabled: false, Mode: productcatalog.ProRatingModeProratePrices}
				target.ProRating = productcatalog.ProRatingConfig{}
			},
			expectedMatch: true,
		},
		{
			name: "enabling proration is a source intent change",
			update: func(existing *chargesflatfee.Intent, target *chargesflatfee.Intent) {
				existing.ProRating = productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices}
				target.ProRating = productcatalog.ProRatingConfig{}
			},
			expectedMatch: false,
		},
		{
			name: "timestamps changed",
			update: func(existing *chargesflatfee.Intent, _ *chargesflatfee.Intent) {
				existing.ServicePeriod.To = existing.ServicePeriod.To.AddDate(0, 1, 0)
				existing.FullServicePeriod.To = existing.FullServicePeriod.To.AddDate(0, 1, 0)
				existing.BillingPeriod.To = existing.BillingPeriod.To.AddDate(0, 1, 0)
				existing.InvoiceAt = existing.InvoiceAt.AddDate(0, 1, 0)
			},
			expectedMatch: true,
		},
		{
			name: "equal deletion instants in different time zones",
			update: func(existing *chargesflatfee.Intent, target *chargesflatfee.Intent) {
				instant := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
				local := instant.In(time.FixedZone("offset", 3600))
				existing.IntentDeletedAt = &local
				target.IntentDeletedAt = &instant
			},
			expectedMatch: true,
		},
		{
			name: "currency expansion is not source intent",
			update: func(existing *chargesflatfee.Intent, _ *chargesflatfee.Intent) {
				existing.Currency.CostBasis = &[]currencies.CostBasis{}
			},
			expectedMatch: true,
		},
		{
			name: "mutable field change requires replacement",
			update: func(existing *chargesflatfee.Intent, _ *chargesflatfee.Intent) {
				existing.Name = "old name"
			},
			expectedMatch: false,
		},
		{
			name: "price and physical subscription reference changes require replacement",
			update: func(existing *chargesflatfee.Intent, _ *chargesflatfee.Intent) {
				existing.Subscription.ItemID = "old-item-id"
				existing.AmountBeforeProration = alpacadecimal.NewFromInt(20)
			},
			expectedMatch: false,
		},
		{
			name: "source price change still replaces with disabled proration",
			update: func(existing *chargesflatfee.Intent, target *chargesflatfee.Intent) {
				existing.ProRating = productcatalog.ProRatingConfig{Enabled: false, Mode: productcatalog.ProRatingModeProratePrices}
				target.ProRating = productcatalog.ProRatingConfig{}
				existing.AmountBeforeProration = alpacadecimal.NewFromInt(20)
			},
			expectedMatch: false,
		},
		{
			name: "immutable field change requires replacement",
			update: func(existing *chargesflatfee.Intent, _ *chargesflatfee.Intent) {
				existing.TaxConfig.TaxCodeID = "old-tax-code-id"
			},
			expectedMatch: false,
		},
		{
			name: "resolved default tax code is not a source intent change",
			update: func(existing *chargesflatfee.Intent, target *chargesflatfee.Intent) {
				existing.TaxConfig.TaxCodeID = "resolved-default-tax-code"
				target.TaxConfig.TaxCodeID = ""
			},
			expectedMatch: true,
		},
		{
			name: "discount correlation ID is not a source intent change",
			update: func(existing *chargesflatfee.Intent, target *chargesflatfee.Intent) {
				existing.PercentageDiscounts = &billing.PercentageDiscount{
					PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(10)},
					CorrelationID:      "existing-correlation-id",
				}
				target.PercentageDiscounts = &billing.PercentageDiscount{
					PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(10)},
					CorrelationID:      "target-correlation-id",
				}
			},
			expectedMatch: true,
		},
		{
			name: "workflow patch annotation is not a source intent change",
			update: func(existing *chargesflatfee.Intent, _ *chargesflatfee.Intent) {
				if existing.Annotations == nil {
					existing.Annotations = models.Annotations{}
				}
				existing.Annotations[subscriptionworkflow.AnnotationEditUniqueKey] = "patch-id"
			},
			expectedMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := cloneFlatFeeComparisonTestIntent(base)
			target := cloneFlatFeeComparisonTestIntent(base)
			if tt.update != nil {
				tt.update(&existing, &target)
			}

			require.Equal(t, tt.expectedMatch, flatFeeIntentsMatchIgnoringPeriodsAndSubscriptionReference(existing, target))
		})
	}
}

func TestChargeIntentComparisonClearsSubscriptionPlan(t *testing.T) {
	intent := newFlatFeeComparisonTestIntent(t).Intent
	intent.SubscriptionPlan = &chargesmeta.SubscriptionPlan{Key: "plan", Version: 1}

	comparable, _ := chargeIntentWithoutPeriodsAndSubscriptionReference(intent, chargesmeta.IntentMutableFields{})

	require.Nil(t, comparable.SubscriptionPlan)
	require.Equal(t, &chargesmeta.SubscriptionPlan{Key: "plan", Version: 1}, intent.SubscriptionPlan)
}

func TestServiceDiffItemFlatFeeSubscriptionReferenceChange(t *testing.T) {
	t.Run("matching reference leaves reconciliation to the existing flow", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		existingIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
		collection := newFlatFeeChargeCollection(1)
		referencePatches := make(ChargeReferencePatches, 1)

		require.NoError(t, (&Service{}).diffItem(&target, existing, collection, referencePatches))
		require.True(t, referencePatches.IsEmpty())
		require.True(t, collection.Patches().IsEmpty())
	})

	t.Run("reference-only change uses the reference patch batch", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		targetIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existingIntent := cloneFlatFeeComparisonTestIntent(targetIntent)
		existingIntent.Subscription.ItemID = "old-item-id"
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
		collection := newFlatFeeChargeCollection(1)
		referencePatches := make(ChargeReferencePatches, 1)

		require.NoError(t, (&Service{}).diffItem(&target, existing, collection, referencePatches))

		require.True(t, collection.Patches().IsEmpty())
		require.Len(t, referencePatches, 1)
		chargeID := chargesmeta.ChargeID{
			Namespace: target.Subscription.Namespace,
			ID:        "flat-fee-charge",
		}
		patch, ok := referencePatches[chargeID]
		require.True(t, ok)
		updated, err := patch.Apply(*existingIntent.Subscription)
		require.NoError(t, err)
		require.Equal(t, *targetIntent.Subscription, updated)
	})

	t.Run("reference and compatible period changes fall through to the existing flow", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		targetIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existingIntent := cloneFlatFeeComparisonTestIntent(targetIntent)
		existingIntent.Subscription.ItemID = "old-item-id"
		existingIntent.ServicePeriod.To = existingIntent.ServicePeriod.To.AddDate(0, 1, 0)
		existingIntent.FullServicePeriod.To = existingIntent.FullServicePeriod.To.AddDate(0, 1, 0)
		existingIntent.BillingPeriod.To = existingIntent.BillingPeriod.To.AddDate(0, 1, 0)
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
		collection := newFlatFeeChargeCollection(1)
		referencePatches := make(ChargeReferencePatches, 1)

		require.NoError(t, (&Service{}).diffItem(&target, existing, collection, referencePatches))

		require.Len(t, referencePatches, 1)
		chargePatches := collection.Patches()
		require.Len(t, chargePatches.PatchesByChargeID, 1)
		_, ok := chargePatches.PatchesByChargeID["flat-fee-charge"].(chargesmeta.PatchShrink)
		require.True(t, ok)
	})

	t.Run("disabled proration persistence default and derived amount preserve charge identity", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		target.Subscription.ProRatingConfig = productcatalog.ProRatingConfig{}
		targetIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existingIntent := cloneFlatFeeComparisonTestIntent(targetIntent)
		existingIntent.Subscription.ItemID = "old-item-id"
		existingIntent.ProRating.Mode = productcatalog.ProRatingModeProratePrices
		existingIntent.ServicePeriod.To = existingIntent.ServicePeriod.To.AddDate(0, 1, 0)
		existingIntent.FullServicePeriod.To = existingIntent.FullServicePeriod.To.AddDate(0, 1, 0)
		existingIntent.BillingPeriod.To = existingIntent.BillingPeriod.To.AddDate(0, 1, 0)

		existingCharge := chargesflatfee.Charge{
			ChargeBase: chargesflatfee.ChargeBase{
				ManagedResource: newChargePatchTestManagedResource(target.Subscription.Namespace, "flat-fee-charge"),
				Intent:          existingIntent.AsOverridableIntent(),
				Status:          chargesflatfee.StatusActive,
				State: chargesflatfee.State{
					AmountAfterProration: alpacadecimal.NewFromInt(1),
				},
			},
		}
		require.NotEqual(t, targetIntent.AmountBeforeProration, existingCharge.State.AmountAfterProration)
		existing, err := persistedstate.NewChargeItemFromChargeType(chargesmeta.ChargeTypeFlatFee, nil, &existingCharge)
		require.NoError(t, err)
		collection := newFlatFeeChargeCollection(1)
		referencePatches := make(ChargeReferencePatches, 1)

		require.NoError(t, (&Service{}).diffItem(&target, existing, collection, referencePatches))

		require.Len(t, referencePatches, 1)
		chargePatches := collection.Patches()
		require.Len(t, chargePatches.PatchesByChargeID, 1)
		_, ok := chargePatches.PatchesByChargeID["flat-fee-charge"].(chargesmeta.PatchShrink)
		require.True(t, ok)
		require.Empty(t, chargePatches.Creates)
	})

	t.Run("service period start change replaces the charge", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		existingIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existingIntent.ServicePeriod.From = existingIntent.ServicePeriod.From.AddDate(0, -1, 0)
		existingIntent.FullServicePeriod.From = existingIntent.FullServicePeriod.From.AddDate(0, -1, 0)
		existingIntent.BillingPeriod.From = existingIntent.BillingPeriod.From.AddDate(0, -1, 0)
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
		collection := newFlatFeeChargeCollection(1)
		referencePatches := make(ChargeReferencePatches, 1)

		require.NoError(t, (&Service{}).diffItem(&target, existing, collection, referencePatches))

		require.True(t, referencePatches.IsEmpty())
		chargePatches := collection.Patches()
		require.Len(t, chargePatches.PatchesByChargeID, 1)
		_, ok := chargePatches.PatchesByChargeID["flat-fee-charge"].(chargesmeta.PatchDelete)
		require.True(t, ok)
		require.Len(t, chargePatches.Creates, 1)
	})

	t.Run("root subscription change is rejected", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		existingIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existingIntent.Subscription.SubscriptionID = "old-subscription-id"
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
		collection := newFlatFeeChargeCollection(1)
		referencePatches := make(ChargeReferencePatches, 1)

		err := (&Service{}).diffItem(&target, existing, collection, referencePatches)
		require.ErrorContains(t, err, "subscription ID cannot be updated")
		require.True(t, referencePatches.IsEmpty())
		require.True(t, collection.Patches().IsEmpty())
	})

	t.Run("source intent change replaces the charge", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		targetIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existingIntent := cloneFlatFeeComparisonTestIntent(targetIntent)
		existingIntent.Subscription.ItemID = "old-item-id"
		existingIntent.AmountBeforeProration = alpacadecimal.NewFromInt(20)
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
		collection := newFlatFeeChargeCollection(1)
		referencePatches := make(ChargeReferencePatches, 1)

		require.NoError(t, (&Service{}).diffItem(&target, existing, collection, referencePatches))

		require.True(t, referencePatches.IsEmpty())
		chargePatches := collection.Patches()
		require.Len(t, chargePatches.PatchesByChargeID, 1)
		_, ok := chargePatches.PatchesByChargeID["flat-fee-charge"].(chargesmeta.PatchDelete)
		require.True(t, ok)
		require.Len(t, chargePatches.Creates, 1)
	})
}

func TestUsageBasedIntentsMatchIgnoringPeriodsAndSubscriptionReference(t *testing.T) {
	target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestUsageRateCard())
	base := newUsageBasedComparisonTestIntentFromTarget(t, target)

	tests := []struct {
		name          string
		update        func(existing *chargesusagebased.Intent, target *chargesusagebased.Intent)
		expectedMatch bool
	}{
		{name: "unchanged", expectedMatch: true},
		{
			name: "physical subscription reference and periods changed",
			update: func(existing *chargesusagebased.Intent, _ *chargesusagebased.Intent) {
				existing.Subscription.ItemID = "old-item-id"
				existing.ServicePeriod.To = existing.ServicePeriod.To.AddDate(0, 1, 0)
				existing.FullServicePeriod.To = existing.FullServicePeriod.To.AddDate(0, 1, 0)
				existing.BillingPeriod.To = existing.BillingPeriod.To.AddDate(0, 1, 0)
				existing.InvoiceAt = existing.InvoiceAt.AddDate(0, 1, 0)
			},
			expectedMatch: true,
		},
		{
			name: "subscription plan attribution changed",
			update: func(existing *chargesusagebased.Intent, target *chargesusagebased.Intent) {
				existing.SubscriptionPlan = &chargesmeta.SubscriptionPlan{Key: "old", Version: 1}
				target.SubscriptionPlan = &chargesmeta.SubscriptionPlan{Key: "new", Version: 2}
			},
			expectedMatch: true,
		},
		{
			name: "price change",
			update: func(existing *chargesusagebased.Intent, _ *chargesusagebased.Intent) {
				existing.Price = *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(2)})
			},
		},
		{
			name: "feature change",
			update: func(existing *chargesusagebased.Intent, _ *chargesusagebased.Intent) {
				existing.FeatureKey = "old-feature-key"
			},
		},
		{
			name: "unit configuration change",
			update: func(existing *chargesusagebased.Intent, _ *chargesusagebased.Intent) {
				existing.UnitConfig = &productcatalog.UnitConfig{
					Operation:        productcatalog.UnitConfigOperationDivide,
					ConversionFactor: alpacadecimal.NewFromInt(1000),
				}
			},
		},
		{
			name: "resolved default tax code is not source intent",
			update: func(existing *chargesusagebased.Intent, target *chargesusagebased.Intent) {
				existing.TaxConfig.TaxCodeID = "resolved-default-tax-code"
				target.TaxConfig.TaxCodeID = ""
			},
			expectedMatch: true,
		},
		{
			name: "discount correlation IDs are not source intent",
			update: func(existing *chargesusagebased.Intent, target *chargesusagebased.Intent) {
				existing.Discounts.Percentage = &billing.PercentageDiscount{
					PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(10)},
					CorrelationID:      "existing-correlation-id",
				}
				target.Discounts.Percentage = &billing.PercentageDiscount{
					PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(10)},
					CorrelationID:      "target-correlation-id",
				}
			},
			expectedMatch: true,
		},
		{
			name: "usage discount correlation IDs are not source intent",
			update: func(existing *chargesusagebased.Intent, target *chargesusagebased.Intent) {
				existing.Discounts.Usage = &billing.UsageDiscount{
					UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(10)},
					CorrelationID: "existing-correlation-id",
				}
				target.Discounts.Usage = &billing.UsageDiscount{
					UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(10)},
					CorrelationID: "target-correlation-id",
				}
			},
			expectedMatch: true,
		},
		{
			name: "usage discount quantity change",
			update: func(existing *chargesusagebased.Intent, target *chargesusagebased.Intent) {
				existing.Discounts.Usage = &billing.UsageDiscount{
					UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(20)},
				}
				target.Discounts.Usage = &billing.UsageDiscount{
					UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(10)},
				}
			},
		},
		{
			name: "workflow patch annotation is not source intent",
			update: func(existing *chargesusagebased.Intent, _ *chargesusagebased.Intent) {
				if existing.Annotations == nil {
					existing.Annotations = models.Annotations{}
				}
				existing.Annotations[subscriptionworkflow.AnnotationEditUniqueKey] = "patch-id"
			},
			expectedMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := cloneUsageBasedComparisonTestIntent(base)
			targetIntent := cloneUsageBasedComparisonTestIntent(base)
			if tt.update != nil {
				tt.update(&existing, &targetIntent)
			}

			require.Equal(t, tt.expectedMatch, usageBasedIntentsMatchIgnoringPeriodsAndSubscriptionReference(existing, targetIntent))
		})
	}
}

func TestServiceDiffItemUsageBasedSubscriptionReferenceChange(t *testing.T) {
	tests := []struct {
		name            string
		settlementMode  productcatalog.SettlementMode
		update          func(existing *chargesusagebased.Intent)
		expectedPatch   chargesmeta.PatchType
		expectReference bool
		expectCreate    bool
		expectError     string
	}{
		{name: "matching reference", settlementMode: productcatalog.CreditOnlySettlementMode},
		{
			name:           "item reference repair",
			settlementMode: productcatalog.CreditOnlySettlementMode,
			update: func(existing *chargesusagebased.Intent) {
				existing.Subscription.ItemID = "old-item-id"
			},
			expectReference: true,
		},
		{
			name:           "phase reference repair",
			settlementMode: productcatalog.CreditOnlySettlementMode,
			update: func(existing *chargesusagebased.Intent) {
				existing.Subscription.PhaseID = "old-phase-id"
			},
			expectReference: true,
		},
		{
			name:           "reference repair with shrink",
			settlementMode: productcatalog.CreditOnlySettlementMode,
			update: func(existing *chargesusagebased.Intent) {
				existing.Subscription.ItemID = "old-item-id"
				existing.ServicePeriod.To = existing.ServicePeriod.To.AddDate(0, 1, 0)
			},
			expectedPatch:   chargesmeta.PatchTypeShrink,
			expectReference: true,
		},
		{
			name:           "reference repair with extend",
			settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			update: func(existing *chargesusagebased.Intent) {
				existing.Subscription.ItemID = "old-item-id"
				existing.ServicePeriod.To = existing.ServicePeriod.To.AddDate(0, -1, 0)
			},
			expectedPatch:   chargesmeta.PatchTypeExtend,
			expectReference: true,
		},
		{
			name:           "price change replaces physical item",
			settlementMode: productcatalog.CreditOnlySettlementMode,
			update: func(existing *chargesusagebased.Intent) {
				existing.Subscription.ItemID = "old-item-id"
				existing.Price = *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(2)})
			},
			expectedPatch: chargesmeta.PatchTypeDelete,
			expectCreate:  true,
		},
		{
			name:           "start change replaces physical item",
			settlementMode: productcatalog.CreditOnlySettlementMode,
			update: func(existing *chargesusagebased.Intent) {
				existing.ServicePeriod.From = existing.ServicePeriod.From.AddDate(0, -1, 0)
			},
			expectedPatch: chargesmeta.PatchTypeDelete,
			expectCreate:  true,
		},
		{
			name:           "root subscription change is rejected",
			settlementMode: productcatalog.CreditOnlySettlementMode,
			update: func(existing *chargesusagebased.Intent) {
				existing.Subscription.SubscriptionID = "old-subscription-id"
			},
			expectError: "subscription ID cannot be updated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a persisted usage charge for the same logical subscription item.
			target := newChargePatchTestTarget(t, tt.settlementMode, newChargePatchTestUsageRateCard())
			existingIntent := newUsageBasedComparisonTestIntentFromTarget(t, target)
			if tt.update != nil {
				tt.update(&existingIntent)
			}
			charge := chargesusagebased.Charge{
				ChargeBase: chargesusagebased.ChargeBase{
					ManagedResource: newChargePatchTestManagedResource(target.Subscription.Namespace, "usage-based-charge"),
					Intent:          existingIntent.AsOverridableIntent(),
					Status:          chargesusagebased.StatusActive,
					State: chargesusagebased.State{
						RatingEngine: chargesusagebased.RatingEngineDelta,
					},
				},
			}
			existing, err := persistedstate.NewChargeItemFromChargeType(chargesmeta.ChargeTypeUsageBased, &charge, nil)
			require.NoError(t, err)
			collection := newUsageBasedChargeCollection(1)
			referencePatches := make(ChargeReferencePatches, 1)

			// When subscription sync diffs the current item against that persisted charge.
			err = (&Service{}).diffItem(&target, existing, collection, referencePatches)
			if tt.expectError != "" {
				require.ErrorContains(t, err, tt.expectError)
				require.True(t, referencePatches.IsEmpty())
				require.True(t, collection.Patches().IsEmpty())
				return
			}
			require.NoError(t, err)

			// Then the plan repairs the reference, patches the period, or replaces the charge.
			chargeID := chargesmeta.ChargeID{Namespace: target.Subscription.Namespace, ID: "usage-based-charge"}
			if tt.expectReference {
				require.Len(t, referencePatches, 1)
				updated, err := referencePatches[chargeID].Apply(*existingIntent.Subscription)
				require.NoError(t, err)
				require.Equal(t, *target.GetSubscriptionReference(), updated)
			} else {
				require.True(t, referencePatches.IsEmpty())
			}

			chargePatches := collection.Patches()
			if tt.expectedPatch == "" {
				require.Empty(t, chargePatches.PatchesByChargeID)
			} else {
				patch, ok := chargePatches.PatchesByChargeID[chargeID.ID]
				require.True(t, ok)
				require.Equal(t, tt.expectedPatch, patch.Op())
			}
			if tt.expectCreate {
				require.Len(t, chargePatches.Creates, 1)
			} else {
				require.Empty(t, chargePatches.Creates)
			}
		})
	}
}

func newUsageBasedComparisonTestIntentFromTarget(t *testing.T, target targetstate.StateItem) chargesusagebased.Intent {
	t.Helper()

	chargeIntent, err := newUsageBasedChargeIntent(target)
	require.NoError(t, err)

	intent, err := chargeIntent.AsUsageBasedIntent()
	require.NoError(t, err)

	return intent
}

func cloneUsageBasedComparisonTestIntent(intent chargesusagebased.Intent) chargesusagebased.Intent {
	out := intent
	out.Intent = intent.Intent.Clone()
	out.IntentMutableFields = intent.IntentMutableFields.Clone()

	if intent.CostBasis != nil {
		costBasis := intent.CostBasis.Clone()
		out.CostBasis = &costBasis
	}

	return out
}

func newFlatFeeComparisonTestIntent(t *testing.T) chargesflatfee.Intent {
	t.Helper()

	target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
	return newFlatFeeComparisonTestIntentFromTarget(t, target)
}

func newFlatFeeComparisonTestIntentFromTarget(t *testing.T, target targetstate.StateItem) chargesflatfee.Intent {
	t.Helper()

	chargeIntent, err := newFlatFeeChargeIntent(target)
	require.NoError(t, err)

	intent, err := chargeIntent.AsFlatFeeIntent()
	require.NoError(t, err)

	return intent
}

func newFlatFeeComparisonTestItem(t *testing.T, target targetstate.StateItem, id string, intent chargesflatfee.Intent) persistedstate.Item {
	t.Helper()

	charge := chargesflatfee.Charge{
		ChargeBase: chargesflatfee.ChargeBase{
			ManagedResource: newChargePatchTestManagedResource(target.Subscription.Namespace, id),
			Intent:          intent.AsOverridableIntent(),
			Status:          chargesflatfee.StatusActive,
			State: chargesflatfee.State{
				AmountAfterProration: intent.AmountBeforeProration,
			},
		},
	}

	item, err := persistedstate.NewChargeItemFromChargeType(chargesmeta.ChargeTypeFlatFee, nil, &charge)
	require.NoError(t, err)

	return item
}

func cloneFlatFeeComparisonTestIntent(intent chargesflatfee.Intent) chargesflatfee.Intent {
	out := intent
	out.Intent = intent.Intent.Clone()
	out.IntentMutableFields = intent.IntentMutableFields.Clone()

	if intent.FeatureKey != nil {
		featureKey := *intent.FeatureKey
		out.FeatureKey = &featureKey
	}
	if intent.FeatureID != nil {
		featureID := *intent.FeatureID
		out.FeatureID = &featureID
	}
	if intent.CostBasis != nil {
		costBasis := intent.CostBasis.Clone()
		out.CostBasis = &costBasis
	}

	return out
}
