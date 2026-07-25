package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
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

	eventTypes := []string{
		uniqueKey("sanity_event"),
		uniqueKey("sanity_event"),
	}

	meterKeys := []string{
		uniqueKey("sanity_meter"),
		uniqueKey("sanity_meter"),
	}

	meters := make([]v3sdk.Meter, 0, len(meterKeys))

	for i := range meterKeys {
		valueProperty := "$.value"

		m, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
			Key:           meterKeys[i],
			Name:          "Test Meter " + meterKeys[i],
			Aggregation:   v3sdk.MeterAggregationSum,
			EventType:     eventTypes[i],
			ValueProperty: &valueProperty,
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, m)
		require.NotEmpty(t, m.ID)

		meters = append(meters, *m)
	}

	featureKeys := []string{
		uniqueKey("sanity_feature"),
		uniqueKey("sanity_feature"),
	}

	features := make([]v3sdk.Feature, 0, len(featureKeys))

	for i := range featureKeys {
		f, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:  featureKeys[i],
			Name: "Test Feature " + featureKeys[i],
			Meter: &v3sdk.FeatureMeterReferenceInput{
				ID: meters[i].ID,
			},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, f)
		require.NotEmpty(t, f.ID)

		features = append(features, *f)
	}

	var (
		planID      string
		addonID     string
		planAddonID string
		phaseKey    string
	)

	t.Run("Should create a draft plan with a single flat rate card", func(t *testing.T) {
		body := validPlanRequest("sanity_plan")
		phaseKey = body.Phases[0].Key

		plan, err := c.Plans.Create(t.Context(), body)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)
		assert.Equal(t, v3sdk.PlanStatusDraft, plan.Status)
		require.Len(t, plan.Phases, 1)
		require.Len(t, plan.Phases[0].RateCards, 1)

		planID = plan.ID
	})

	t.Run("Should update the plan to carry flat + usage + graduated rate cards", func(t *testing.T) {
		// Three different rate card shapes on one phase. The flat fee is
		// in_advance, both usage-based ones are in_arrears (unit/graduated
		// prices cannot be in_advance). Only the unit one carries a feature
		// reference here — the graduated case follows the addon mixed
		// round-trip pattern (no feature) to keep this iteration close to
		// known-good shapes.
		flat := validFlatRateCard("sanity_flat")
		usage := validUnitRateCard(features[0])
		graduated := validGraduatedRateCard(features[1])

		update := v3sdk.UpsertPlanRequest{
			Name: "Sanity Plan",
			Phases: []v3sdk.PlanPhaseInput{{
				Key:       phaseKey,
				Name:      "Sanity Phase",
				RateCards: []v3sdk.RateCardInput{flat, usage, graduated},
			}},
		}

		plan, err := c.Plans.Update(t.Context(), planID, update)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)
		require.Len(t, plan.Phases, 1)
		assert.Len(t, plan.Phases[0].RateCards, 3, "expected three rate cards on the phase")

		var usageRC *v3sdk.RateCard
		for i := range plan.Phases[0].RateCards {
			if plan.Phases[0].RateCards[i].Key == usage.Key {
				usageRC = &plan.Phases[0].RateCards[i]
				break
			}
		}
		require.NotNil(t, usageRC, "usage rate card missing after update")
		require.NotNil(t, usageRC.Feature, "usage rate card lost its feature binding after update")
		assert.Equal(t, usage.Feature.ID, usageRC.Feature.ID)
	})

	// Track the three valid rate cards across the invalid-loop subtests so
	// "remove defective" PUTs can rebuild the phase from the same baseline.
	var validRateCards []v3sdk.RateCardInput

	t.Run("Should add a defective rate card and surface validation_errors", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, phaseKey)

		// Read the current valid rate cards back from the plan so we don't
		// drift from server-normalized values (e.g., "0.10" → "0.1").
		plan, err := c.Plans.Get(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)
		require.Len(t, plan.Phases, 1)
		require.Len(t, plan.Phases[0].RateCards, 3)
		validRateCards = lo.Map(plan.Phases[0].RateCards, func(rc v3sdk.RateCard, _ int) v3sdk.RateCardInput {
			return v3sdk.RateCardInput{
				Name:           rc.Name,
				Description:    rc.Description,
				Labels:         lo.EmptyableToPtr(rc.Labels),
				Key:            rc.Key,
				Feature:        rc.Feature,
				BillingCadence: rc.BillingCadence,
				Price:          rc.Price,
				UnitConfig:     rc.UnitConfig,
				PaymentTerm:    rc.PaymentTerm,
				Commitments:    rc.Commitments,
				Discounts:      rc.Discounts,
				TaxConfig:      rc.TaxConfig,
				Entitlement:    rc.Entitlement,
			}
		})

		// The defect: a flat rate card whose billing cadence (P2W) doesn't
		// align with the plan's P1M cadence. Picked because it surfaces a
		// single actionable validation error with a useful field path —
		// other defects (e.g., usage discount on flat price) emit
		// duplicates with empty fields, which makes assertions brittle.
		defective := validFlatRateCard("defective_cadence")
		misaligned := "P2W"
		defective.BillingCadence = &misaligned

		update := v3sdk.UpsertPlanRequest{
			Name: "Sanity Plan",
			Phases: []v3sdk.PlanPhaseInput{{
				Key:       phaseKey,
				Name:      "Sanity Phase",
				RateCards: append(append([]v3sdk.RateCardInput{}, validRateCards...), defective),
			}},
		}

		_, err = c.Plans.Update(t.Context(), planID, update)
		c.requireStatus(http.StatusOK, err)

		got, err := c.Plans.Get(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, got)
		require.NotEmpty(t, got.ValidationErrors, "expected validation_errors on the draft")

		var codes []string
		for _, e := range got.ValidationErrors {
			codes = append(codes, e.Code)
		}
		assert.Contains(t, codes, "rate_card_billing_cadence_unaligned")

		// Publish should reject with the same code.
		_, err = c.Plans.Publish(t.Context(), planID)
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertValidationCode(t, problem, "rate_card_billing_cadence_unaligned")
	})

	t.Run("Should remove the defective rate card and clear validation_errors", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, phaseKey)
		require.NotEmpty(t, validRateCards)

		update := v3sdk.UpsertPlanRequest{
			Name: "Sanity Plan",
			Phases: []v3sdk.PlanPhaseInput{{
				Key:       phaseKey,
				Name:      "Sanity Phase",
				RateCards: validRateCards,
			}},
		}

		plan, err := c.Plans.Update(t.Context(), planID, update)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)
		assert.Len(t, plan.Phases[0].RateCards, 3, "expected three rate cards after removal")

		got, err := c.Plans.Get(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, got)
		assert.Empty(t, got.ValidationErrors, "expected validation_errors to clear after removing the defective rate card")
	})

	t.Run("Should create a draft addon", func(t *testing.T) {
		addon, err := c.Addons.Create(t.Context(), validAddonRequest("sanity_addon"))
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, addon)
		assert.Equal(t, v3sdk.AddonStatusDraft, addon.Status)
		addonID = addon.ID
	})

	t.Run("Should publish the addon", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		addon, err := c.Addons.Publish(t.Context(), addonID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, addon)
		assert.Equal(t, v3sdk.AddonStatusActive, addon.Status)
	})

	t.Run("Should attach the published addon to the plan", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, addonID)
		require.NotEmpty(t, phaseKey)

		planAddon, err := c.PlanAddons.Create(t.Context(), planID, validPlanAddonRequest(phaseKey, addonID))
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, planAddon)
		assert.Equal(t, addonID, planAddon.Addon.ID)
		assert.Equal(t, phaseKey, planAddon.FromPlanPhase)

		planAddonID = planAddon.ID
	})

	t.Run("Should publish the plan and keep the attached addon", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, planAddonID)

		plan, err := c.Plans.Publish(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)
		assert.Equal(t, v3sdk.PlanStatusActive, plan.Status)
		require.NotNil(t, plan.EffectiveFrom)

		page, err := c.PlanAddons.List(t.Context(), planID, v3sdk.PlanAddonListParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, page)

		found := false
		for _, pa := range page.Data {
			if pa.ID == planAddonID {
				found = true
				break
			}
		}
		assert.True(t, found, "attached plan-addon missing after plan publish")
	})
}
