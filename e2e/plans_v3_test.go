package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
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
		status, plan, problem := c.CreatePlan(createBody)
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, plan)

		assert.Equal(t, planKey, plan.Key)
		assert.Equal(t, 1, plan.Version)
		assert.Equal(t, apiv3.BillingPlanStatusDraft, plan.Status)
		assert.Nil(t, plan.EffectiveFrom)
		assert.Nil(t, plan.EffectiveTo)

		planID = plan.Id
	})

	t.Run("Should get the plan in draft", func(t *testing.T) {
		require.NotEmpty(t, planID)

		status, plan, problem := c.GetPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plan)

		assert.Equal(t, planID, plan.Id)
		assert.Equal(t, apiv3.BillingPlanStatusDraft, plan.Status)
		assert.Equal(t, 1, plan.Version)
		assert.Nil(t, plan.EffectiveFrom)
	})

	t.Run("Should list plans and find the created plan", func(t *testing.T) {
		// Bump page size so a shared DB with prior fixtures doesn't push this
		// freshly-created plan off page 1.
		status, page, problem := c.ListPlans(withPageSize(1000))
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, page)

		found := false
		for _, p := range page.Data {
			if p.Id == planID {
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

		updateBody := apiv3.UpsertPlanRequest{
			Name:   createBody.Name,
			Phases: []apiv3.BillingPlanPhase{updatedPhase},
		}

		status, plan, problem := c.UpdatePlan(planID, updateBody)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plan)

		require.Len(t, plan.Phases, 1)
		assert.Equal(t, originalPhaseKey, plan.Phases[0].Key, "phase key is immutable")
		assert.Equal(t, "Phase Renamed", plan.Phases[0].Name)
		assert.Len(t, plan.Phases[0].RateCards, 2, "second rate card was not added")
		assert.Equal(t, apiv3.BillingPlanStatusDraft, plan.Status)
	})

	t.Run("Should get the plan and see the update persisted", func(t *testing.T) {
		require.NotEmpty(t, planID)

		status, plan, problem := c.GetPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plan)

		require.Len(t, plan.Phases, 1)
		assert.Equal(t, "Phase Renamed", plan.Phases[0].Name)
		assert.Len(t, plan.Phases[0].RateCards, 2)
	})

	t.Run("Should publish the plan", func(t *testing.T) {
		require.NotEmpty(t, planID)

		status, plan, problem := c.PublishPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plan)

		assert.Equal(t, apiv3.BillingPlanStatusActive, plan.Status)
		assert.NotNil(t, plan.EffectiveFrom)
		assert.Nil(t, plan.EffectiveTo)
	})

	t.Run("Should archive the published plan", func(t *testing.T) {
		require.NotEmpty(t, planID)

		status, plan, problem := c.ArchivePlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plan)

		assert.Equal(t, apiv3.BillingPlanStatusArchived, plan.Status)
		assert.NotNil(t, plan.EffectiveTo)
	})

	t.Run("Should delete an archived plan", func(t *testing.T) {
		require.NotEmpty(t, planID)

		status, _ := c.DeletePlan(planID)
		assert.Equal(t, http.StatusNoContent, status)
	})

	t.Run("Should return deleted_at after deletion", func(t *testing.T) {
		require.NotEmpty(t, planID)

		status, plan, problem := c.GetPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
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
		body.Phases = []apiv3.BillingPlanPhase{}

		status, _, problem := c.CreatePlan(body)
		require.Equal(t, http.StatusBadRequest, status)
		assertInvalidParameterRule(t, problem, "min_items")
	})

	t.Run("duplicate phase key → 400 plan_phase_duplicated_key", func(t *testing.T) {
		c := newV3Client(t)

		body := validPlanRequest("phase_validation_dup")
		p1 := validPlanPhase("same_key", false /* isLast */)
		p2 := validPlanPhase("ignored", true /* isLast */)
		p2.Key = p1.Key
		body.Phases = []apiv3.BillingPlanPhase{p1, p2}

		status, _, problem := c.CreatePlan(body)
		require.Equal(t, http.StatusBadRequest, status)
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
	phase.RateCards = []apiv3.BillingRateCard{}

	body := validPlanRequest("invalid_draft_lifecycle")
	body.Phases = []apiv3.BillingPlanPhase{phase}

	var planID string

	t.Run("create accepts the invalid draft", func(t *testing.T) {
		status, plan, problem := c.CreatePlan(body)
		require.Equal(t, http.StatusCreated, status, "create should accept an invalid draft: %+v", problem)
		require.NotNil(t, plan)
		planID = plan.Id
	})

	t.Run("validation_errors surfaces on GET", func(t *testing.T) {
		require.NotEmpty(t, planID)
		status, got, problem := c.GetPlan(planID)
		require.Equal(t, http.StatusOK, status, "%+v", problem)
		require.NotNil(t, got)
		require.NotNil(t, got.ValidationErrors, "expected validation_errors on the draft")

		var codes []string
		for _, e := range *got.ValidationErrors {
			codes = append(codes, e.Code)
		}
		assert.Contains(t, codes, "plan_phase_has_no_rate_cards")
	})

	t.Run("publish rejects with the same code", func(t *testing.T) {
		require.NotEmpty(t, planID)
		status, _, problem := c.PublishPlan(planID)
		require.Equal(t, http.StatusBadRequest, status)
		assertValidationCode(t, problem, "plan_phase_has_no_rate_cards")
	})

	t.Run("fix by adding a rate card to the phase", func(t *testing.T) {
		require.NotEmpty(t, planID)
		update := apiv3.UpsertPlanRequest{
			Name: body.Name,
			Phases: []apiv3.BillingPlanPhase{{
				Key:       phase.Key,
				Name:      phase.Name,
				Duration:  phase.Duration,
				RateCards: []apiv3.BillingRateCard{validFlatRateCard("added_rc")},
			}},
		}
		status, _, problem := c.UpdatePlan(planID, update)
		require.Equal(t, http.StatusOK, status, "%+v", problem)
	})

	t.Run("publish succeeds after fix", func(t *testing.T) {
		require.NotEmpty(t, planID)
		status, published, problem := c.PublishPlan(planID)
		require.Equal(t, http.StatusOK, status, "%+v", problem)
		require.NotNil(t, published)
		assert.Equal(t, apiv3.BillingPlanStatusActive, published.Status)
	})
}

// A second draft with the same key while v1 is still draft is rejected.
// Only one draft per key may exist at a time.
func TestV3PlanDuplicateDraftKeyRejected(t *testing.T) {
	c := newV3Client(t)

	first := validPlanRequest("duplicate_draft")
	key := first.Key

	status, plan, problem := c.CreatePlan(first)
	require.Equal(t, http.StatusCreated, status, "first create: %+v", problem)
	require.NotNil(t, plan)
	require.Equal(t, apiv3.BillingPlanStatusDraft, plan.Status)

	// Second POST with the same key, while v1 is still draft.
	second := validPlanRequest("duplicate_draft_ignored")
	second.Key = key

	status, _, problem = c.CreatePlan(second)
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusConflict}, status,
		"expected 400 or 409 for duplicate draft key, got %d: %+v", status, problem)
	require.NotNil(t, problem)
}

