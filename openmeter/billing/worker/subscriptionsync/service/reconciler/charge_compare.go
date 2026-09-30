package reconciler

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargesflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargesusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/equal"
)

func flatFeeBillingTermsMatch(existing persistedstate.FlatFeeChargeGetter, target targetstate.StateItem) (bool, error) {
	targetChargeIntent, err := newFlatFeeChargeIntent(target)
	if err != nil {
		return false, fmt.Errorf("building target flat fee intent: %w", err)
	}

	targetIntent, err := targetChargeIntent.AsFlatFeeIntent()
	if err != nil {
		return false, fmt.Errorf("getting target flat fee intent: %w", err)
	}

	existingIntent := existing.GetFlatFeeCharge().Intent.GetBaseIntent()

	return flatFeeIntentBillingTermsMatch(existingIntent, targetIntent), nil
}

func flatFeeIntentBillingTermsMatch(existing, target chargesflatfee.Intent) bool {
	// Reconciliation always follows the subscription-owned base intent. Manual overrides
	// remain attached to the charge but do not change whether the subscription replaced it.
	existing = existing.Normalized()
	target = target.Normalized()
	if !chargeBillingTermsMatch(existing.Intent, target.Intent) ||
		existing.SettlementMode != target.SettlementMode ||
		!equal.ComparablePtrEqual(existing.FeatureKey, target.FeatureKey) ||
		!equal.PtrEqual(existing.CostBasis, target.CostBasis) ||
		!existing.AmountBeforeProration.Equal(target.AmountBeforeProration) ||
		!percentageDiscountBillingTermsMatch(existing.PercentageDiscounts, target.PercentageDiscounts) ||
		existing.ProRating.Enabled != target.ProRating.Enabled {
		return false
	}

	existingPaymentTerm := existing.PaymentTerm
	if existingPaymentTerm == "" {
		existingPaymentTerm = productcatalog.DefaultPaymentTerm
	}
	targetPaymentTerm := target.PaymentTerm
	if targetPaymentTerm == "" {
		targetPaymentTerm = productcatalog.DefaultPaymentTerm
	}

	if existingPaymentTerm != targetPaymentTerm {
		return false
	}
	if target.ProRating.Enabled {
		if existing.ProRating.Mode != target.ProRating.Mode {
			return false
		}
	}

	return true
}

func usageBasedBillingTermsMatch(existing persistedstate.UsageBasedChargeGetter, target targetstate.StateItem) (bool, error) {
	targetChargeIntent, err := newUsageBasedChargeIntent(target)
	if err != nil {
		return false, fmt.Errorf("building target usage based intent: %w", err)
	}

	targetIntent, err := targetChargeIntent.AsUsageBasedIntent()
	if err != nil {
		return false, fmt.Errorf("getting target usage based intent: %w", err)
	}

	existingIntent := existing.GetUsageBasedCharge().Intent.GetBaseIntent()

	return usageBasedIntentBillingTermsMatch(existingIntent, targetIntent), nil
}

func usageBasedIntentBillingTermsMatch(existing, target chargesusagebased.Intent) bool {
	// A customer's override changes the effective rating inputs, but the subscription's
	// base intent still decides whether the physical subscription item was replaced.
	return chargeBillingTermsMatch(existing.Intent, target.Intent) &&
		existing.SettlementMode == target.SettlementMode &&
		existing.FeatureKey == target.FeatureKey &&
		equal.PtrEqual(existing.CostBasis, target.CostBasis) &&
		(&existing.Price).Equal(&target.Price) &&
		percentageDiscountBillingTermsMatch(existing.Discounts.Percentage, target.Discounts.Percentage) &&
		usageDiscountBillingTermsMatch(existing.Discounts.Usage, target.Discounts.Usage) &&
		unitConfigBillingTermsMatch(existing.UnitConfig, target.UnitConfig)
}

func chargeBillingTermsMatch(existing, target chargesmeta.Intent) bool {
	if !existing.Currency.Reference().Equal(target.Currency.Reference()) {
		return false
	}
	// An omitted target tax-code ID uses the system default for new charges. The
	// existing charge's persisted ID remains compatible with that source intent.
	if target.TaxConfig.TaxCodeID != "" {
		if existing.TaxConfig.TaxCodeID != target.TaxConfig.TaxCodeID {
			return false
		}
	}

	return equal.ComparablePtrEqual(existing.TaxConfig.Behavior, target.TaxConfig.Behavior)
}

func percentageDiscountBillingTermsMatch(existing, target *billing.PercentageDiscount) bool {
	if existing == nil || target == nil {
		return existing == target
	}

	return existing.Percentage.Decimal.Equal(target.Percentage.Decimal)
}

func usageDiscountBillingTermsMatch(existing, target *billing.UsageDiscount) bool {
	if existing == nil || target == nil {
		return existing == target
	}

	return existing.Quantity.Equal(target.Quantity)
}

func unitConfigBillingTermsMatch(existing, target *productcatalog.UnitConfig) bool {
	if existing == nil || target == nil {
		return existing == target
	}

	if existing.Operation != target.Operation ||
		!existing.ConversionFactor.Equal(target.ConversionFactor) ||
		existing.Rounding.IsNone() != target.Rounding.IsNone() {
		return false
	}

	if !existing.Rounding.IsNone() {
		if existing.Rounding != target.Rounding || existing.Precision != target.Precision {
			return false
		}
	}

	return true
}
