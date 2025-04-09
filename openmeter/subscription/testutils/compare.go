package subscriptiontestutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Ensures the created view matches the input spec
func ValidateSpecAndView(t *testing.T, expected subscription.SubscriptionSpec, found subscription.SubscriptionView) {
	// Let's validate the Subscription itself
	assert.Equal(t, expected.Name, found.Subscription.Name)
	assert.Equal(t, expected.Description, found.Subscription.Description)
	assert.Equal(t, expected.Plan, found.Subscription.PlanRef)
	assert.Equal(t, expected.Currency, found.Subscription.Currency)
	assert.Equal(t, expected.CustomerId, found.Subscription.CustomerId)
	assert.Equal(t, expected.ActiveFrom, found.Subscription.ActiveFrom)
	assert.Equal(t, expected.ActiveTo, found.Subscription.ActiveTo)
	assert.Equal(t, expected.Metadata, found.Subscription.Metadata)

	// Let's validate the phases

	foundPhases := found.Phases
	specPhases := expected.GetSortedPhases()

	require.Equal(t, len(specPhases), len(foundPhases), "phase count mismatch")

	for i := range specPhases {
		specPhase := specPhases[i]
		foundPhase := foundPhases[i]

		// Let's validate the phase properties
		assert.Equal(t, specPhase.PhaseKey, foundPhase.SubscriptionPhase.Key)
		assert.Equal(t, specPhase.Name, foundPhase.SubscriptionPhase.Name)
		assert.Equal(t, specPhase.Description, foundPhase.SubscriptionPhase.Description)
		assert.Equal(t, specPhase.Metadata, foundPhase.SubscriptionPhase.Metadata)

		expectedStart, _ := specPhases[i].StartAfter.AddTo(found.Subscription.ActiveFrom)
		assert.Equal(t, expectedStart.UTC(), foundPhases[i].ActiveFrom(found.Subscription.CadencedModel))

		// Test Rate Cards of Phase
		specItemsByKey := specPhase.ItemsByKey
		foundItemsByKey := foundPhase.ItemsByKey

		require.Equal(t, len(specItemsByKey), len(foundItemsByKey), "item count mismatch for phase %s", specPhase.PhaseKey)

		for specItemsKey := range specItemsByKey {
			specItemsByKey, ok := specItemsByKey[specItemsKey]
			require.True(t, ok, "item %s not found in spec phase %s", specItemsKey, specPhase.PhaseKey)
			foundItemsByKey, ok := foundItemsByKey[specItemsKey]
			require.True(t, ok, "item %s not found in found phase %s", specItemsKey, specPhase.PhaseKey)

			require.Equal(t, len(specItemsByKey), len(foundItemsByKey), "item count mismatch for item %s in phase %s", specItemsKey, specPhase.PhaseKey)

			for idx, specItem := range specItemsByKey {
				foundItem := foundItemsByKey[idx]

				// Let's validate the item properties

				assert.Equal(t, specItem.ItemKey, foundItem.SubscriptionItem.Key)
				// Validate phase linking both ways
				assert.Equal(t, foundPhase.SubscriptionPhase.Key, specItem.PhaseKey)
				assert.Equal(t, foundPhase.SubscriptionPhase.ID, foundItem.SubscriptionItem.PhaseId)

				// Let's validate the RateCard is equal
				assert.True(t, specItem.RateCard.Equal(foundItem.SubscriptionItem.RateCard), "rate card mismatch for item %s in phase %s: \nspec: %+v\n\nview: %+v", specItem.ItemKey, specPhase.PhaseKey, specItem.RateCard, foundItem.SubscriptionItem.RateCard)

				// Let's validate the Feature linking
				pFeatureKey := specItem.RateCard.FeatureKey
				if foundItem.SubscriptionItem.RateCard.FeatureKey != nil {
					require.NotNil(t, pFeatureKey)
					assert.Equal(t, pFeatureKey, foundItem.SubscriptionItem.RateCard.FeatureKey)
				} else {
					assert.Empty(t, pFeatureKey)
				}

				rcInp := specItem.CreateSubscriptionItemPlanInput

				// Let's validate the Entitlement
				if rcEnt := rcInp.RateCard.EntitlementTemplate; rcEnt != nil {
					ent := foundItem.Entitlement
					exists := ent != nil
					require.True(t, exists)
					entInp := ent.ToScheduleSubscriptionEntitlementInput()
					assert.Equal(t, rcEnt.Type(), entInp.CreateEntitlementInputs.GetType())

					// Let's validate that subscriptionID annotation is present
					assert.Equal(t, foundItem.Entitlement.Entitlement.Annotations[subscription.AnnotationSubscriptionID], found.Subscription.NamespacedID.ID)

					// Let's validate that the UsagePeriod is aligned
					require.NotNil(t, specItem.RateCard.EntitlementTemplate)
					period := GetEntitlementTemplateUsagePeriod(t, *specItem.RateCard.EntitlementTemplate)
					require.NotNil(t, period)

					// Unfortunately entitlements has minute precision so it can only be aligned to the truncated minute
					rec, err := timeutil.FromISODuration(period, ent.Cadence.ActiveFrom.Truncate(time.Minute))
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
				} else {
					// If an entitlement wasn't defined then there shouldn't be an entitlement
					assert.Nil(t, foundItem.Entitlement)
				}
			}
		}
	}
}

