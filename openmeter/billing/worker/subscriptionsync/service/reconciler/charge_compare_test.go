package reconciler

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargesflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	chargesusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFlatFeeIntentBillingTermsMatch(t *testing.T) {
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
			name: "name change preserves billing terms",
			update: func(existing *chargesflatfee.Intent, _ *chargesflatfee.Intent) {
				existing.Name = "old name"
			},
			expectedMatch: true,
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

			require.Equal(t, tt.expectedMatch, flatFeeIntentBillingTermsMatch(existing, target))
		})
	}
}

func TestBillingComparisonPreservesGeneralEquality(t *testing.T) {
	flatExisting := newFlatFeeComparisonTestIntent(t)
	flatTarget := cloneFlatFeeComparisonTestIntent(flatExisting)
	flatExisting.Name = "previous name"
	require.False(t, flatExisting.Equal(flatTarget))
	require.True(t, flatFeeIntentBillingTermsMatch(flatExisting, flatTarget))

	usageState := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestUsageRateCard())
	usageExisting := newUsageBasedComparisonTestIntentFromTarget(t, usageState)
	usageTarget := cloneUsageBasedComparisonTestIntent(usageExisting)
	usageExisting.Name = "previous name"
	require.False(t, usageExisting.Equal(usageTarget))
	require.True(t, usageBasedIntentBillingTermsMatch(usageExisting, usageTarget))
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

