package usagebased

import (
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

func TestAdvanceChargeInputValidateFeatureMeterHint(t *testing.T) {
	chargeID := meta.ChargeID{Namespace: "namespace", ID: "charge-id"}

	t.Run("when feature meters are omitted", func(t *testing.T) {
		// given:
		// - Advancement does not supply a feature-meter hint.
		// when:
		// - The input is validated.
		// then:
		// - Validation allows the service to resolve the feature meter itself.
		require.NoError(t, (AdvanceChargeInput{ChargeID: chargeID}).Validate())
	})

	t.Run("when an explicit nil feature-meter hint is supplied", func(t *testing.T) {
		// given:
		// - Advancement explicitly supplies a nil authoritative feature-meter hint.
		// when:
		// - The input is validated.
		// then:
		// - Validation rejects the unusable authoritative snapshot.
		err := (AdvanceChargeInput{
			ChargeID:      chargeID,
			FeatureMeters: mo.Some[feature.FeatureMeters](nil),
		}).Validate()
		require.ErrorContains(t, err, "feature meters cannot be nil when provided")
	})
}

func TestValidateExpands(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateExpands(meta.Expands{meta.ExpandRealizations}))
	require.NoError(t, validateExpands(meta.Expands{meta.ExpandRealizations, meta.ExpandDetailedLines}))
	require.NoError(t, validateExpands(meta.Expands{meta.ExpandRealizations, meta.ExpandDeletedRealizations}))
	require.Error(t, validateExpands(meta.Expands{meta.ExpandDetailedLines}))
	require.Error(t, validateExpands(meta.Expands{meta.ExpandDeletedRealizations}))
}

func TestRatingEngineValidate(t *testing.T) {
	t.Parallel()

	require.NoError(t, RatingEngineDelta.Validate())
	require.NoError(t, RatingEnginePeriodPreserving.Validate())
	require.Error(t, RatingEngine("").Validate())
	require.Error(t, RatingEngine("unknown").Validate())
}
