package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// Plan lifecycle: create → get → list → update → publish → archive → delete,
// asserting the status and effective_from/to transitions at each step.
func TestV3PlanLifecycle(t *testing.T) {
	c := newV3Client(t)

	createBody := validPlanRequest("test_v3_plan_lifecycle")
	planKey := createBody.Key
	originalPhaseKey := createBody.Phases[0].Key

	var planID string

	t.Run("Should create a plan in draft status", func(t *testing.T) {
		plan, err := c.Plans.Create(t.Context(), createBody)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)

		assert.Equal(t, planKey, plan.Key)
		assert.EqualValues(t, 1, plan.Version)
		assert.Equal(t, v3sdk.PlanStatusDraft, plan.Status)
		assert.Nil(t, plan.EffectiveFrom)
		assert.Nil(t, plan.EffectiveTo)

		planID = plan.ID
	})

	t.Run("Should get the plan in draft", func(t *testing.T) {
		require.NotEmpty(t, planID)

		plan, err := c.Plans.Get(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)

		assert.Equal(t, planID, plan.ID)
		assert.Equal(t, v3sdk.PlanStatusDraft, plan.Status)
		assert.EqualValues(t, 1, plan.Version)
		assert.Nil(t, plan.EffectiveFrom)
	})

	t.Run("Should list plans and find the created plan", func(t *testing.T) {
		// Bump page size so a shared DB with prior fixtures doesn't push this
		// freshly-created plan off page 1.
		page, err := c.Plans.List(t.Context(), v3sdk.PlanListParams{
			Page: &v3sdk.PageParams{Size: lo.ToPtr(1000)},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, page)

		found := false
		for _, p := range page.Data {
			if p.ID == planID {
				found = true
				break
			}
		}
		assert.True(t, found, "created plan not found in list")
	})

	t.Run("Should update the plan — rename phase and add a second rate card", func(t *testing.T) {
		require.NotEmpty(t, planID)

		updatedPhase := createBody.Phases[0]
		updatedPhase.Name = "Phase Renamed"
		updatedPhase.RateCards = append(updatedPhase.RateCards, validFlatRateCard("fee_2"))

		updateBody := v3sdk.UpsertPlanRequest{
			Name:   createBody.Name,
			Phases: []v3sdk.PlanPhaseInput{updatedPhase},
		}

		plan, err := c.Plans.Update(t.Context(), planID, updateBody)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)

		require.Len(t, plan.Phases, 1)
		assert.Equal(t, originalPhaseKey, plan.Phases[0].Key, "phase key is immutable")
		assert.Equal(t, "Phase Renamed", plan.Phases[0].Name)
		assert.Len(t, plan.Phases[0].RateCards, 2, "second rate card was not added")
		assert.Equal(t, v3sdk.PlanStatusDraft, plan.Status)
	})

	t.Run("Should get the plan and see the update persisted", func(t *testing.T) {
		require.NotEmpty(t, planID)

		plan, err := c.Plans.Get(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)

		require.Len(t, plan.Phases, 1)
		assert.Equal(t, "Phase Renamed", plan.Phases[0].Name)
		assert.Len(t, plan.Phases[0].RateCards, 2)
	})

	t.Run("Should publish the plan", func(t *testing.T) {
		require.NotEmpty(t, planID)

		plan, err := c.Plans.Publish(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)

		assert.Equal(t, v3sdk.PlanStatusActive, plan.Status)
		assert.NotNil(t, plan.EffectiveFrom)
		assert.Nil(t, plan.EffectiveTo)
	})

	t.Run("Should archive the published plan", func(t *testing.T) {
		require.NotEmpty(t, planID)

		plan, err := c.Plans.Archive(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)

		assert.Equal(t, v3sdk.PlanStatusArchived, plan.Status)
		assert.NotNil(t, plan.EffectiveTo)
	})

	t.Run("Should delete an archived plan", func(t *testing.T) {
		require.NotEmpty(t, planID)

		err := c.Plans.Delete(t.Context(), planID)
		c.requireStatus(http.StatusNoContent, err)
	})

	t.Run("Should return deleted_at after deletion", func(t *testing.T) {
		require.NotEmpty(t, planID)

		plan, err := c.Plans.Get(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)

		assert.NotNil(t, plan.DeletedAt)
	})
}

