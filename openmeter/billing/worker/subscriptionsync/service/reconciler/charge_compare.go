package reconciler

import (
	"fmt"
	"time"

	chargesflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargesusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func flatFeeChargeMatchesTargetIgnoringPeriodsAndSubscriptionReference(existing persistedstate.FlatFeeChargeGetter, target targetstate.StateItem) (bool, error) {
	targetChargeIntent, err := newFlatFeeChargeIntent(target)
	if err != nil {
		return false, fmt.Errorf("building target flat fee intent: %w", err)
	}

	targetIntent, err := targetChargeIntent.AsFlatFeeIntent()
	if err != nil {
		return false, fmt.Errorf("getting target flat fee intent: %w", err)
	}

	existingIntent := existing.GetFlatFeeCharge().Intent.GetBaseIntent()

	return flatFeeIntentsMatchIgnoringPeriodsAndSubscriptionReference(existingIntent, targetIntent), nil
}

func flatFeeIntentsMatchIgnoringPeriodsAndSubscriptionReference(existing, target chargesflatfee.Intent) bool {
	// Reconciliation always follows the subscription-owned base intent. Manual overrides
	// remain attached to the charge but do not change whether the subscription replaced it.
	existingComparable := flatFeeIntentWithoutPeriodsAndSubscriptionReference(existing.Normalized())
	targetComparable := flatFeeIntentWithoutPeriodsAndSubscriptionReference(target.Normalized())

	if targetComparable.TaxConfig.TaxCodeID == "" {
		// Charge creation resolves an omitted subscription tax code to the namespace default.
		// The resolved ID is persistence state, not a change to the subscription source intent.
		existingComparable.TaxConfig.TaxCodeID = ""
	}

	return existingComparable.Equal(targetComparable)
}

func usageBasedChargeMatchesTargetIgnoringPeriodsAndSubscriptionReference(existing persistedstate.UsageBasedChargeGetter, target targetstate.StateItem) (bool, error) {
	targetChargeIntent, err := newUsageBasedChargeIntent(target)
	if err != nil {
		return false, fmt.Errorf("building target usage based intent: %w", err)
	}

	targetIntent, err := targetChargeIntent.AsUsageBasedIntent()
	if err != nil {
		return false, fmt.Errorf("getting target usage based intent: %w", err)
	}

	existingIntent := existing.GetUsageBasedCharge().Intent.GetBaseIntent()

	return usageBasedIntentsMatchIgnoringPeriodsAndSubscriptionReference(existingIntent, targetIntent), nil
}

func usageBasedIntentsMatchIgnoringPeriodsAndSubscriptionReference(existing, target chargesusagebased.Intent) bool {
	// A customer's override changes the effective rating inputs, but the subscription's
	// base intent still decides whether the physical subscription item was replaced.
	existingComparable := usageBasedIntentWithoutPeriodsAndSubscriptionReference(existing.Normalized())
	targetComparable := usageBasedIntentWithoutPeriodsAndSubscriptionReference(target.Normalized())

	if targetComparable.TaxConfig.TaxCodeID == "" {
		existingComparable.TaxConfig.TaxCodeID = ""
	}

	return existingComparable.Equal(targetComparable)
}

func flatFeeIntentWithoutPeriodsAndSubscriptionReference(intent chargesflatfee.Intent) chargesflatfee.Intent {
	intent.Intent, intent.IntentMutableFields.IntentMutableFields = chargeIntentWithoutPeriodsAndSubscriptionReference(intent.Intent, intent.IntentMutableFields.IntentMutableFields)
	intent.InvoiceAt = time.Time{}
	if !intent.ProRating.Enabled {
		// Persistence supplies the default mode when reading disabled proration,
		// while subscription intent may leave that economically inert mode empty.
		intent.ProRating.Mode = productcatalog.ProRatingModeProratePrices
	}
	if intent.PercentageDiscounts != nil {
		intent.PercentageDiscounts = intent.PercentageDiscounts.CloneOrNil()
		intent.PercentageDiscounts.CorrelationID = ""
	}

	return intent
}

func usageBasedIntentWithoutPeriodsAndSubscriptionReference(intent chargesusagebased.Intent) chargesusagebased.Intent {
	intent.Intent, intent.IntentMutableFields.IntentMutableFields = chargeIntentWithoutPeriodsAndSubscriptionReference(intent.Intent, intent.IntentMutableFields.IntentMutableFields)
	intent.InvoiceAt = time.Time{}
	intent.Discounts = intent.Discounts.Clone()
	if intent.Discounts.Percentage != nil {
		intent.Discounts.Percentage.CorrelationID = ""
	}
	if intent.Discounts.Usage != nil {
		intent.Discounts.Usage.CorrelationID = ""
	}

	return intent
}

func chargeIntentWithoutPeriodsAndSubscriptionReference(intent chargesmeta.Intent, mutable chargesmeta.IntentMutableFields) (chargesmeta.Intent, chargesmeta.IntentMutableFields) {
	intent = intent.Clone()
	delete(intent.Annotations, subscriptionworkflow.AnnotationEditUniqueKey)
	if len(intent.Annotations) == 0 {
		intent.Annotations = nil
	}
	if len(mutable.Metadata) == 0 {
		mutable.Metadata = nil
	}
	intent.Subscription = nil
	// Plan attribution is a best-effort snapshot, not evidence that the underlying item changed.
	intent.SubscriptionPlan = nil

	mutable.ServicePeriod = timeutil.ClosedPeriod{}
	mutable.FullServicePeriod = timeutil.ClosedPeriod{}
	mutable.BillingPeriod = timeutil.ClosedPeriod{}

	return intent, mutable
}
