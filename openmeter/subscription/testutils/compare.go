package subscriptiontestutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

// Ensures the created view matches the input spec
// TODO: Missing validations (OM-1053)
func ValidateSpecAndView(t *testing.T, expected subscription.SubscriptionSpec, found subscription.SubscriptionView) {
	// Test Phases

	foundPhases := found.Phases
	specPhases := expected.GetSortedPhases()

	require.Equal(t, len(specPhases), len(foundPhases))

	for i := range specPhases {
		assert.Equal(t, specPhases[i].PhaseKey, foundPhases[i].SubscriptionPhase.Key)

		expectedStart, _ := specPhases[i].StartAfter.AddTo(found.Subscription.ActiveFrom)

		assert.Equal(t, expectedStart.UTC(), foundPhases[i].ActiveFrom(found.Subscription.CadencedModel))

		// Test Rate Cards of Phase
		specPhase := specPhases[i]
		foundPhase := foundPhases[i]

		specItemsByKey := specPhase.ItemsByKey
		foundItemsByKey := foundPhase.ItemsByKey

		require.Equal(t, len(specItemsByKey), len(foundItemsByKey), "item count mismatch for phase %s", specPhase.PhaseKey)

		for specItemsKey := range specItemsByKey {
			specItems, ok := specItemsByKey[specItemsKey]
			require.True(t, ok, "item %s not found in spec phase %s", specItemsKey, specPhase.PhaseKey)
			foundItemsByKey, foundItemsForKey := foundItemsByKey[specItemsKey]
			require.True(t, foundItemsForKey, "item %s not found in found phase %s", specItemsKey, specPhase.PhaseKey)

			require.Equal(t, len(specItems), len(foundItemsByKey), "item count mismatch for item %s in phase %s", specItemsKey, specPhase.PhaseKey)

			for idx, specItem := range specItems {
				foundItem := foundItemsByKey[idx]

				assert.Equal(t, specItem.ItemKey, foundItem.SubscriptionItem.Key)

				pFeatureKey := specItem.RateCard.FeatureKey
				if foundItem.SubscriptionItem.RateCard.FeatureKey != nil {
					require.NotNil(t, pFeatureKey)
					assert.Equal(t, pFeatureKey, foundItem.SubscriptionItem.RateCard.FeatureKey)
				} else {
					assert.Empty(t, pFeatureKey)
				}

				rcInp := specItem.CreateSubscriptionItemPlanInput

				if rcEnt := rcInp.RateCard.EntitlementTemplate; rcEnt != nil {
					ent := foundItem.Entitlement
					exists := ent != nil
					require.True(t, exists)
					entInp := ent.ToScheduleSubscriptionEntitlementInput()
					assert.Equal(t, rcEnt.Type(), entInp.CreateEntitlementInputs.GetType())

					// Let's validate that the UsagePeriod is aligned
					require.NotNil(t, specItem.RateCard.EntitlementTemplate)
					period := GetEntitlementTemplateUsagePeriod(t, *specItem.RateCard.EntitlementTemplate)
					require.NotNil(t, period)

					// Unfortunately entitlements has minute precision so it can only be aligned to the truncated minute
					rec, err := recurrence.FromISODuration(period, ent.Cadence.ActiveFrom.Truncate(time.Minute))
					up := entitlement.UsagePeriod(rec)
					assert.NoError(t, err)
					assert.Equal(t, &up, ent.Entitlement.UsagePeriod)

					// Validate that entitlement UsagePeriod matches expected by anchor which is the phase start time
					// Unfortunately entitlement usage period can only be aligned to the minute (due to rounding)
					assert.Equal(t, foundPhase.ActiveFrom(found.Subscription.CadencedModel).Truncate(time.Minute), ent.Entitlement.UsagePeriod.Anchor)

					// Validate that entitlement activeFrom is the same as the phase activeFrom
					require.NotNil(t, ent.Entitlement.ActiveFrom)
					assert.Equal(t, foundPhase.ActiveFrom(found.Subscription.CadencedModel), *ent.Entitlement.ActiveFrom)

					// Validate that the entitlement is only active until the phase is scheduled to be
					if i < len(specPhases)-1 {
						nextPhase := specPhases[i+1]
						nextPhaseStart, _ := nextPhase.StartAfter.AddTo(found.Subscription.ActiveFrom)
						require.NotNil(t, ent.Entitlement.ActiveTo)
						assert.Equal(t, nextPhaseStart.UTC(), *ent.Entitlement.ActiveTo)
					}
				}
			}
		}
	}
}