func TestUsageBasedIntentBillingTermsMatch(t *testing.T) {
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

			require.Equal(t, tt.expectedMatch, usageBasedIntentBillingTermsMatch(existing, targetIntent))
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
			existing := newUsageBasedComparisonTestItem(t, target, "usage-based-charge", existingIntent)
			collection := newUsageBasedChargeCollection(1)
			referencePatches := make(ChargeReferencePatches, 1)

			// When subscription sync diffs the current item against that persisted charge.
			err := (&Service{}).diffItem(&target, existing, collection, referencePatches)
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

func newUsageBasedComparisonTestItem(t *testing.T, target targetstate.StateItem, id string, intent chargesusagebased.Intent) persistedstate.Item {
	t.Helper()

	charge := chargesusagebased.Charge{
		ChargeBase: chargesusagebased.ChargeBase{
			ManagedResource: newChargePatchTestManagedResource(target.Subscription.Namespace, id),
			Intent:          intent.AsOverridableIntent(),
			Status:          chargesusagebased.StatusActive,
			State: chargesusagebased.State{
				RatingEngine: chargesusagebased.RatingEngineDelta,
			},
		},
	}

	item, err := persistedstate.NewChargeItemFromChargeType(chargesmeta.ChargeTypeUsageBased, &charge, nil)
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

func assertSharedChargeBillingTerms(t *testing.T, name string, existing, target chargesmeta.Intent, want bool) {
	t.Helper()

	flatExisting := newFlatFeeComparisonTestIntent(t)
	flatTarget := cloneFlatFeeComparisonTestIntent(flatExisting)
	flatExisting.Intent = existing.Clone()
	flatTarget.Intent = target.Clone()
	if got := flatFeeIntentBillingTermsMatch(flatExisting, flatTarget); got != want {
		t.Errorf("%s: flat fee match = %t, want %t", name, got, want)
	}

	usageTarget := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestUsageRateCard())
	usageExisting := newUsageBasedComparisonTestIntentFromTarget(t, usageTarget)
	usageDesired := cloneUsageBasedComparisonTestIntent(usageExisting)
	usageExisting.Intent = existing.Clone()
	usageDesired.Intent = target.Clone()
	if got := usageBasedIntentBillingTermsMatch(usageExisting, usageDesired); got != want {
		t.Errorf("%s: usage match = %t, want %t", name, got, want)
	}
}

func assertFlatFeeBillingTerms(t *testing.T, name string, existing, target chargesflatfee.Intent, want bool) {
	t.Helper()
	if got := flatFeeIntentBillingTermsMatch(existing, target); got != want {
		t.Errorf("%s: match = %t, want %t", name, got, want)
	}
}

func assertUsageBillingTerms(t *testing.T, name string, existing, target chargesusagebased.Intent, want bool) {
	t.Helper()
	if got := usageBasedIntentBillingTermsMatch(existing, target); got != want {
		t.Errorf("%s: match = %t, want %t", name, got, want)
	}
}

func TestSharedChargeBillingTerms(t *testing.T) {
	base := newFlatFeeComparisonTestIntent(t).Intent
	assertSharedChargeBillingTerms(t, "unchanged", base.Clone(), base.Clone(), true)

	existing := base.Clone()
	target := base.Clone()
	existing.Annotations = models.Annotations{
		"dbmigration:backfill_subscription_item_currencies": "historical-marker",
		"subscription.owner": "previous-owner",
		"arbitrary":          "previous-value",
	}
	target.Annotations = models.Annotations{"arbitrary": "current-value"}
	assertSharedChargeBillingTerms(t, "all annotations are excluded", existing, target, true)

	existing = base.Clone()
	target = base.Clone()
	existing.Currency.CostBasis = &[]currencies.CostBasis{}
	assertSharedChargeBillingTerms(t, "expanded currency data is excluded", existing, target, true)

	existing = base.Clone()
	target = base.Clone()
	existing.Currency = testutils.NewFiatCurrency(t, "EUR")
	assertSharedChargeBillingTerms(t, "currency identity is included", existing, target, false)

	existing = base.Clone()
	target = base.Clone()
	existing.TaxConfig.TaxCodeID = ""
	target.TaxConfig.TaxCodeID = ""
	assertSharedChargeBillingTerms(t, "continuously omitted tax code", existing, target, true)

	existing = base.Clone()
	target = base.Clone()
	target.TaxConfig.TaxCodeID = ""
	assertSharedChargeBillingTerms(t, "cleared source tax code accepts stored ID", existing, target, true)

	existing = base.Clone()
	target = base.Clone()
	target.TaxConfig.TaxCodeID = "another-tax-code"
	assertSharedChargeBillingTerms(t, "explicit tax code differs", existing, target, false)

	existing = base.Clone()
	target = base.Clone()
	existing.TaxConfig.TaxCodeID = ""
	target.TaxConfig.TaxCodeID = "explicit-tax-code"
	assertSharedChargeBillingTerms(t, "explicit target tax code rejects blank stored ID", existing, target, false)

	existing = base.Clone()
	target = base.Clone()
	existing.TaxConfig.TaxCodeID = "persisted-default-tax-code"
	target.TaxConfig.TaxCodeID = ""
	existing.TaxConfig.Behavior = ptr(productcatalog.InclusiveTaxBehavior)
	target.TaxConfig.Behavior = ptr(productcatalog.ExclusiveTaxBehavior)
	assertSharedChargeBillingTerms(t, "omitted target tax code still compares behavior", existing, target, false)

	existing = base.Clone()
	target = base.Clone()
	target.TaxConfig.Behavior = ptr(productcatalog.InclusiveTaxBehavior)
	assertSharedChargeBillingTerms(t, "omitted tax behavior is not a wildcard", existing, target, false)

	existing = base.Clone()
	target = base.Clone()
	existing.TaxConfig.Behavior = ptr(productcatalog.InclusiveTaxBehavior)
	target.TaxConfig.Behavior = ptr(productcatalog.ExclusiveTaxBehavior)
	assertSharedChargeBillingTerms(t, "explicit tax behavior differs", existing, target, false)

	existing = base.Clone()
	target = base.Clone()
	existing.CustomerID = "historical-customer"
	existing.ManagedBy = billing.ManuallyManagedLine
	existing.UniqueReferenceID = ptr("historical-logical-reference")
	existing.Subscription.SubscriptionID = "historical-subscription"
	existing.Subscription.PhaseID = "historical-phase"
	existing.Subscription.ItemID = "historical-item"
	existing.SubscriptionPlan = &chargesmeta.SubscriptionPlan{Key: "historical-plan", Version: 1}
	assertSharedChargeBillingTerms(t, "ownership matching and attribution are outside billing terms", existing, target, true)
}

func TestFlatFeeBillingTerms(t *testing.T) {
	base := newFlatFeeComparisonTestIntent(t)
	assertFlatFeeBillingTerms(t, "unchanged", cloneFlatFeeComparisonTestIntent(base), cloneFlatFeeComparisonTestIntent(base), true)

	existing := cloneFlatFeeComparisonTestIntent(base)
	target := cloneFlatFeeComparisonTestIntent(base)
	existing.Name = "previous name"
	existing.Description = ptr("previous description")
	existing.Metadata = models.Metadata{"display": "previous"}
	existing.FeatureID = ptr("previous-feature-id")
	deletedAt := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	existing.IntentDeletedAt = &deletedAt
	existing.InvoiceAt = existing.InvoiceAt.Add(time.Hour)
	existing.ServicePeriod.To = existing.ServicePeriod.To.Add(time.Hour)
	existing.FullServicePeriod.To = existing.FullServicePeriod.To.Add(time.Hour)
	existing.BillingPeriod.To = existing.BillingPeriod.To.Add(time.Hour)
	assertFlatFeeBillingTerms(t, "presentation lineage and timing are excluded", existing, target, true)

	existing = cloneFlatFeeComparisonTestIntent(base)
	target = cloneFlatFeeComparisonTestIntent(base)
	existing.AmountBeforeProration = alpacadecimal.NewFromInt(20)
	assertFlatFeeBillingTerms(t, "contracted amount", existing, target, false)

	existing = cloneFlatFeeComparisonTestIntent(base)
	target = cloneFlatFeeComparisonTestIntent(base)
	existing.AmountBeforeProration = alpacadecimal.RequireFromString("10.004")
	assertFlatFeeBillingTerms(t, "amount uses persisted currency precision", existing, target, true)

	existing = cloneFlatFeeComparisonTestIntent(base)
	target = cloneFlatFeeComparisonTestIntent(base)
	existing.AmountBeforeProration = alpacadecimal.RequireFromString("10.01")
	assertFlatFeeBillingTerms(t, "amount outside currency precision differs", existing, target, false)

	existing = cloneFlatFeeComparisonTestIntent(base)
	target = cloneFlatFeeComparisonTestIntent(base)
	existing.PaymentTerm = ""
	target.PaymentTerm = productcatalog.DefaultPaymentTerm
	assertFlatFeeBillingTerms(t, "empty payment term uses default", existing, target, true)

	existing = cloneFlatFeeComparisonTestIntent(base)
	target = cloneFlatFeeComparisonTestIntent(base)
	existing.PaymentTerm = productcatalog.InAdvancePaymentTerm
	target.PaymentTerm = ""
	assertFlatFeeBillingTerms(t, "omitted target payment term matches stored default", existing, target, true)

	existing = cloneFlatFeeComparisonTestIntent(base)
	target = cloneFlatFeeComparisonTestIntent(base)
	existing.PaymentTerm = productcatalog.InArrearsPaymentTerm
	target.PaymentTerm = ""
	assertFlatFeeBillingTerms(t, "omitted target payment term differs from stored arrears", existing, target, false)

	existing = cloneFlatFeeComparisonTestIntent(base)
	target = cloneFlatFeeComparisonTestIntent(base)
	existing.PaymentTerm = productcatalog.InArrearsPaymentTerm
	assertFlatFeeBillingTerms(t, "payment term changes obligation", existing, target, false)

	existing = cloneFlatFeeComparisonTestIntent(base)
	target = cloneFlatFeeComparisonTestIntent(base)
	existing.FeatureKey = ptr("previous-feature")
	assertFlatFeeBillingTerms(t, "feature key", existing, target, false)

	existing = cloneFlatFeeComparisonTestIntent(base)
	target = cloneFlatFeeComparisonTestIntent(base)
	existing.SettlementMode = productcatalog.CreditThenInvoiceSettlementMode
	assertFlatFeeBillingTerms(t, "settlement mode", existing, target, false)

	existing = cloneFlatFeeComparisonTestIntent(base)
	target = cloneFlatFeeComparisonTestIntent(base)
	existing.PercentageDiscounts = &billing.PercentageDiscount{
		PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(10)},
		CorrelationID:      "previous-correlation",
	}
	target.PercentageDiscounts = existing.PercentageDiscounts.CloneOrNil()
	target.PercentageDiscounts.CorrelationID = "current-correlation"
	assertFlatFeeBillingTerms(t, "discount lineage", existing, target, true)
	target.PercentageDiscounts.Percentage = models.NewPercentage(20)
	assertFlatFeeBillingTerms(t, "discount percentage", existing, target, false)
	assertFlatFeeBillingTerms(t, "discount presence", existing, base, false)

	existing = cloneFlatFeeComparisonTestIntent(base)
	target = cloneFlatFeeComparisonTestIntent(base)
	existing.ProRating = productcatalog.ProRatingConfig{Mode: productcatalog.ProRatingModeProratePrices}
	assertFlatFeeBillingTerms(t, "disabled proration mode is inert", existing, target, true)
	existing.ProRating.Enabled = true
	assertFlatFeeBillingTerms(t, "proration enabled", existing, target, false)
	target.ProRating.Enabled = true
	target.ProRating.Mode = existing.ProRating.Mode
	assertFlatFeeBillingTerms(t, "matching enabled proration mode", existing, target, true)
	target.ProRating.Mode = productcatalog.ProRatingMode("another-mode")
	assertFlatFeeBillingTerms(t, "enabled proration mode", existing, target, false)
}