// Phase validation at create time.
//
// Only two phase defects are actually create-time on the live server. Others
// (non-last without duration, last with duration, empty rate cards) are
// silently accepted as drafts and surface as ProductCatalogValidationErrors on
// GET / Publish — those are covered by TestV3PlanInvalidDraftLifecycle and
// TestV3PlanPublishValidationCases.
//
// The two true create-time cases also use different error shapes:
//   - Zero phases → TypeSpec `min_items` (BaseAPIError.InvalidParameters).
//     Note: the PC code plan_with_no_phases is shadowed by schema validation
//     and never reachable via create.
//   - Duplicate phase key → ProductCatalogValidationError
//     (extensions.validationErrors[].code = plan_phase_duplicated_key).
func TestV3PlanPhaseValidationOnCreate(t *testing.T) {
	t.Run("zero phases → 400 schema min_items", func(t *testing.T) {
		c := newV3Client(t)

		body := validPlanRequest("phase_validation_zero")
		body.Phases = []v3sdk.PlanPhaseInput{}

		_, err := c.Plans.Create(t.Context(), body)
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertInvalidParameterRule(t, problem, "min_items")
	})

	t.Run("duplicate phase key → 400 plan_phase_duplicated_key", func(t *testing.T) {
		c := newV3Client(t)

		body := validPlanRequest("phase_validation_dup")
		p1 := validPlanPhase("same_key", false /* isLast */)
		p2 := validPlanPhase("ignored", true /* isLast */)
		p2.Key = p1.Key
		body.Phases = []v3sdk.PlanPhaseInput{p1, p2}

		_, err := c.Plans.Create(t.Context(), body)
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertValidationCode(t, problem, "plan_phase_duplicated_key")
	})
}

// Invalid-draft lifecycle. Shares a single invalid-draft fixture across two
// narratives:
//   - validation_errors surfaces on GET for invalid drafts.
//   - publish is blocked by the same code; fix via PUT; republish succeeds.
//
// Uses "phase with zero rate cards" as the defect (accepted at create, flagged
// on GET, rejected at publish). Duplicate rate-card keys would seem a natural
// fit but are create-time rejected, so can't be used as a draft-with-errors
// fixture.
func TestV3PlanInvalidDraftLifecycle(t *testing.T) {
	c := newV3Client(t)

	phase := validPlanPhase("empty_rc_phase", true /* isLast */)
	phase.RateCards = []v3sdk.RateCardInput{}

	body := validPlanRequest("invalid_draft_lifecycle")
	body.Phases = []v3sdk.PlanPhaseInput{phase}

	var planID string

	t.Run("create accepts the invalid draft", func(t *testing.T) {
		plan, err := c.Plans.Create(t.Context(), body)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)
		planID = plan.ID
	})

	t.Run("validation_errors surfaces on GET", func(t *testing.T) {
		require.NotEmpty(t, planID)
		got, err := c.Plans.Get(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, got)
		require.NotEmpty(t, got.ValidationErrors, "expected validation_errors on the draft")

		var codes []string
		for _, e := range got.ValidationErrors {
			codes = append(codes, e.Code)
		}
		assert.Contains(t, codes, "plan_phase_has_no_rate_cards")
	})

	t.Run("publish rejects with the same code", func(t *testing.T) {
		require.NotEmpty(t, planID)
		_, err := c.Plans.Publish(t.Context(), planID)
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertValidationCode(t, problem, "plan_phase_has_no_rate_cards")
	})

	t.Run("fix by adding a rate card to the phase", func(t *testing.T) {
		require.NotEmpty(t, planID)
		update := v3sdk.UpsertPlanRequest{
			Name: body.Name,
			Phases: []v3sdk.PlanPhaseInput{{
				Key:       phase.Key,
				Name:      phase.Name,
				Duration:  phase.Duration,
				RateCards: []v3sdk.RateCardInput{validFlatRateCard("added_rc")},
			}},
		}
		_, err := c.Plans.Update(t.Context(), planID, update)
		c.requireStatus(http.StatusOK, err)
	})

	t.Run("publish succeeds after fix", func(t *testing.T) {
		require.NotEmpty(t, planID)
		published, err := c.Plans.Publish(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, published)
		assert.Equal(t, v3sdk.PlanStatusActive, published.Status)
	})
}

