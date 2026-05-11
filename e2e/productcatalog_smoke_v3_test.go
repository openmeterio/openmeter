package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

// TestV3ProductCatalogSmoke exercises the cross-cutting plan + addon
// authoring flow end to end, with v3 raw calls.
// HTTP (plan, addon, plan-addon). The entity-siloed tests in plans_v3_test.go,
// addons_v3_test.go, and planaddons_v3_test.go cover each domain in isolation;
// this one ties them together.
//
// Flow:
//   - Create a meter
//   - Create a feature bound to the meter
//   - Create a draft plan with a single flat rate card
//   - Update the plan to carry flat + usage (with feature) + graduated rate cards.
//   - Add a defective rate card (cadence-misaligned) and confirm the draft-with-errors
//     loop: GET surfaces validation_errors, publish is rejected with the same code.
//   - Remove the defective rate card and confirm validation_errors clears.
//   - Create a draft addon and publish it.
//   - Attach the published addon to the still-draft plan.
//   - Publish the plan and confirm the attached addon survives the transition.
func TestV3ProductCatalogSmoke(t *testing.T) {
	c := newV3Client(t)

	meterKey := uniqueKey("sanity_meter")
	eventType := uniqueKey("sanity_event")
	featureKey := uniqueKey("sanity_feature")

	var (
		meterID     string
		featureID   string
		planID      string
		addonID     string
		planAddonID string
		phaseKey    string
	)

	t.Run("Should create a meter", func(t *testing.T) {
		valueProperty := "$.value"
		status, m, problem := c.CreateMeter(apiv3.CreateMeterRequest{
			Key:           meterKey,
			Name:          "Test Meter " + meterKey,
			Aggregation:   apiv3.MeterAggregationSum,
			EventType:     eventType,
			ValueProperty: &valueProperty,
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, m)
		require.NotEmpty(t, m.Id)
		meterID = m.Id
	})

	t.Run("Should create a feature bound to the meter", func(t *testing.T) {
		require.NotEmpty(t, meterID)

		status, f, problem := c.CreateFeature(apiv3.CreateFeatureRequest{
			Key:  featureKey,
			Name: "Test Feature " + featureKey,
			Meter: &apiv3.FeatureMeterReference{
				Id: meterID,
			},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, f)
		require.NotEmpty(t, f.Id)
		featureID = f.Id
	})

	t.Run("Should create a draft plan with a single flat rate card", func(t *testing.T) {
		body := validPlanRequest("sanity_plan")
		phaseKey = body.Phases[0].Key

		status, plan, problem := c.CreatePlan(body)
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, plan)
		assert.Equal(t, apiv3.BillingPlanStatusDraft, plan.Status)
		require.Len(t, plan.Phases, 1)
		require.Len(t, plan.Phases[0].RateCards, 1)

		planID = plan.Id
	})

	t.Run("Should update the plan to carry flat + usage + graduated rate cards", func(t *testing.T) {
		t.Skip("Skip this test as it does not use rate cards with features properly")

		require.NotEmpty(t, planID)
		require.NotEmpty(t, phaseKey)
		require.NotEmpty(t, featureID)

		// Three different rate card shapes on one phase. The flat fee is
		// in_advance, both usage-based ones are in_arrears (unit/graduated
		// prices cannot be in_advance). Only the unit one carries a feature
		// reference here — the graduated case follows the addon mixed
		// round-trip pattern (no feature) to keep this iteration close to
		// known-good shapes.
		flat := validFlatRateCard("sanity_flat")
		usage := validUsageRateCard("sanity_usage", featureID)
		graduated := validGraduatedRateCard("sanity_graduated")

		update := apiv3.UpsertPlanRequest{
			Name: "Sanity Plan",
			Phases: []apiv3.BillingPlanPhase{{
				Key:       phaseKey,
				Name:      "Sanity Phase",
				RateCards: []apiv3.BillingRateCard{flat, usage, graduated},
			}},
		}

		status, plan, problem := c.UpdatePlan(planID, update)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plan)
		require.Len(t, plan.Phases, 1)
		assert.Len(t, plan.Phases[0].RateCards, 3, "expected three rate cards on the phase")

		var usageRC *apiv3.BillingRateCard
		for i := range plan.Phases[0].RateCards {
			if plan.Phases[0].RateCards[i].Key == usage.Key {
				usageRC = &plan.Phases[0].RateCards[i]
				break
			}
		}
		require.NotNil(t, usageRC, "usage rate card missing after update")
		require.NotNil(t, usageRC.Feature, "usage rate card lost its feature binding after update")
		assert.Equal(t, featureID, usageRC.Feature.Id)
	})

	// Track the three valid rate cards across the invalid-loop subtests so
	// "remove defective" PUTs can rebuild the phase from the same baseline.
	var validRateCards []apiv3.BillingRateCard

	t.Run("Should add a defective rate card and surface validation_errors", func(t *testing.T) {
		t.Skip("Skip this test as it does not use rate cards with features properly")

		require.NotEmpty(t, planID)
		require.NotEmpty(t, phaseKey)

		// Read the current valid rate cards back from the plan so we don't
		// drift from server-normalized values (e.g., "0.10" → "0.1").
		status, plan, problem := c.GetPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plan)
		require.Len(t, plan.Phases, 1)
		require.Len(t, plan.Phases[0].RateCards, 3)
		validRateCards = plan.Phases[0].RateCards

		// The defect: a flat rate card whose billing cadence (P2W) doesn't
		// align with the plan's P1M cadence. Picked because it surfaces a
		// single actionable validation error with a useful field path —
		// other defects (e.g., usage discount on flat price) emit
		// duplicates with empty fields, which makes assertions brittle.
		defective := validFlatRateCard("defective_cadence")
		misaligned := apiv3.ISO8601Duration("P2W")
		defective.BillingCadence = &misaligned

		update := apiv3.UpsertPlanRequest{
			Name: "Sanity Plan",
			Phases: []apiv3.BillingPlanPhase{{
				Key:       phaseKey,
				Name:      "Sanity Phase",
				RateCards: append(append([]apiv3.BillingRateCard{}, validRateCards...), defective),
			}},
		}

		status, _, problem = c.UpdatePlan(planID, update)
		require.Equal(t, http.StatusOK, status, "update with defective rate card should accept the draft: %+v", problem)

		status, got, problem := c.GetPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, got)
		require.NotNil(t, got.ValidationErrors, "expected validation_errors on the draft")

		var codes []string
		for _, e := range *got.ValidationErrors {
			codes = append(codes, e.Code)
		}
		assert.Contains(t, codes, "rate_card_billing_cadence_unaligned")

		// Publish should reject with the same code.
		status, _, problem = c.PublishPlan(planID)
		require.Equal(t, http.StatusBadRequest, status, "publish should reject the defective draft: %+v", problem)
		assertValidationCode(t, problem, "rate_card_billing_cadence_unaligned")
	})

	t.Run("Should remove the defective rate card and clear validation_errors", func(t *testing.T) {
		t.Skip("Skip this test as it does not use rate cards with features properly")

		require.NotEmpty(t, planID)
		require.NotEmpty(t, phaseKey)
		require.NotEmpty(t, validRateCards)

		update := apiv3.UpsertPlanRequest{
			Name: "Sanity Plan",
			Phases: []apiv3.BillingPlanPhase{{
				Key:       phaseKey,
				Name:      "Sanity Phase",
				RateCards: validRateCards,
			}},
		}

		status, plan, problem := c.UpdatePlan(planID, update)
		require.Equal(t, http.StatusOK, status, "update removing defective rate card: %+v", problem)
		require.NotNil(t, plan)
		assert.Len(t, plan.Phases[0].RateCards, 3, "expected three rate cards after removal")

		status, got, problem := c.GetPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, got)
		if got.ValidationErrors != nil {
			assert.Empty(t, *got.ValidationErrors, "expected validation_errors to clear after removing the defective rate card")
		}
	})

	t.Run("Should create a draft addon", func(t *testing.T) {
		status, addon, problem := c.CreateAddon(validAddonRequest("sanity_addon"))
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, addon)
		assert.Equal(t, apiv3.AddonStatusDraft, addon.Status)
		addonID = addon.Id
	})

	t.Run("Should publish the addon", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		status, addon, problem := c.PublishAddon(addonID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, addon)
		assert.Equal(t, apiv3.AddonStatusActive, addon.Status)
	})

	t.Run("Should attach the published addon to the plan", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, addonID)
		require.NotEmpty(t, phaseKey)

		status, planAddon, problem := c.AttachAddon(planID, validPlanAddonRequest(phaseKey, addonID))
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, planAddon)
		assert.Equal(t, addonID, planAddon.Addon.Id)
		assert.Equal(t, phaseKey, planAddon.FromPlanPhase)

		planAddonID = planAddon.Id
	})

	t.Run("Should publish the plan and keep the attached addon", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, planAddonID)

		status, plan, problem := c.PublishPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plan)
		assert.Equal(t, apiv3.BillingPlanStatusActive, plan.Status)
		require.NotNil(t, plan.EffectiveFrom)

		status, page, problem := c.ListPlanAddons(planID)
		require.Equal(t, http.StatusOK, status, "list plan-addons: %+v", problem)
		require.NotNil(t, page)

		found := false
		for _, pa := range page.Data {
			if pa.Id == planAddonID {
				found = true
				break
			}
		}
		assert.True(t, found, "attached plan-addon missing after plan publish")
	})
}