func TestUsageBillingTerms(t *testing.T) {
	targetState := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestUsageRateCard())
	base := newUsageBasedComparisonTestIntentFromTarget(t, targetState)
	assertUsageBillingTerms(t, "unchanged", cloneUsageBasedComparisonTestIntent(base), cloneUsageBasedComparisonTestIntent(base), true)

	existing := cloneUsageBasedComparisonTestIntent(base)
	target := cloneUsageBasedComparisonTestIntent(base)
	existing.Name = "previous name"
	existing.Description = ptr("previous description")
	existing.Metadata = models.Metadata{"display": "previous"}
	existing.FeatureID = "previous-feature-id"
	deletedAt := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	existing.IntentDeletedAt = &deletedAt
	existing.InvoiceAt = existing.InvoiceAt.Add(time.Hour)
	existing.ServicePeriod.To = existing.ServicePeriod.To.Add(time.Hour)
	existing.FullServicePeriod.To = existing.FullServicePeriod.To.Add(time.Hour)
	existing.BillingPeriod.To = existing.BillingPeriod.To.Add(time.Hour)
	assertUsageBillingTerms(t, "presentation lineage and timing are excluded", existing, target, true)

	existing = cloneUsageBasedComparisonTestIntent(base)
	target = cloneUsageBasedComparisonTestIntent(base)
	existing.SettlementMode = productcatalog.CreditThenInvoiceSettlementMode
	assertUsageBillingTerms(t, "settlement mode", existing, target, false)

	existing = cloneUsageBasedComparisonTestIntent(base)
	target = cloneUsageBasedComparisonTestIntent(base)
	existing.FeatureKey = "previous-feature"
	assertUsageBillingTerms(t, "feature key", existing, target, false)

	existing = cloneUsageBasedComparisonTestIntent(base)
	target = cloneUsageBasedComparisonTestIntent(base)
	existing.Price = *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(2)})
	assertUsageBillingTerms(t, "unit price amount", existing, target, false)

	existing = cloneUsageBasedComparisonTestIntent(base)
	target = cloneUsageBasedComparisonTestIntent(base)
	existing.Price = *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
		Amount: alpacadecimal.NewFromInt(1), QuantityPerPackage: alpacadecimal.NewFromInt(10),
	})
	assertUsageBillingTerms(t, "price type", existing, target, false)

	existing = cloneUsageBasedComparisonTestIntent(base)
	target = cloneUsageBasedComparisonTestIntent(base)
	existing.Discounts.Percentage = &billing.PercentageDiscount{
		PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(10)},
		CorrelationID:      "previous-correlation",
	}
	target.Discounts.Percentage = existing.Discounts.Percentage.CloneOrNil()
	target.Discounts.Percentage.CorrelationID = "current-correlation"
	assertUsageBillingTerms(t, "percentage discount lineage", existing, target, true)
	target.Discounts.Percentage.Percentage = models.NewPercentage(20)
	assertUsageBillingTerms(t, "percentage discount value", existing, target, false)
	assertUsageBillingTerms(t, "percentage discount presence", existing, base, false)

	existing = cloneUsageBasedComparisonTestIntent(base)
	target = cloneUsageBasedComparisonTestIntent(base)
	existing.Discounts.Usage = &billing.UsageDiscount{
		UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(10)},
		CorrelationID: "previous-correlation",
	}
	target.Discounts.Usage = ptr(existing.Discounts.Usage.Clone())
	target.Discounts.Usage.CorrelationID = "current-correlation"
	assertUsageBillingTerms(t, "usage discount lineage", existing, target, true)
	target.Discounts.Usage.Quantity = alpacadecimal.NewFromInt(20)
	assertUsageBillingTerms(t, "usage discount quantity", existing, target, false)
	assertUsageBillingTerms(t, "usage discount presence", existing, base, false)

	existing = cloneUsageBasedComparisonTestIntent(base)
	target = cloneUsageBasedComparisonTestIntent(base)
	existing.UnitConfig = &productcatalog.UnitConfig{
		Operation: productcatalog.UnitConfigOperationMultiply, ConversionFactor: alpacadecimal.NewFromInt(1),
		Precision: 7, DisplayUnit: ptr("previous unit"),
	}
	target.UnitConfig = ptr(existing.UnitConfig.Clone())
	target.UnitConfig.Precision = 0
	target.UnitConfig.Rounding = productcatalog.UnitConfigRoundingModeNone
	target.UnitConfig.DisplayUnit = ptr("current unit")
	assertUsageBillingTerms(t, "rounding none and display unit are inert", existing, target, true)

	target.UnitConfig.Operation = productcatalog.UnitConfigOperationDivide
	assertUsageBillingTerms(t, "unit conversion operation", existing, target, false)
	target.UnitConfig.Operation = existing.UnitConfig.Operation
	target.UnitConfig.ConversionFactor = alpacadecimal.RequireFromString("1.000000001")
	assertUsageBillingTerms(t, "unit conversion factor uses full precision", existing, target, false)
	target.UnitConfig.ConversionFactor = existing.UnitConfig.ConversionFactor
	target.UnitConfig.Rounding = productcatalog.UnitConfigRoundingModeCeiling
	assertUsageBillingTerms(t, "rounding mode enabled", existing, target, false)

	existing.UnitConfig.Rounding = productcatalog.UnitConfigRoundingModeCeiling
	existing.UnitConfig.Precision = 2
	target.UnitConfig.Precision = existing.UnitConfig.Precision
	assertUsageBillingTerms(t, "matching active rounding settings", existing, target, true)
	target.UnitConfig.Precision = 3
	assertUsageBillingTerms(t, "enabled rounding precision", existing, target, false)
	target.UnitConfig.Precision = existing.UnitConfig.Precision
	target.UnitConfig.Rounding = productcatalog.UnitConfigRoundingModeFloor
	assertUsageBillingTerms(t, "enabled rounding mode", existing, target, false)
	assertUsageBillingTerms(t, "unit conversion presence", existing, base, false)
}

