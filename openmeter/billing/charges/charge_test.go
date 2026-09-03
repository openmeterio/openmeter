package charges

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
)

func TestChargeIntentGetFeatureMeterRef(t *testing.T) {
	// given:
	// - flat-fee and usage-based intents reference the same feature
	// when:
	// - their feature meter references are read
	// then:
	// - each ref carries the meter requirement derived from the charge type
	const featureKey = "api-requests"

	intents := ChargeIntents{
		NewChargeIntent(flatfee.Intent{FeatureKey: lo.ToPtr(featureKey)}),
		NewChargeIntent(usagebased.Intent{FeatureKey: featureKey}),
	}

	flatRef := intents[0].GetFeatureMeterRef()
	usageRef := intents[1].GetFeatureMeterRef()

	require.Equal(t, featureKey, flatRef.IDOrKey.Key)
	require.False(t, flatRef.RequireMeter)
	require.Equal(t, featureKey, usageRef.IDOrKey.Key)
	require.True(t, usageRef.RequireMeter)
	_, flatFeeHasOwner := any(intents[0]).(featuremeter.FeatureReferenceOwner)
	_, usageBasedHasOwner := any(intents[1]).(featuremeter.FeatureReferenceOwner)
	require.False(t, flatFeeHasOwner)
	require.False(t, usageBasedHasOwner)
}

func TestChargeFeatureMeterReferenceForExpansion(t *testing.T) {
	// given:
	// - a persisted usage-based charge whose operational workflows require a meter
	concreteCharge := usagebased.Charge{ChargeBase: usagebased.ChargeBase{
		ManagedResource: meta.ManagedResource{ID: "charge-id"},
		Intent:          usagebased.Intent{FeatureKey: "feature-key"}.AsOverridableIntent(),
		Status:          usagebased.StatusActive,
		State:           usagebased.State{FeatureID: "feature-id"},
	}}
	charge := NewCharge(concreteCharge)

	// when:
	// - the concrete charge and the API-expansion union expose their feature dependency
	concreteRef := concreteCharge.GetFeatureMeterRef()
	expansionRef := charge.GetFeatureMeterRef()

	// then:
	// - the root charge delegates the exact feature reference to the concrete charge
	require.Equal(t, concreteRef, expansionRef)
	require.Equal(t, featuremeter.FeatureReferenceIdentity{
		Kind: featuremeter.FeatureReferenceKindCharges,
		ID:   "charge-id",
	}, charge.GetFeatureMeterOwner())
}
