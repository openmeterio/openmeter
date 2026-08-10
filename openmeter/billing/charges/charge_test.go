package charges

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

func TestChargeIntentsCollectFeatureMeterRefs(t *testing.T) {
	// given:
	// - flat-fee and usage-based intents reference the same feature
	// when:
	// - their feature meter references are collected
	// then:
	// - both refs are retained with the meter requirement derived from the charge type
	const featureKey = "api-requests"

	refs, err := (ChargeIntents{
		NewChargeIntent(flatfee.Intent{FeatureKey: lo.ToPtr(featureKey)}),
		NewChargeIntent(usagebased.Intent{FeatureKey: featureKey}),
	}).CollectFeatureMeterRefs()

	require.NoError(t, err)
	require.Len(t, refs, 2)
	require.Equal(t, featureKey, refs[0].IDOrKey.Key)
	require.False(t, refs[0].RequireMeter)
	require.Equal(t, featureKey, refs[1].IDOrKey.Key)
	require.True(t, refs[1].RequireMeter)
}