func TestUsagePriceBillingTerms(t *testing.T) {
	targetState := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestUsageRateCard())
	base := newUsageBasedComparisonTestIntentFromTarget(t, targetState)

	unit := productcatalog.UnitPrice{
		Commitments: productcatalog.Commitments{
			MinimumAmount: ptr(alpacadecimal.NewFromInt(5)),
			MaximumAmount: ptr(alpacadecimal.NewFromInt(50)),
		},
		Amount: alpacadecimal.NewFromInt(2),
	}
	existing := cloneUsageBasedComparisonTestIntent(base)
	target := cloneUsageBasedComparisonTestIntent(base)
	existing.Price = *productcatalog.NewPriceFrom(unit)
	target.Price = *productcatalog.NewPriceFrom(unit.Clone())
	assertUsageBillingTerms(t, "matching unit commitments", existing, target, true)
	changedUnit := unit.Clone()
	changedUnit.MinimumAmount = ptr(alpacadecimal.NewFromInt(6))
	target.Price = *productcatalog.NewPriceFrom(changedUnit)
	assertUsageBillingTerms(t, "minimum amount", existing, target, false)
	changedUnit = unit.Clone()
	changedUnit.MaximumAmount = ptr(alpacadecimal.NewFromInt(60))
	target.Price = *productcatalog.NewPriceFrom(changedUnit)
	assertUsageBillingTerms(t, "maximum amount", existing, target, false)

	tiered := productcatalog.TieredPrice{
		Mode: productcatalog.VolumeTieredPrice,
		Commitments: productcatalog.Commitments{
			MinimumAmount: ptr(alpacadecimal.NewFromInt(5)),
			MaximumAmount: ptr(alpacadecimal.NewFromInt(50)),
		},
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: ptr(alpacadecimal.NewFromInt(10)),
				FlatPrice:  &productcatalog.PriceTierFlatPrice{Amount: alpacadecimal.NewFromInt(1)},
				UnitPrice:  &productcatalog.PriceTierUnitPrice{Amount: alpacadecimal.NewFromInt(2)},
			},
			{UnitPrice: &productcatalog.PriceTierUnitPrice{Amount: alpacadecimal.NewFromInt(3)}},
		},
	}
	existing = cloneUsageBasedComparisonTestIntent(base)
	target = cloneUsageBasedComparisonTestIntent(base)
	existing.Price = *productcatalog.NewPriceFrom(tiered)
	target.Price = *productcatalog.NewPriceFrom(tiered.Clone())
	assertUsageBillingTerms(t, "matching tiered price", existing, target, true)
	changedTiered := tiered.Clone()
	changedTiered.Mode = productcatalog.GraduatedTieredPrice
	target.Price = *productcatalog.NewPriceFrom(changedTiered)
	assertUsageBillingTerms(t, "tier mode", existing, target, false)
	changedTiered = tiered.Clone()
	changedTiered.Tiers[0].UpToAmount = ptr(alpacadecimal.NewFromInt(11))
	target.Price = *productcatalog.NewPriceFrom(changedTiered)
	assertUsageBillingTerms(t, "tier threshold", existing, target, false)
	changedTiered = tiered.Clone()
	changedTiered.Tiers[0].FlatPrice.Amount = alpacadecimal.NewFromInt(4)
	target.Price = *productcatalog.NewPriceFrom(changedTiered)
	assertUsageBillingTerms(t, "tier flat component", existing, target, false)
	changedTiered = tiered.Clone()
	changedTiered.Tiers[0].UnitPrice.Amount = alpacadecimal.NewFromInt(4)
	target.Price = *productcatalog.NewPriceFrom(changedTiered)
	assertUsageBillingTerms(t, "tier unit component", existing, target, false)
	changedTiered = tiered.Clone()
	changedTiered.Tiers[0], changedTiered.Tiers[1] = changedTiered.Tiers[1], changedTiered.Tiers[0]
	target.Price = *productcatalog.NewPriceFrom(changedTiered)
	assertUsageBillingTerms(t, "tier sequence", existing, target, false)
	changedTiered = tiered.Clone()
	changedTiered.MinimumAmount = ptr(alpacadecimal.NewFromInt(6))
	target.Price = *productcatalog.NewPriceFrom(changedTiered)
	assertUsageBillingTerms(t, "tiered minimum commitment", existing, target, false)
	changedTiered = tiered.Clone()
	changedTiered.Tiers = changedTiered.Tiers[1:]
	target.Price = *productcatalog.NewPriceFrom(changedTiered)
	assertUsageBillingTerms(t, "tier count", existing, target, false)

	packaged := productcatalog.PackagePrice{
		Amount: alpacadecimal.NewFromInt(10), QuantityPerPackage: alpacadecimal.NewFromInt(100),
	}
	existing = cloneUsageBasedComparisonTestIntent(base)
	target = cloneUsageBasedComparisonTestIntent(base)
	existing.Price = *productcatalog.NewPriceFrom(packaged)
	target.Price = *productcatalog.NewPriceFrom(packaged.Clone())
	assertUsageBillingTerms(t, "matching package price", existing, target, true)
	changedPackage := packaged.Clone()
	changedPackage.Amount = alpacadecimal.NewFromInt(11)
	target.Price = *productcatalog.NewPriceFrom(changedPackage)
	assertUsageBillingTerms(t, "package amount", existing, target, false)
	changedPackage = packaged.Clone()
	changedPackage.QuantityPerPackage = alpacadecimal.NewFromInt(101)
	target.Price = *productcatalog.NewPriceFrom(changedPackage)
	assertUsageBillingTerms(t, "package quantity", existing, target, false)

	dynamic := productcatalog.DynamicPrice{Multiplier: alpacadecimal.NewFromInt(2)}
	existing = cloneUsageBasedComparisonTestIntent(base)
	target = cloneUsageBasedComparisonTestIntent(base)
	existing.Price = *productcatalog.NewPriceFrom(dynamic)
	target.Price = *productcatalog.NewPriceFrom(dynamic.Clone())
	assertUsageBillingTerms(t, "matching dynamic price", existing, target, true)
	changedDynamic := dynamic.Clone()
	changedDynamic.Multiplier = alpacadecimal.NewFromInt(3)
	target.Price = *productcatalog.NewPriceFrom(changedDynamic)
	assertUsageBillingTerms(t, "dynamic multiplier", existing, target, false)
}