// A second draft with the same key while v1 is still draft is rejected.
// Only one draft per key may exist at a time.
func TestV3PlanDuplicateDraftKeyRejected(t *testing.T) {
	c := newV3Client(t)

	first := validPlanRequest("duplicate_draft")
	key := first.Key

	plan, err := c.Plans.Create(t.Context(), first)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plan)
	require.Equal(t, v3sdk.PlanStatusDraft, plan.Status)

	// Second POST with the same key, while v1 is still draft.
	second := validPlanRequest("duplicate_draft_ignored")
	second.Key = key

	_, err = c.Plans.Create(t.Context(), second)
	apiErr, ok := v3sdk.AsAPIError(err)
	require.True(t, ok, "expected APIError, got %T: %v", err, err)
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusConflict}, apiErr.StatusCode,
		"expected 400 or 409 for duplicate draft key, got %d: %s", apiErr.StatusCode, string(apiErr.RawBody))
	require.NotEmpty(t, apiErr.RawBody)
	var problem v3Problem
	require.NoError(t, apiErr.Decode(&problem))
	require.True(t, problem.Status != 0 || problem.Title != "",
		"expected a problem+json body, got: %s", string(apiErr.RawBody))
}

// Delete-while-active is rejected; after archive, delete is accepted.
func TestV3PlanDeleteWhileActiveRejected(t *testing.T) {
	c := newV3Client(t)

	plan, err := c.Plans.Create(t.Context(), validPlanRequest("delete_while_active"))
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plan)

	_, err = c.Plans.Publish(t.Context(), plan.ID)
	c.requireStatus(http.StatusOK, err)

	err = c.Plans.Delete(t.Context(), plan.ID)
	problem := requireProblem(t, err, http.StatusBadRequest)
	assert.NotNil(t, problem, "delete-while-active should reject")

	_, err = c.Plans.Archive(t.Context(), plan.ID)
	c.requireStatus(http.StatusOK, err)

	err = c.Plans.Delete(t.Context(), plan.ID)
	c.requireStatus(http.StatusNoContent, err)
}

// Update after publish is rejected.
func TestV3PlanUpdateAfterPublishRejected(t *testing.T) {
	c := newV3Client(t)

	createBody := validPlanRequest("update_after_publish")

	plan, err := c.Plans.Create(t.Context(), createBody)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plan)

	_, err = c.Plans.Publish(t.Context(), plan.ID)
	c.requireStatus(http.StatusOK, err)

	// Attempt to rename a phase after publish.
	renamed := createBody.Phases[0]
	renamed.Name = "Renamed After Publish"

	update := v3sdk.UpsertPlanRequest{
		Name:   createBody.Name,
		Phases: []v3sdk.PlanPhaseInput{renamed},
	}

	_, err = c.Plans.Update(t.Context(), plan.ID, update)
	problem := requireProblem(t, err, http.StatusBadRequest)
	assertProblemDetail(t, problem, "only Plans in [draft scheduled] can be updated")
}

