package subscriptiontestutils

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Ensures the created view matches the input spec
func ValidateSpecAndView(t *testing.T, expected subscription.SubscriptionSpec, found subscription.SubscriptionView) {
	t.Helper()

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
		assert.Equal(t, expectedStart.UTC(), foundPhases[i].SubscriptionPhase.ActiveFrom.UTC())

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
				rc1JSON, _ := json.Marshal(specItem.RateCard)
				rc2JSON, _ := json.Marshal(foundItem.SubscriptionItem.RateCard)

				assert.True(t, specItem.RateCard.Equal(foundItem.SubscriptionItem.RateCard), "rate card mismatch for item %s in phase %s: \nspec: %s \n\nview: %s", specItem.ItemKey, specPhase.PhaseKey, rc1JSON, rc2JSON)

				// Let's validate the Feature linking
				pFeatureKey := specItem.RateCard.AsMeta().FeatureKey
				if foundItem.SubscriptionItem.RateCard.AsMeta().FeatureKey != nil {
					require.NotNil(t, pFeatureKey)
					assert.Equal(t, pFeatureKey, foundItem.SubscriptionItem.RateCard.AsMeta().FeatureKey)
				} else {
					assert.Empty(t, pFeatureKey)
				}

				rcInp := specItem.CreateSubscriptionItemPlanInput

				// Let's validate the Entitlement
				if rcEnt := rcInp.RateCard.AsMeta().EntitlementTemplate; rcEnt != nil {
					ent := foundItem.Entitlement
					exists := ent != nil
					require.True(t, exists)
					entInp := ent.ToScheduleSubscriptionEntitlementInput()
					assert.Equal(t, rcEnt.Type(), entInp.CreateEntitlementInputs.GetType())

					// Let's validate that subscriptionID annotation is present
					assert.Equal(t, foundItem.Entitlement.Entitlement.Annotations[subscription.AnnotationSubscriptionID], found.Subscription.NamespacedID.ID)

					// Let's validate that the UsagePeriod is aligned
					period := GetEntitlementTemplateUsagePeriod(t, *specItem.RateCard.AsMeta().EntitlementTemplate)
					require.NotNil(t, period)

					// Entitlement UsagePeriod should be aligned to the subscription billing anchor, which means
					truncatedBillingAnchor := found.Subscription.BillingAnchor.Truncate(time.Minute) // Due to minute precision
					// - its duration should be identical
					entPeriod := ent.Entitlement.UsagePeriod.GetOriginalValueAsUsagePeriodInput().GetValue().Interval.Period
					assert.True(t, entPeriod.Equal(period), "usage period interval mismatch, expected %s, got %s", period, entPeriod)
					// - its anchor should be "aligned" with the subscription's billingAnchor
					require.NotNil(t, ent.Entitlement.UsagePeriod)

					// billinganchor would be in the past compared to entitlement start, so usageperiod would normalize to a later iteration
					// to avoid that, lets test with recurrence instead
					recAtAnchor, _, err := ent.Entitlement.UsagePeriod.GetUsagePeriodInputAt(truncatedBillingAnchor)
					require.NoError(t, err)

					entPerAtAnchor, err := recAtAnchor.GetValue().GetPeriodAt(truncatedBillingAnchor)
					require.NoError(t, err)

					require.Equal(t, truncatedBillingAnchor, entPerAtAnchor.From, "entitlement usage period anchor should be aligned with the subscription billing anchor, subscription billing anchor: %s, entitlement usage period: %+v", truncatedBillingAnchor, *ent.Entitlement.UsagePeriod)

					switch rcInp.RateCard.AsMeta().EntitlementTemplate.Type() {
					case entitlement.EntitlementTypeMetered:
						// Validate measureUsageFrom, it should measure usage form the start of the current phase
						require.NotNil(t, ent.Entitlement.MeasureUsageFrom)
						assert.Equal(t, foundPhase.SubscriptionPhase.ActiveFrom.UTC().Truncate(time.Minute), ent.Entitlement.MeasureUsageFrom.UTC().Truncate(time.Minute), "measureUsageFrom should equal the truncated phase start, expected %s, got %s", foundPhase.SubscriptionPhase.ActiveFrom.UTC().Truncate(time.Minute), ent.Entitlement.MeasureUsageFrom.UTC().Truncate(time.Minute))
					}

					// Validate that entitlement activeFrom is the same as the item activeFrom
					require.NotNil(t, ent.Entitlement.ActiveFrom)
					assert.Equal(t, foundItem.SubscriptionItem.ActiveFrom, *ent.Entitlement.ActiveFrom)

					// Validate that the entitlement is only active until the item is scheduled to be
					assert.Equal(t, foundItem.SubscriptionItem.ActiveTo, ent.Entitlement.ActiveTo)
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
		p1Cad, err := s1.GetPhaseCadence(key)
		require.NoError(t, err)

		p2, ok := s2.Phases[key]
		p2Cad, err := s2.GetPhaseCadence(key)
		require.NoError(t, err)

		require.True(t, ok, "phase %s not found in second spec", key)

		// Let's validate the phase properties
		assert.Equal(t, p1.Name, p2.Name, "mismatch for phase %s", key)
		assert.Equal(t, p1.Description, p2.Description, "mismatch for phase %s", key)
		assert.Equal(t, p1.Metadata, p2.Metadata, "mismatch for phase %s", key)
		assert.Equal(t, p1.PhaseKey, p2.PhaseKey, "mismatch for phase %s", key)
		assert.Equal(t, p1.StartAfter, p2.StartAfter, "mismatch for phase %s", key)

		// Let's validate the items
		require.Equal(t, len(p1.ItemsByKey), len(p2.ItemsByKey), "item count mismatch for phase %s, expected %+v and got %+v", key, lo.Keys(p1.ItemsByKey), lo.Keys(p2.ItemsByKey))

		for itemKey := range p1.ItemsByKey {
			p1Items := p1.ItemsByKey[itemKey]
			p2Items, ok := p2.ItemsByKey[itemKey]
			require.True(t, ok, "item %s not found in phase %s", itemKey, key)

			require.Equal(
				t,
				len(p1Items),
				len(p2Items),
				"item count mismatch for item %s in phase %s\n\nexpected: %+v\n\nfound: %+v",
				itemKey,
				key,
				lo.Map(p1Items, func(item *subscription.SubscriptionItemSpec, _ int) models.CadencedModel {
					return item.GetCadence(p1Cad)
				}),
				lo.Map(p2Items, func(item *subscription.SubscriptionItemSpec, _ int) models.CadencedModel {
					return item.GetCadence(p2Cad)
				}),
			)

			for i := range p1Items {
				i1 := p1Items[i]
				i2 := p2Items[i]

				// Let's validate the item properties
				assert.Equal(t, i1.ItemKey, i2.ItemKey)
				assert.True(t, i1.RateCard.Equal(i2.RateCard), "rate card mismatch for item %s in phase %s: \nspec: %+v\n\nview: %+v", itemKey, key, i1.RateCard, i2.RateCard)
				assert.Equal(t, i1.CreateSubscriptionItemPlanInput, i2.CreateSubscriptionItemPlanInput, "create subscription item plan input mismatch for item %s in phase %s", itemKey, key)

				// We'll compare the time offsets separately
				i1af := i1.ActiveFromOverrideRelativeToPhaseStart
				i2af := i2.ActiveFromOverrideRelativeToPhaseStart

				equalNilableTime(
					t,
					tsPlusNillableISO(p1Cad.ActiveFrom, i1af),
					tsPlusNillableISO(p2Cad.ActiveFrom, i2af),
					"active from override relative to phase start mismatch for item %s in phase %s",
					itemKey,
					key,
				)

				i1at := i1.ActiveToOverrideRelativeToPhaseStart
				i2at := i2.ActiveToOverrideRelativeToPhaseStart

				equalNilableTime(
					t,
					tsPlusNillableISO(p1Cad.ActiveFrom, i1at),
					tsPlusNillableISO(p2Cad.ActiveFrom, i2at),
					"active to override relative to phase start mismatch for item %s in phase %s",
					itemKey,
					key,
				)

				// Then we compare the rest without the offsets
				c1 := subscription.CreateSubscriptionItemCustomerInput{
					BillingBehaviorOverride: i1.CreateSubscriptionItemCustomerInput.BillingBehaviorOverride,
				}

				c2 := subscription.CreateSubscriptionItemCustomerInput{
					BillingBehaviorOverride: i2.CreateSubscriptionItemCustomerInput.BillingBehaviorOverride,
				}

				assert.Equal(t, c1, c2, "create subscription item customer input mismatch for item %s in phase %s", itemKey, key)
			}
		}
	}
}

func tsPlusNillableISO(ts time.Time, iso *isodate.Period) *time.Time {
	if iso == nil {
		return nil
	}

	out, _ := iso.AddTo(ts)

	return &out
}

func equalNilableTime(t *testing.T, t1, t2 *time.Time, msgAndArgs ...interface{}) {
	getTpl := func() error {
		if len(msgAndArgs) == 0 {
			return nil
		}

		msg, ok := msgAndArgs[0].(string)
		if !ok {
			return fmt.Errorf("expected string message, got %T", msgAndArgs[0])
		}

		return fmt.Errorf(msg, msgAndArgs[1:]...)
	}

	if t1 == nil != (t2 == nil) {
		t.Fatalf("%s: mismatch for time %v and %v", getTpl(), t1, t2)
	}

	if t1 != nil {
		assert.Equal(t, *t1, *t2, "%s: mismatch for time %v and %v", getTpl(), t1, t2)
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