func assertCostBasisBillingTerms(t *testing.T, name string, existingBasis, targetBasis *costbasis.Intent, want bool) {
	t.Helper()
	currency := testutils.NewCustomCurrency(t, "CREDITS", 4)

	flatExisting := newFlatFeeComparisonTestIntent(t)
	flatTarget := cloneFlatFeeComparisonTestIntent(flatExisting)
	flatExisting.Currency = currency
	flatTarget.Currency = currency.Clone()
	flatExisting.SettlementMode = productcatalog.CreditThenInvoiceSettlementMode
	flatTarget.SettlementMode = productcatalog.CreditThenInvoiceSettlementMode
	if existingBasis != nil {
		flatExisting.CostBasis = ptr(existingBasis.Clone())
	}
	if targetBasis != nil {
		flatTarget.CostBasis = ptr(targetBasis.Clone())
	}
	assertFlatFeeBillingTerms(t, name+" flat fee", flatExisting, flatTarget, want)

	usageTargetState := newChargePatchTestTarget(t, productcatalog.CreditThenInvoiceSettlementMode, newChargePatchTestUsageRateCard())
	usageExisting := newUsageBasedComparisonTestIntentFromTarget(t, usageTargetState)
	usageTarget := cloneUsageBasedComparisonTestIntent(usageExisting)
	usageExisting.Currency = currency.Clone()
	usageTarget.Currency = currency.Clone()
	if existingBasis != nil {
		usageExisting.CostBasis = ptr(existingBasis.Clone())
	}
	if targetBasis != nil {
		usageTarget.CostBasis = ptr(targetBasis.Clone())
	}
	assertUsageBillingTerms(t, name+" usage", usageExisting, usageTarget, want)
}

