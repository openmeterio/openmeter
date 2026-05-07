package httpexpect_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

// TestProductCatalogSmokeV3 replicates the cross-cutting plan + addon authoring
// flow from e2e/productcatalog_smoke_v3_test.go using httpexpect for transport.
//
// Flow:
//   - Create a meter
//   - Create a feature bound to the meter
//   - Create a draft plan with a single flat rate card
//   - Update the plan to carry flat + usage (with feature) + graduated rate cards
//   - Add a defective rate card and confirm the draft-with-errors loop:
//     GET surfaces validation_errors, publish is rejected with the same code
//   - Remove the defective rate card and confirm validation_errors clears
//   - Create a draft addon and publish it
//   - Attach the published addon to the still-draft plan
//   - Publish the plan and confirm the attached addon survives the transition
func TestProductCatalogSmokeV3(t *testing.T) {
	e := newV3Expect(t)

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
		var meter apiv3.Meter
		e.POST("/meters").
			WithJSON(apiv3.CreateMeterRequest{
				Key:           meterKey,
				Name:          "Test Meter " + meterKey,
				Aggregation:   apiv3.MeterAggregationSum,
				EventType:     eventType,
				ValueProperty: &valueProperty,
			}).
			Expect().
			Status(http.StatusCreated).
			JSON().Decode(&meter)

		require.NotEmpty(t, meter.Id)
		meterID = meter.Id
	})

	t.Run("Should create a feature bound to the meter", func(t *testing.T) {
		require.NotEmpty(t, meterID)

		var feature apiv3.Feature
		e.POST("/features").
			WithJSON(apiv3.CreateFeatureRequest{
				Key:  featureKey,
				Name: "Test Feature " + featureKey,
				Meter: &struct {
					Filters *map[string]apiv3.QueryFilterStringMapItem `json:"filters,omitempty"`
					Id      apiv3.ULID                                 `json:"id"`
				}{
					Id: meterID,
				},
			}).
			Expect().
			Status(http.StatusCreated).
			JSON().Decode(&feature)

		require.NotEmpty(t, feature.Id)
		featureID = feature.Id
	})

	t.Run("Should create a draft plan with a single flat rate card", func(t *testing.T) {
		body := validPlanRequest("sanity_plan")
		phaseKey = body.Phases[0].Key

		var plan apiv3.BillingPlan
		e.POST("/plans").
			WithJSON(body).
			Expect().
			Status(http.StatusCreated).
			JSON().Decode(&plan)

		assert.Equal(t, apiv3.BillingPlanStatusDraft, plan.Status)
		require.Len(t, plan.Phases, 1)
		require.Len(t, plan.Phases[0].RateCards, 1)
		planID = plan.Id
	})

	t.Run("Should update the plan to carry flat + usage + graduated rate cards", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, phaseKey)
		require.NotEmpty(t, featureID)

		flat := validFlatRateCard("sanity_flat")
		usage := validUsageRateCard("sanity_usage", featureID)
		graduated := validGraduatedRateCard("sanity_graduated")

		var plan apiv3.BillingPlan
		e.PUT("/plans/" + planID).
			WithJSON(apiv3.UpsertPlanRequest{
				Name: "Sanity Plan",
				Phases: []apiv3.BillingPlanPhase{{
					Key:       phaseKey,
					Name:      "Sanity Phase",
					RateCards: []apiv3.BillingRateCard{flat, usage, graduated},
				}},
			}).
			Expect().
			Status(http.StatusOK).
			JSON().Decode(&plan)

		require.Len(t, plan.Phases, 1)
		assert.Len(t, plan.Phases[0].RateCards, 3)

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

	// Baseline rate cards captured after the valid update — reused in the
	// "remove defective" subtest to rebuild the phase without drift.
	var validRateCards []apiv3.BillingRateCard

	t.Run("Should add a defective rate card and surface validation_errors", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, phaseKey)

		// Read current valid rate cards back from the server to avoid
		// diverging from server-normalised values (e.g. "0.10" → "0.1").
		var current apiv3.BillingPlan
		e.GET("/plans/" + planID).Expect().Status(http.StatusOK).JSON().Decode(&current)
		require.Len(t, current.Phases, 1)
		require.Len(t, current.Phases[0].RateCards, 3)
		validRateCards = current.Phases[0].RateCards

		// Defect: billing cadence (P2W) misaligned with plan cadence (P1M).
		defective := validFlatRateCard("defective_cadence")
		misaligned := apiv3.ISO8601Duration("P2W")
		defective.BillingCadence = &misaligned

		// PUT with the defective rate card should succeed — drafts accept errors.
		e.PUT("/plans/" + planID).
			WithJSON(apiv3.UpsertPlanRequest{
				Name: "Sanity Plan",
				Phases: []apiv3.BillingPlanPhase{{
					Key:       phaseKey,
					Name:      "Sanity Phase",
					RateCards: append(append([]apiv3.BillingRateCard{}, validRateCards...), defective),
				}},
			}).
			Expect().
			Status(http.StatusOK)

		// GET should now surface validation_errors.
		var planWithErrors apiv3.BillingPlan
		e.GET("/plans/" + planID).Expect().Status(http.StatusOK).JSON().Decode(&planWithErrors)
		require.NotNil(t, planWithErrors.ValidationErrors, "expected validation_errors on the draft")

		var codes []string
		for _, ve := range *planWithErrors.ValidationErrors {
			codes = append(codes, ve.Code)
		}
		assert.Contains(t, codes, "rate_card_billing_cadence_unaligned")

		// Publish must be rejected with the same code.
		prob := decodeProblem(e.POST("/plans/" + planID + "/publish").Expect().Status(http.StatusBadRequest))
		assertValidationCode(t, prob, "rate_card_billing_cadence_unaligned")
	})

	t.Run("Should remove the defective rate card and clear validation_errors", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, phaseKey)
		require.NotEmpty(t, validRateCards)

		var plan apiv3.BillingPlan
		e.PUT("/plans/" + planID).
			WithJSON(apiv3.UpsertPlanRequest{
				Name: "Sanity Plan",
				Phases: []apiv3.BillingPlanPhase{{
					Key:       phaseKey,
					Name:      "Sanity Phase",
					RateCards: validRateCards,
				}},
			}).
			Expect().
			Status(http.StatusOK).
			JSON().Decode(&plan)

		assert.Len(t, plan.Phases[0].RateCards, 3)

		var clean apiv3.BillingPlan
		e.GET("/plans/" + planID).Expect().Status(http.StatusOK).JSON().Decode(&clean)
		if clean.ValidationErrors != nil {
			assert.Empty(t, *clean.ValidationErrors, "validation_errors should clear after removing defective rate card")
		}
	})

	t.Run("Should create a draft addon", func(t *testing.T) {
		var addon apiv3.Addon
		e.POST("/addons").
			WithJSON(validAddonRequest("sanity_addon")).
			Expect().
			Status(http.StatusCreated).
			JSON().Decode(&addon)

		assert.Equal(t, apiv3.AddonStatusDraft, addon.Status)
		addonID = addon.Id
	})

	t.Run("Should publish the addon", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		var addon apiv3.Addon
		e.POST("/addons/" + addonID + "/publish").
			Expect().
			Status(http.StatusOK).
			JSON().Decode(&addon)

		assert.Equal(t, apiv3.AddonStatusActive, addon.Status)
	})

	t.Run("Should attach the published addon to the plan", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, addonID)
		require.NotEmpty(t, phaseKey)

		var planAddon apiv3.PlanAddon
		e.POST("/plans/" + planID + "/addons").
			WithJSON(validPlanAddonRequest(phaseKey, addonID)).
			Expect().
			Status(http.StatusCreated).
			JSON().Decode(&planAddon)

		assert.Equal(t, addonID, planAddon.Addon.Id)
		assert.Equal(t, phaseKey, planAddon.FromPlanPhase)
		planAddonID = planAddon.Id
	})

	t.Run("Should publish the plan and keep the attached addon", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, planAddonID)

		var plan apiv3.BillingPlan
		e.POST("/plans/" + planID + "/publish").
			Expect().
			Status(http.StatusOK).
			JSON().Decode(&plan)

		assert.Equal(t, apiv3.BillingPlanStatusActive, plan.Status)
		require.NotNil(t, plan.EffectiveFrom)

		var page apiv3.PlanAddonPagePaginatedResponse
		e.GET("/plans/" + planID + "/addons").
			Expect().
			Status(http.StatusOK).
			JSON().Decode(&page)

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