// Delete-while-active is rejected; after archive, delete is accepted.
func TestV3PlanDeleteWhileActiveRejected(t *testing.T) {
	c := newV3Client(t)

	status, plan, problem := c.CreatePlan(validPlanRequest("delete_while_active"))
	require.Equal(t, http.StatusCreated, status, "%+v", problem)
	require.NotNil(t, plan)

	status, _, problem = c.PublishPlan(plan.Id)
	require.Equal(t, http.StatusOK, status, "%+v", problem)

	status, problem = c.DeletePlan(plan.Id)
	assert.Equal(t, http.StatusBadRequest, status, "delete-while-active should reject: %+v", problem)

	status, _, problem = c.ArchivePlan(plan.Id)
	require.Equal(t, http.StatusOK, status, "archive: %+v", problem)

	status, problem = c.DeletePlan(plan.Id)
	assert.Equal(t, http.StatusNoContent, status, "delete-after-archive should succeed: %+v", problem)
}

// Update after publish is rejected.
func TestV3PlanUpdateAfterPublishRejected(t *testing.T) {
	c := newV3Client(t)

	createBody := validPlanRequest("update_after_publish")

	status, plan, problem := c.CreatePlan(createBody)
	require.Equal(t, http.StatusCreated, status, "%+v", problem)
	require.NotNil(t, plan)

	status, _, problem = c.PublishPlan(plan.Id)
	require.Equal(t, http.StatusOK, status, "%+v", problem)

	// Attempt to rename a phase after publish.
	renamed := createBody.Phases[0]
	renamed.Name = "Renamed After Publish"

	update := apiv3.UpsertPlanRequest{
		Name:   createBody.Name,
		Phases: []apiv3.BillingPlanPhase{renamed},
	}

	status, _, problem = c.UpdatePlan(plan.Id, update)
	assert.Equal(t, http.StatusBadRequest, status)
	assertProblemDetail(t, problem, "only Plans in [draft scheduled] can be updated")
}