func SpecsEqual(t *testing.T, s1, s2 subscription.SubscriptionSpec) {
	// Let's validate the Subscription itself
	assert.Equal(t, s1.Name, s2.Name)
	assert.Equal(t, s1.Description, s2.Description)
	assert.Equal(t, s1.Plan, s2.Plan)
	assert.Equal(t, s1.Currency, s2.Currency)
	assert.Equal(t, s1.CustomerId, s2.CustomerId)
	assert.Equal(t, s1.ActiveFrom, s2.ActiveFrom)
	assert.Equal(t, s1.ActiveTo, s2.ActiveTo)
	assert.Equal(t, s1.Metadata, s2.Metadata)

	// Let's validate the phases
	require.Equal(t, len(s1.Phases), len(s2.Phases), "phase count mismatch")

	for key := range s1.Phases {
		p1 := s1.Phases[key]
		p2, ok := s2.Phases[key]
		require.True(t, ok, "phase %s not found in second spec", key)

		// Let's validate the phase properties
		assert.Equal(t, p1.Name, p2.Name, "mismatch for phase %s", key)
		assert.Equal(t, p1.Description, p2.Description, "mismatch for phase %s", key)
		assert.Equal(t, p1.Metadata, p2.Metadata, "mismatch for phase %s", key)
		assert.Equal(t, p1.PhaseKey, p2.PhaseKey, "mismatch for phase %s", key)
		assert.Equal(t, p1.StartAfter, p2.StartAfter, "mismatch for phase %s", key)

		// Let's validate the items
		require.Equal(t, len(p1.ItemsByKey), len(p2.ItemsByKey), "item count mismatch for phase %s", key)

		for itemKey := range p1.ItemsByKey {
			p1Items := p1.ItemsByKey[itemKey]
			p2Items, ok := p2.ItemsByKey[itemKey]
			require.True(t, ok, "item %s not found in phase %s", itemKey, key)

			require.Equal(t, len(p1Items), len(p2Items), "item count mismatch for item %s in phase %s", itemKey, key)

			for i := range p1Items {
				i1 := p1Items[i]
				i2 := p2Items[i]

				// Let's validate the item properties
				assert.Equal(t, i1.ItemKey, i2.ItemKey)
				assert.True(t, i1.RateCard.Equal(i2.RateCard), "rate card mismatch for item %s in phase %s: \nspec: %+v\n\nview: %+v", itemKey, key, i1.RateCard, i2.RateCard)
				assert.Equal(t, i1.CreateSubscriptionItemPlanInput, i2.CreateSubscriptionItemPlanInput)
				assert.Equal(t, i1.CreateSubscriptionItemCustomerInput, i2.CreateSubscriptionItemCustomerInput)
			}
		}
	}
}

func SubscriptionAddonsEqual(t *testing.T, a1, a2 subscriptionaddon.SubscriptionAddon) {
	assert.Equal(t, a1.Addon.ID, a2.Addon.ID) // TODO: check all fields?
	assert.Equal(t, a1.SubscriptionID, a2.SubscriptionID)
	assert.Equal(t, a1.Metadata, a2.Metadata)
	assert.Equal(t, a1.RateCards, a2.RateCards)

	require.Equal(t, len(a1.Quantities.GetTimes()), len(a2.Quantities.GetTimes()))
	for i := 0; i < len(a1.Quantities.GetTimes()); i++ {
		require.Equal(t, a1.Quantities.GetAt(i).GetValue().Quantity, a2.Quantities.GetAt(i).GetValue().Quantity)
		require.Equal(t, a1.Quantities.GetAt(i).GetTime(), a2.Quantities.GetAt(i).GetTime())
	}
}