func TestCostBasisBillingTerms(t *testing.T) {
	usd, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)
	anotherUSD, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)
	eur, err := currencyx.NewFiatCurrency("EUR")
	require.NoError(t, err)

	manual := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: usd, Rate: alpacadecimal.RequireFromString("1.25"),
	})
	sameManual := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: anotherUSD, Rate: alpacadecimal.RequireFromString("1.25"),
	})
	changedRate := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: usd, Rate: alpacadecimal.RequireFromString("1.26"),
	})
	changedFiat := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: eur, Rate: alpacadecimal.RequireFromString("1.25"),
	})
	pinned := costbasis.NewIntent(costbasis.PinnedIntent{
		FiatCurrency: usd, CurrencyCostBasisID: "basis-one",
	})
	changedPinned := costbasis.NewIntent(costbasis.PinnedIntent{
		FiatCurrency: usd, CurrencyCostBasisID: "basis-two",
	})
	dynamic := costbasis.NewIntent(costbasis.DynamicIntent{FiatCurrency: usd})

	assertCostBasisBillingTerms(t, "same fiat identity and rate", &manual, &sameManual, true)
	samePinned := pinned.Clone()
	assertCostBasisBillingTerms(t, "matching pinned basis", &pinned, &samePinned, true)
	sameDynamic := dynamic.Clone()
	assertCostBasisBillingTerms(t, "matching dynamic basis", &dynamic, &sameDynamic, true)
	assertCostBasisBillingTerms(t, "manual rate", &manual, &changedRate, false)
	assertCostBasisBillingTerms(t, "settlement fiat identity", &manual, &changedFiat, false)
	assertCostBasisBillingTerms(t, "pinned basis ID", &pinned, &changedPinned, false)
	assertCostBasisBillingTerms(t, "intent mode", &pinned, &dynamic, false)
	assertCostBasisBillingTerms(t, "intent presence", &manual, nil, false)
}