// Plan versioning and auto-archive. Create v1, publish it, create a new plan
// with the same key, assert v2 is auto-assigned. Publish v2 and assert v1 is
// auto-archived with EffectiveTo == v2.EffectiveFrom (transactional auto-archive).
func TestV3PlanVersioningAndAutoArchive(t *testing.T) {
	c := newV3Client(t)

	createBody := validPlanRequest("versioning")
	sharedKey := createBody.Key

	v1, err := c.Plans.Create(t.Context(), createBody)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, v1)
	assert.EqualValues(t, 1, v1.Version)

	v1Active, err := c.Plans.Publish(t.Context(), v1.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, v1Active)
	require.Equal(t, v3sdk.PlanStatusActive, v1Active.Status)

	// v2: same key, different internal fixtures.
	v2Body := validPlanRequest("versioning_v2")
	v2Body.Key = sharedKey

	v2, err := c.Plans.Create(t.Context(), v2Body)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, v2)
	assert.EqualValues(t, 2, v2.Version)
	assert.Equal(t, v3sdk.PlanStatusDraft, v2.Status)
	assert.Equal(t, sharedKey, v2.Key)
	assert.NotEqual(t, v1.ID, v2.ID, "v1 and v2 must have distinct IDs")

	v2Active, err := c.Plans.Publish(t.Context(), v2.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, v2Active)
	require.Equal(t, v3sdk.PlanStatusActive, v2Active.Status)
	require.NotNil(t, v2Active.EffectiveFrom)

	// v1 should now be auto-archived with EffectiveTo == v2.EffectiveFrom.
	v1After, err := c.Plans.Get(t.Context(), v1.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, v1After)
	assert.Equal(t, v3sdk.PlanStatusArchived, v1After.Status)
	require.NotNil(t, v1After.EffectiveTo)
	assert.True(t, v1After.EffectiveTo.Equal(*v2Active.EffectiveFrom),
		"v1.EffectiveTo (%s) must equal v2.EffectiveFrom (%s)",
		v1After.EffectiveTo, v2Active.EffectiveFrom)
}

// TestV3PlanPublishValidationCases covers the three cases that were
// incorrectly listed under scenario #7 but turned out to be publish-time
// validations: the server accepts them as drafts and only rejects at publish.
// Each case creates a draft, attempts publish, asserts the expected code.
func TestV3PlanPublishValidationCases(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*v3sdk.CreatePlanRequest)
		code   string
	}{
		{
			name: "non-last phase missing duration → plan_has_non_last_phase_with_no_duration",
			mutate: func(r *v3sdk.CreatePlanRequest) {
				nonLast := validPlanPhase("non_last", true /* isLast — no duration */)
				last := validPlanPhase("last", true /* isLast */)
				r.Phases = []v3sdk.PlanPhaseInput{nonLast, last}
			},
			code: "plan_has_non_last_phase_with_no_duration",
		},
		{
			name: "last phase with a duration → plan_has_last_phase_with_duration",
			mutate: func(r *v3sdk.CreatePlanRequest) {
				first := validPlanPhase("first", false /* isLast */)
				last := validPlanPhase("last", false /* bounded — should be open */)
				r.Phases = []v3sdk.PlanPhaseInput{first, last}
			},
			code: "plan_has_last_phase_with_duration",
		},
		{
			name: "phase with zero rate cards → plan_phase_has_no_rate_cards",
			mutate: func(r *v3sdk.CreatePlanRequest) {
				phase := validPlanPhase("empty_rc", true /* isLast */)
				phase.RateCards = []v3sdk.RateCardInput{}
				r.Phases = []v3sdk.PlanPhaseInput{phase}
			},
			code: "plan_phase_has_no_rate_cards",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newV3Client(t)

			body := validPlanRequest("publish_validation")
			tc.mutate(&body)

			plan, err := c.Plans.Create(t.Context(), body)
			c.requireStatus(http.StatusCreated, err)
			require.NotNil(t, plan)

			_, err = c.Plans.Publish(t.Context(), plan.ID)
			problem := requireProblem(t, err, http.StatusBadRequest)
			assertValidationCode(t, problem, tc.code)
		})
	}
}

