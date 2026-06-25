package chargeadapter_test

import (
	"testing"

	chargeflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargeusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

func editFlatFeeBaseIntentForTest(t testing.TB, charge *chargeflatfee.Charge, edit func(*chargeflatfee.Intent)) {
	t.Helper()

	intent := charge.Intent.GetBaseIntent()
	edit(&intent)
	charge.Intent = chargeflatfee.NewOverridableIntent(intent, charge.Intent.GetOverrideLayerMutableFields())
}

func editFlatFeeBaseLayerForTest(t testing.TB, charge *chargeflatfee.Charge, edit func(*chargeflatfee.IntentMutableFields)) {
	t.Helper()

	editFlatFeeBaseIntentForTest(t, charge, func(intent *chargeflatfee.Intent) {
		edit(&intent.IntentMutableFields)
	})
}

func editUsageBasedBaseIntentForTest(t testing.TB, charge *chargeusagebased.Charge, edit func(*chargeusagebased.Intent)) {
	t.Helper()

	intent := charge.Intent.GetBaseIntent()
	edit(&intent)
	charge.Intent = chargeusagebased.NewOverridableIntent(intent, charge.Intent.GetOverrideLayerMutableFields())
}

func editUsageBasedBaseLayerForTest(t testing.TB, charge *chargeusagebased.Charge, edit func(*chargeusagebased.IntentMutableFields)) {
	t.Helper()

	editUsageBasedBaseIntentForTest(t, charge, func(intent *chargeusagebased.Intent) {
		edit(&intent.IntentMutableFields)
	})
}

func setUsageBasedSubscriptionForTest(t testing.TB, charge *chargeusagebased.Charge, subscription meta.SubscriptionReference) {
	t.Helper()

	editUsageBasedBaseIntentForTest(t, charge, func(intent *chargeusagebased.Intent) {
		intent.Subscription = &subscription
	})
}