// Plan versioning and auto-archive. Create v1, publish it, create a new plan
// with the same key, assert v2 is auto-assigned. Publish v2 and assert v1 is
// auto-archived with EffectiveTo == v2.EffectiveFrom (transactional auto-archive).
func TestV3PlanVersioningAndAutoArchive(t *testing.T) {
	c := newV3Client(t)

	createBody := validPlanRequest("versioning")
	sharedKey := createBody.Key

	status, v1, problem := c.CreatePlan(createBody)
	require.Equal(t, http.StatusCreated, status, "create v1: %+v", problem)
	require.NotNil(t, v1)
	assert.Equal(t, 1, v1.Version)

	status, v1Active, problem := c.PublishPlan(v1.Id)
	require.Equal(t, http.StatusOK, status, "publish v1: %+v", problem)
	require.NotNil(t, v1Active)
	require.Equal(t, apiv3.BillingPlanStatusActive, v1Active.Status)

	// v2: same key, different internal fixtures.
	v2Body := validPlanRequest("versioning_v2")
	v2Body.Key = sharedKey

	status, v2, problem := c.CreatePlan(v2Body)
	require.Equal(t, http.StatusCreated, status, "create v2: %+v", problem)
	require.NotNil(t, v2)
	assert.Equal(t, 2, v2.Version)
	assert.Equal(t, apiv3.BillingPlanStatusDraft, v2.Status)
	assert.Equal(t, sharedKey, v2.Key)
	assert.NotEqual(t, v1.Id, v2.Id, "v1 and v2 must have distinct IDs")

	status, v2Active, problem := c.PublishPlan(v2.Id)
	require.Equal(t, http.StatusOK, status, "publish v2: %+v", problem)
	require.NotNil(t, v2Active)
	require.Equal(t, apiv3.BillingPlanStatusActive, v2Active.Status)
	require.NotNil(t, v2Active.EffectiveFrom)

	// v1 should now be auto-archived with EffectiveTo == v2.EffectiveFrom.
	status, v1After, problem := c.GetPlan(v1.Id)
	require.Equal(t, http.StatusOK, status, "get v1: %+v", problem)
	require.NotNil(t, v1After)
	assert.Equal(t, apiv3.BillingPlanStatusArchived, v1After.Status)
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
		mutate func(*apiv3.CreatePlanRequest)
		code   string
	}{
		{
			name: "non-last phase missing duration → plan_has_non_last_phase_with_no_duration",
			mutate: func(r *apiv3.CreatePlanRequest) {
				nonLast := validPlanPhase("non_last", true /* isLast — no duration */)
				last := validPlanPhase("last", true /* isLast */)
				r.Phases = []apiv3.BillingPlanPhase{nonLast, last}
			},
			code: "plan_has_non_last_phase_with_no_duration",
		},
		{
			name: "last phase with a duration → plan_has_last_phase_with_duration",
			mutate: func(r *apiv3.CreatePlanRequest) {
				first := validPlanPhase("first", false /* isLast */)
				last := validPlanPhase("last", false /* bounded — should be open */)
				r.Phases = []apiv3.BillingPlanPhase{first, last}
			},
			code: "plan_has_last_phase_with_duration",
		},
		{
			name: "phase with zero rate cards → plan_phase_has_no_rate_cards",
			mutate: func(r *apiv3.CreatePlanRequest) {
				phase := validPlanPhase("empty_rc", true /* isLast */)
				phase.RateCards = []apiv3.BillingRateCard{}
				r.Phases = []apiv3.BillingPlanPhase{phase}
			},
			code: "plan_phase_has_no_rate_cards",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newV3Client(t)

			body := validPlanRequest("publish_validation")
			tc.mutate(&body)

			status, plan, problem := c.CreatePlan(body)
			require.Equal(t, http.StatusCreated, status, "create should accept an invalid draft: %+v", problem)
			require.NotNil(t, plan)

			status, _, problem = c.PublishPlan(plan.Id)
			assert.Equal(t, http.StatusBadRequest, status)
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

	status, _, problem := c.CreatePlan(body)
	assert.Equal(t, http.StatusBadRequest, status, "problem: %+v", problem)
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

		status, plan, problem := v3.GetPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plan)

		rc := findRateCardByKey(t, plan, dynamicRCKey)

		assertUnitPriceAmount(t, rc, "1")

		require.NotNil(t, rc.UnitConfig, "expected synthesized unit_config")
		assert.Equal(t, apiv3.BillingUnitConfigOperationMultiply, rc.UnitConfig.Operation)
		assert.Equal(t, apiv3.Numeric("1.2"), rc.UnitConfig.ConversionFactor)
		assert.Nil(t, rc.UnitConfig.Rounding, "dynamic translation does not set rounding")

		require.NotNil(t, rc.Commitments, "v1 commitments should round-trip via v3")
		assert.Equal(t, lo.ToPtr(apiv3.Numeric("10")), rc.Commitments.MinimumAmount)
		assert.Equal(t, lo.ToPtr(apiv3.Numeric("100")), rc.Commitments.MaximumAmount)
	})

	t.Run("v3 GET should translate package price to unit + divide+ceiling unit_config", func(t *testing.T) {
		require.NotEmpty(t, planID)

		status, plan, problem := v3.GetPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plan)

		rc := findRateCardByKey(t, plan, packageRCKey)

		assertUnitPriceAmount(t, rc, "0.5")

		require.NotNil(t, rc.UnitConfig, "expected synthesized unit_config")
		assert.Equal(t, apiv3.BillingUnitConfigOperationDivide, rc.UnitConfig.Operation)
		assert.Equal(t, apiv3.Numeric("1000"), rc.UnitConfig.ConversionFactor)
		require.NotNil(t, rc.UnitConfig.Rounding, "package translation must set rounding=ceiling")
		assert.Equal(t, apiv3.BillingUnitConfigRoundingModeCeiling, *rc.UnitConfig.Rounding)

		require.NotNil(t, rc.Commitments, "v1 commitments should round-trip via v3")
		assert.Equal(t, lo.ToPtr(apiv3.Numeric("5")), rc.Commitments.MinimumAmount)
		assert.Nil(t, rc.Commitments.MaximumAmount)
	})

	t.Run("v3 LIST should include the plan with both rate cards translated", func(t *testing.T) {
		require.NotEmpty(t, planID)

		// Bump page size so a fresh fixture isn't pushed off page 1 on a shared DB.
		status, page, problem := v3.ListPlans(withPageSize(1000))
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, page)

		var found *apiv3.BillingPlan
		for i := range page.Data {
			if page.Data[i].Id == planID {
				found = &page.Data[i]
				break
			}
		}
		require.NotNil(t, found, "created plan not in list response (the v3 list handler should no longer skip plans with v1 dynamic/package prices)")

		dynRC := findRateCardByKey(t, found, dynamicRCKey)
		assertUnitPriceAmount(t, dynRC, "1")
		require.NotNil(t, dynRC.UnitConfig)
		assert.Equal(t, apiv3.BillingUnitConfigOperationMultiply, dynRC.UnitConfig.Operation)
		assert.Equal(t, apiv3.Numeric("1.2"), dynRC.UnitConfig.ConversionFactor)

		pkgRC := findRateCardByKey(t, found, packageRCKey)
		assertUnitPriceAmount(t, pkgRC, "0.5")
		require.NotNil(t, pkgRC.UnitConfig)
		assert.Equal(t, apiv3.BillingUnitConfigOperationDivide, pkgRC.UnitConfig.Operation)
		assert.Equal(t, apiv3.Numeric("1000"), pkgRC.UnitConfig.ConversionFactor)
		require.NotNil(t, pkgRC.UnitConfig.Rounding)
		assert.Equal(t, apiv3.BillingUnitConfigRoundingModeCeiling, *pkgRC.UnitConfig.Rounding)
	})
}