func TestBillingTermComparisonPhysicalReferenceGate(t *testing.T) {
	t.Run("flat fee with unchanged reference skips billing terms", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		existingIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existingIntent.AmountBeforeProration = alpacadecimal.NewFromInt(20)
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
		patches := newFlatFeeChargeCollection(1)
		references := make(ChargeReferencePatches, 1)

		require.NoError(t, (&Service{}).diffItem(&target, existing, patches, references))
		require.True(t, references.IsEmpty())
		require.True(t, patches.Patches().IsEmpty())
	})

	t.Run("usage with unchanged reference skips billing terms", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestUsageRateCard())
		existingIntent := newUsageBasedComparisonTestIntentFromTarget(t, target)
		existingIntent.Price = *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(2)})
		existing := newUsageBasedComparisonTestItem(t, target, "usage-charge", existingIntent)
		patches := newUsageBasedChargeCollection(1)
		references := make(ChargeReferencePatches, 1)

		require.NoError(t, (&Service{}).diffItem(&target, existing, patches, references))
		require.True(t, references.IsEmpty())
		require.True(t, patches.Patches().IsEmpty())
	})

	t.Run("flat fee presentation and annotation drift repairs reference", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		existingIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existingIntent.Subscription.ItemID = "historical-item"
		existingIntent.Name = "historical display name"
		existingIntent.Annotations = models.Annotations{"migration": "historical-marker"}
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
		patches := newFlatFeeChargeCollection(1)
		references := make(ChargeReferencePatches, 1)

		require.NoError(t, (&Service{}).diffItem(&target, existing, patches, references))
		require.Len(t, references, 1)
		require.True(t, patches.Patches().IsEmpty())
	})

	t.Run("usage presentation and annotation drift repairs reference", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestUsageRateCard())
		existingIntent := newUsageBasedComparisonTestIntentFromTarget(t, target)
		existingIntent.Subscription.ItemID = "historical-item"
		existingIntent.Name = "historical display name"
		existingIntent.Annotations = models.Annotations{"migration": "historical-marker"}
		existing := newUsageBasedComparisonTestItem(t, target, "usage-charge", existingIntent)
		patches := newUsageBasedChargeCollection(1)
		references := make(ChargeReferencePatches, 1)

		require.NoError(t, (&Service{}).diffItem(&target, existing, patches, references))
		require.Len(t, references, 1)
		require.True(t, patches.Patches().IsEmpty())
	})

	t.Run("flat fee target defaults preserve a persisted charge", func(t *testing.T) {
		rateCard := newChargePatchTestFlatRateCard().(*productcatalog.FlatFeeRateCard)
		rateCard.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(10)})
		rateCard.TaxConfig = nil
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, rateCard)
		existingIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existingIntent.PaymentTerm = productcatalog.InAdvancePaymentTerm
		existingIntent.TaxConfig.TaxCodeID = "persisted-default-tax-code"
		existingIntent.Subscription.ItemID = "historical-item"
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
		patches := newFlatFeeChargeCollection(1)
		references := make(ChargeReferencePatches, 1)

		require.NoError(t, (&Service{}).diffItem(&target, existing, patches, references))
		require.Len(t, references, 1)
		require.True(t, patches.Patches().IsEmpty())
	})

	t.Run("usage target with omitted tax code preserves a persisted charge", func(t *testing.T) {
		rateCard := newChargePatchTestUsageRateCard().(*productcatalog.UsageBasedRateCard)
		rateCard.TaxConfig = nil
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, rateCard)
		existingIntent := newUsageBasedComparisonTestIntentFromTarget(t, target)
		existingIntent.TaxConfig.TaxCodeID = "persisted-default-tax-code"
		existingIntent.Subscription.ItemID = "historical-item"
		existing := newUsageBasedComparisonTestItem(t, target, "usage-charge", existingIntent)
		patches := newUsageBasedChargeCollection(1)
		references := make(ChargeReferencePatches, 1)

		require.NoError(t, (&Service{}).diffItem(&target, existing, patches, references))
		require.Len(t, references, 1)
		require.True(t, patches.Patches().IsEmpty())
	})
}

func TestMatchedChargeOwnershipValidation(t *testing.T) {
	t.Run("flat fee customer mismatch is an error", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		existingIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existingIntent.CustomerID = "another-customer"
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
		patches := newFlatFeeChargeCollection(1)
		references := make(ChargeReferencePatches, 1)

		err := (&Service{}).diffItem(&target, existing, patches, references)
		require.ErrorContains(t, err, "customer does not match")
		require.True(t, references.IsEmpty())
		require.True(t, patches.Patches().IsEmpty())
	})

	t.Run("usage managed-by mismatch is an error", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestUsageRateCard())
		existingIntent := newUsageBasedComparisonTestIntentFromTarget(t, target)
		existingIntent.ManagedBy = billing.ManuallyManagedLine
		existing := newUsageBasedComparisonTestItem(t, target, "usage-charge", existingIntent)
		patches := newUsageBasedChargeCollection(1)
		references := make(ChargeReferencePatches, 1)

		err := (&Service{}).diffItem(&target, existing, patches, references)
		require.ErrorContains(t, err, "not subscription-managed")
		require.True(t, references.IsEmpty())
		require.True(t, patches.Patches().IsEmpty())
	})
}