// An unknown currency must be rejected with a structured 400 +
// `currency_invalid` code, not bubble up as a 500 from a handler-level
// fmt.Errorf.
func TestV3PlanInvalidCurrency(t *testing.T) {
	c := newV3Client(t)

	body := validPlanRequest("invalid_currency")
	body.Currency = "ZZZ"

	_, err := c.Plans.Create(t.Context(), body)
	problem := requireProblem(t, err, http.StatusBadRequest)
	assertValidationCode(t, problem, "currency_invalid")
}

// TestV3PlanReadTranslatesV1DynamicAndPackagePrices verifies the v3 read-side
// translation of v1 dynamic and package prices into a unit price plus a
// synthesized unit_config on the rate card.
//
// Mapping (api/v3/handlers/plans/convert.go):
//   - dynamic(multiplier=m)      → unit(amount=1) + unit_config{operation=multiply, conversion_factor=m}
//   - package(amount=a, qpp=q)   → unit(amount=a) + unit_config{operation=divide, conversion_factor=q, rounding=ceiling}
//
// Spend commitments (min/max amount) carried on the v1 price flow through to
// the v3 rate card's commitments field unchanged.
//
// The plan is authored via the v1 SDK and read via v3 GET and v3 LIST. The
// unit_config field is read-only on v3, so this test does not exercise a v3
// write path for it.
func TestV3PlanReadTranslatesV1DynamicAndPackagePrices(t *testing.T) {
	v1 := initClient(t)
	v3 := newV3Client(t)

	suffix := uniqueKey("v3_translates_v1")
	planKey := suffix
	dynamicRCKey := "dynamic_rc_" + suffix
	packageRCKey := "package_rc_" + suffix

	dynamicPrice := api.RateCardUsageBasedPrice{}
	require.NoError(t, dynamicPrice.FromDynamicPriceWithCommitments(api.DynamicPriceWithCommitments{
		Type:          api.DynamicPriceWithCommitmentsTypeDynamic,
		Multiplier:    lo.ToPtr(api.Numeric("1.2")),
		MinimumAmount: lo.ToPtr(api.Numeric("10")),
		MaximumAmount: lo.ToPtr(api.Numeric("100")),
	}))

	dynamicRC := api.RateCard{}
	require.NoError(t, dynamicRC.FromRateCardUsageBased(api.RateCardUsageBased{
		Type:           api.RateCardUsageBasedTypeUsageBased,
		Name:           "Dynamic RC",
		Key:            dynamicRCKey,
		BillingCadence: "P1M",
		Price:          &dynamicPrice,
	}))

	packagePrice := api.RateCardUsageBasedPrice{}
	require.NoError(t, packagePrice.FromPackagePriceWithCommitments(api.PackagePriceWithCommitments{
		Type:               api.PackagePriceWithCommitmentsTypePackage,
		Amount:             "0.5",
		QuantityPerPackage: "1000",
		MinimumAmount:      lo.ToPtr(api.Numeric("5")),
	}))

	packageRC := api.RateCard{}
	require.NoError(t, packageRC.FromRateCardUsageBased(api.RateCardUsageBased{
		Type:           api.RateCardUsageBasedTypeUsageBased,
		Name:           "Package RC",
		Key:            packageRCKey,
		BillingCadence: "P1M",
		Price:          &packagePrice,
	}))

	planCreate := api.PlanCreate{
		Currency:       api.CurrencyCode("USD"),
		Name:           "v1 Plan with Dynamic and Package Prices",
		Key:            planKey,
		BillingCadence: "P1M",
		Phases: []api.PlanPhase{
			{
				Name:      "Phase 1",
				Key:       "phase_1_" + suffix,
				RateCards: []api.RateCard{dynamicRC, packageRC},
			},
		},
	}

	var planID string

	t.Run("Should create the v1 plan with dynamic and package rate cards", func(t *testing.T) {
		resp, err := v1.CreatePlanWithResponse(t.Context(), planCreate)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON201)

		planID = resp.JSON201.Id
	})

	t.Run("v3 GET should translate dynamic price to unit + multiply unit_config", func(t *testing.T) {
		require.NotEmpty(t, planID)

		plan, err := v3.Plans.Get(t.Context(), planID)
		v3.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)

		rc := findRateCardByKey(t, plan, dynamicRCKey)

		assertUnitPriceAmount(t, rc, "1")

		require.NotNil(t, rc.UnitConfig, "expected synthesized unit_config")
		assert.Equal(t, v3sdk.UnitConfigOperationMultiply, rc.UnitConfig.Operation)
		assert.Equal(t, v3sdk.Numeric("1.2"), rc.UnitConfig.ConversionFactor)
		assert.Nil(t, rc.UnitConfig.Rounding, "dynamic translation does not set rounding")

		require.NotNil(t, rc.Commitments, "v1 commitments should round-trip via v3")
		assert.Equal(t, lo.ToPtr(v3sdk.Numeric("10")), rc.Commitments.MinimumAmount)
		assert.Equal(t, lo.ToPtr(v3sdk.Numeric("100")), rc.Commitments.MaximumAmount)
	})

	t.Run("v3 GET should translate package price to unit + divide+ceiling unit_config", func(t *testing.T) {
		require.NotEmpty(t, planID)

		plan, err := v3.Plans.Get(t.Context(), planID)
		v3.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)

		rc := findRateCardByKey(t, plan, packageRCKey)

		assertUnitPriceAmount(t, rc, "0.5")

		require.NotNil(t, rc.UnitConfig, "expected synthesized unit_config")
		assert.Equal(t, v3sdk.UnitConfigOperationDivide, rc.UnitConfig.Operation)
		assert.Equal(t, v3sdk.Numeric("1000"), rc.UnitConfig.ConversionFactor)
		require.NotNil(t, rc.UnitConfig.Rounding, "package translation must set rounding=ceiling")
		assert.Equal(t, v3sdk.UnitConfigRoundingModeCeiling, *rc.UnitConfig.Rounding)

		require.NotNil(t, rc.Commitments, "v1 commitments should round-trip via v3")
		assert.Equal(t, lo.ToPtr(v3sdk.Numeric("5")), rc.Commitments.MinimumAmount)
		assert.Nil(t, rc.Commitments.MaximumAmount)
	})

	t.Run("v3 LIST should include the plan with both rate cards translated", func(t *testing.T) {
		require.NotEmpty(t, planID)

		// Bump page size so a fresh fixture isn't pushed off page 1 on a shared DB.
		page, err := v3.Plans.List(t.Context(), v3sdk.PlanListParams{
			Page: &v3sdk.PageParams{Size: lo.ToPtr(1000)},
		})
		v3.requireStatus(http.StatusOK, err)
		require.NotNil(t, page)

		var found *v3sdk.Plan
		for i := range page.Data {
			if page.Data[i].ID == planID {
				found = &page.Data[i]
				break
			}
		}
		require.NotNil(t, found, "created plan not in list response (the v3 list handler should no longer skip plans with v1 dynamic/package prices)")

		dynRC := findRateCardByKey(t, found, dynamicRCKey)
		assertUnitPriceAmount(t, dynRC, "1")
		require.NotNil(t, dynRC.UnitConfig)
		assert.Equal(t, v3sdk.UnitConfigOperationMultiply, dynRC.UnitConfig.Operation)
		assert.Equal(t, v3sdk.Numeric("1.2"), dynRC.UnitConfig.ConversionFactor)

		pkgRC := findRateCardByKey(t, found, packageRCKey)
		assertUnitPriceAmount(t, pkgRC, "0.5")
		require.NotNil(t, pkgRC.UnitConfig)
		assert.Equal(t, v3sdk.UnitConfigOperationDivide, pkgRC.UnitConfig.Operation)
		assert.Equal(t, v3sdk.Numeric("1000"), pkgRC.UnitConfig.ConversionFactor)
		require.NotNil(t, pkgRC.UnitConfig.Rounding)
		assert.Equal(t, v3sdk.UnitConfigRoundingModeCeiling, *pkgRC.UnitConfig.Rounding)
	})
}
