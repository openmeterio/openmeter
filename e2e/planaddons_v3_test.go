package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

// Plan-addon attach lifecycle: build a draft plan with two phases + a published
// addon, attach, read, list, update from_plan_phase, detach, list again.
func TestV3PlanAddonAttachLifecycle(t *testing.T) {
	c := newV3Client(t)

	// Plan with two phases (trial + standard). The trial phase is bounded;
	// the last phase ("standard") must not carry a duration.
	trialPhase := validPlanPhase("trial", false /* isLast */)
	standardPhase := validPlanPhase("standard", true /* isLast */)

	planBody := validPlanRequest("test_v3_plan_addon_golden")
	planBody.Phases = []apiv3.BillingPlanPhase{trialPhase, standardPhase}

	addonBody := validAddonRequest("test_v3_plan_addon_golden_addon")

	var planID, addonID, planAddonID string

	t.Run("Should create a draft plan with two phases", func(t *testing.T) {
		status, plan, problem := c.CreatePlan(planBody)
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, plan)
		require.Len(t, plan.Phases, 2)

		planID = plan.Id
	})

	t.Run("Should create and publish the addon", func(t *testing.T) {
		status, addon, problem := c.CreateAddon(addonBody)
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, addon)
		addonID = addon.Id

		status, addon, problem = c.PublishAddon(addonID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, addon)
		require.Equal(t, apiv3.AddonStatusActive, addon.Status)
	})

	t.Run("Should attach the addon to the plan at the standard phase", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, addonID)

		body := validPlanAddonRequest(standardPhase.Key, addonID)

		status, planAddon, problem := c.AttachAddon(planID, body)
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, planAddon)

		assert.Equal(t, standardPhase.Key, planAddon.FromPlanPhase)
		assert.Equal(t, addonID, planAddon.Addon.Id)

		planAddonID = planAddon.Id
	})

	t.Run("Should get the plan-addon junction", func(t *testing.T) {
		require.NotEmpty(t, planAddonID)

		status, planAddon, problem := c.GetPlanAddon(planID, planAddonID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, planAddon)

		assert.Equal(t, planAddonID, planAddon.Id)
		assert.Equal(t, standardPhase.Key, planAddon.FromPlanPhase)
		assert.Equal(t, addonID, planAddon.Addon.Id)
	})

	t.Run("Should list plan-addons and find the attached addon", func(t *testing.T) {
		require.NotEmpty(t, planAddonID)

		status, page, problem := c.ListPlanAddons(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, page)

		found := false
		for _, pa := range page.Data {
			if pa.Id == planAddonID {
				found = true
				break
			}
		}
		assert.True(t, found, "attached plan-addon not found in list")
	})

	t.Run("Should update the junction to attach from the trial phase", func(t *testing.T) {
		require.NotEmpty(t, planAddonID)

		update := apiv3.UpsertPlanAddonRequest{
			Name:          "Test Plan Addon",
			FromPlanPhase: trialPhase.Key,
		}

		status, planAddon, problem := c.UpdatePlanAddon(planID, planAddonID, update)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, planAddon)

		assert.Equal(t, trialPhase.Key, planAddon.FromPlanPhase)
	})

	t.Run("Should detach the addon from the plan", func(t *testing.T) {
		require.NotEmpty(t, planAddonID)

		status, _ := c.DetachAddon(planID, planAddonID)
		assert.Equal(t, http.StatusNoContent, status)
	})

	t.Run("Should not find the junction in the list after detach", func(t *testing.T) {
		require.NotEmpty(t, planAddonID)

		status, page, problem := c.ListPlanAddons(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, page)

		for _, pa := range page.Data {
			assert.NotEqual(t, planAddonID, pa.Id, "detached plan-addon still present in list")
		}
	})
}

// Plan-addon attach status matrix.
//
// Notes on coverage:
//   - The "scheduled" plan row is omitted because the v3 publish endpoint
//     takes no body and hardcodes EffectiveFrom = clock.Now() — there is no
//     way to drive a plan to "scheduled" through the public API.
//   - Attaching an addon mutates the plan, and only draft/scheduled plans
//     accept mutation, so "active plan + active addon" is a 400, not a 201.
//
// Status-mismatch rejections come back as a plain BaseAPIError with the
// reason embedded in Detail, not as a ProductCatalogValidationError under
// extensions.validationErrors.
func TestV3PlanAddonAttachStatusMatrix(t *testing.T) {
	cases := []struct {
		name             string
		planStatus       apiv3.BillingPlanStatus
		addonStatus      apiv3.AddonStatus
		expectedStatus   int
		expectedDetailIn string // substring of Detail; empty when expectedStatus is 2xx
	}{
		{
			name:           "draft plan + active addon → 201",
			planStatus:     apiv3.BillingPlanStatusDraft,
			addonStatus:    apiv3.AddonStatusActive,
			expectedStatus: http.StatusCreated,
		},
		{
			name:             "active plan + active addon → 400 (plan must be draft or scheduled)",
			planStatus:       apiv3.BillingPlanStatusActive,
			addonStatus:      apiv3.AddonStatusActive,
			expectedStatus:   http.StatusBadRequest,
			expectedDetailIn: "invalid active status, allowed statuses: [draft scheduled]",
		},
		{
			name:             "archived plan + active addon → 400",
			planStatus:       apiv3.BillingPlanStatusArchived,
			addonStatus:      apiv3.AddonStatusActive,
			expectedStatus:   http.StatusBadRequest,
			expectedDetailIn: "invalid archived status, allowed statuses: [draft scheduled]",
		},
		{
			name:             "draft plan + draft addon → 400 (addon must be active)",
			planStatus:       apiv3.BillingPlanStatusDraft,
			addonStatus:      apiv3.AddonStatusDraft,
			expectedStatus:   http.StatusBadRequest,
			expectedDetailIn: "invalid draft status, allowed statuses: [active]",
		},
		{
			name:             "draft plan + archived addon → 400",
			planStatus:       apiv3.BillingPlanStatusDraft,
			addonStatus:      apiv3.AddonStatusArchived,
			expectedStatus:   http.StatusBadRequest,
			expectedDetailIn: "invalid archived status, allowed statuses: [active]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newV3Client(t)

			plan := makePlanWithStatus(t, c, "matrix_plan", tc.planStatus)
			addon := makeAddonWithStatus(t, c, "matrix_addon", tc.addonStatus)

			phaseKey := plan.Phases[0].Key
			status, planAddon, problem := c.AttachAddon(plan.Id, validPlanAddonRequest(phaseKey, addon.Id))

			assert.Equal(t, tc.expectedStatus, status, "problem: %+v", problem)

			if tc.expectedDetailIn != "" {
				assertProblemDetail(t, problem, tc.expectedDetailIn)
			} else {
				require.NotNil(t, planAddon)
			}
		})
	}
}

// Attach instance_type × max_quantity matrix.
//
// Two failure layers are exercised here:
//   - Domain validation (extensions.validationErrors[].code) for cases that
//     pass the schema binder but fail PlanAddon.Validate (e.g. nil for multi).
//   - TypeSpec schema binder (invalid_parameters[].rule="minimum") for
//     max_quantity<1, which the API schema enforces before the handler runs.
func TestV3PlanAddonInstanceTypeMaxQuantityMatrix(t *testing.T) {
	cases := []struct {
		name           string
		instanceType   apiv3.AddonInstanceType
		maxQuantity    *int
		expectedStatus int
		expectedCode   string // product-catalog validation code (extensions.validationErrors)
		expectedRule   string // schema-layer rule (invalid_parameters), mutually exclusive with expectedCode
	}{
		{
			name:           "single + nil max_quantity → 201",
			instanceType:   apiv3.AddonInstanceTypeSingle,
			maxQuantity:    nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "single + max_quantity=5 → 400 plan_addon_max_quantity_must_not_be_set",
			instanceType:   apiv3.AddonInstanceTypeSingle,
			maxQuantity:    lo.ToPtr(5),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "plan_addon_max_quantity_must_not_be_set",
		},
		{
			name:           "multiple + nil max_quantity → 400 plan_addon_max_quantity_must_be_set",
			instanceType:   apiv3.AddonInstanceTypeMultiple,
			maxQuantity:    nil,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "plan_addon_max_quantity_must_be_set",
		},
		{
			name:           "multiple + max_quantity=5 → 201",
			instanceType:   apiv3.AddonInstanceTypeMultiple,
			maxQuantity:    lo.ToPtr(5),
			expectedStatus: http.StatusCreated,
		},
		{
			// Caught by the TypeSpec binder (minimum: 1), not the domain
			// validator — the schema layer fires first for max_quantity<1.
			name:           "multiple + max_quantity=0 → 400 schema minimum",
			instanceType:   apiv3.AddonInstanceTypeMultiple,
			maxQuantity:    lo.ToPtr(0),
			expectedStatus: http.StatusBadRequest,
			expectedRule:   "minimum",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newV3Client(t)

			planStatus, plan, problem := c.CreatePlan(validPlanRequest("iq_plan"))
			require.Equal(t, http.StatusCreated, planStatus, "create plan: %+v", problem)
			require.NotNil(t, plan)

			addonBody := validAddonRequest("iq_addon")
			addonBody.InstanceType = tc.instanceType

			addonStatus, addon, problem := c.CreateAddon(addonBody)
			require.Equal(t, http.StatusCreated, addonStatus, "create addon: %+v", problem)
			require.NotNil(t, addon)

			pubStatus, published, problem := c.PublishAddon(addon.Id)
			require.Equal(t, http.StatusOK, pubStatus, "publish addon: %+v", problem)
			require.NotNil(t, published)

			body := validPlanAddonRequest(plan.Phases[0].Key, addon.Id)
			body.MaxQuantity = tc.maxQuantity

			status, planAddon, problem := c.AttachAddon(plan.Id, body)
			assert.Equal(t, tc.expectedStatus, status, "attach response: %+v", problem)

			switch {
			case tc.expectedCode != "":
				assertValidationCode(t, problem, tc.expectedCode)
			case tc.expectedRule != "":
				assertInvalidParameterRule(t, problem, tc.expectedRule)
			case tc.expectedStatus >= 400:
				require.NotNil(t, problem, "expected a problem response")
				assert.NotEmpty(t, problem.Detail, "expected problem.Detail to be populated")
			default:
				require.NotNil(t, planAddon)
			}
		})
	}
}

// Detach rules by plan status. The "scheduled plan" row is omitted because
// the v3 publish endpoint hardcodes EffectiveFrom = clock.Now(), so a
// scheduled plan is unreachable via the public API.
func TestV3PlanAddonDetachRulesByPlanStatus(t *testing.T) {
	t.Run("detach from a draft plan → 204", func(t *testing.T) {
		c := newV3Client(t)

		planID, planAddonID := setupAttachedPlanAddon(t, c, "detach_draft")

		status, problem := c.DetachAddon(planID, planAddonID)
		assert.Equal(t, http.StatusNoContent, status, "%+v", problem)
	})

	t.Run("detach from an active plan → 400", func(t *testing.T) {
		c := newV3Client(t)

		planID, planAddonID := setupAttachedPlanAddon(t, c, "detach_active")

		status, _, problem := c.PublishPlan(planID)
		require.Equal(t, http.StatusOK, status, "publish plan: %+v", problem)

		status, problem = c.DetachAddon(planID, planAddonID)
		assert.Equal(t, http.StatusBadRequest, status, "detach from active should reject: %+v", problem)
	})

	t.Run("detach from a deleted plan → 400 'plan is deleted'", func(t *testing.T) {
		// Plan doc suggested 404; live server returns 400 with a deleted-state
		// detail. Reasonable contract either way — pinning observed behavior.
		c := newV3Client(t)

		planID, planAddonID := setupAttachedPlanAddon(t, c, "detach_deleted")

		// Publish → archive → delete to reach the deleted state.
		status, _, problem := c.PublishPlan(planID)
		require.Equal(t, http.StatusOK, status, "publish: %+v", problem)
		status, _, problem = c.ArchivePlan(planID)
		require.Equal(t, http.StatusOK, status, "archive: %+v", problem)
		status, problem = c.DeletePlan(planID)
		require.Equal(t, http.StatusNoContent, status, "delete: %+v", problem)

		status, problem = c.DetachAddon(planID, planAddonID)
		assert.Equal(t, http.StatusBadRequest, status, "detach from deleted: %+v", problem)
		assertProblemDetail(t, problem, "plan is deleted")
	})
}

// setupAttachedPlanAddon creates a draft plan + published addon, attaches
// them, and returns (planID, planAddonID).
func setupAttachedPlanAddon(t *testing.T, c *v3Client, keyPrefix string) (string, string) {
	t.Helper()

	status, plan, problem := c.CreatePlan(validPlanRequest(keyPrefix + "_plan"))
	require.Equal(t, http.StatusCreated, status, "create plan: %+v", problem)

	status, addon, problem := c.CreateAddon(validAddonRequest(keyPrefix + "_addon"))
	require.Equal(t, http.StatusCreated, status, "create addon: %+v", problem)
	status, _, problem = c.PublishAddon(addon.Id)
	require.Equal(t, http.StatusOK, status, "publish addon: %+v", problem)

	status, planAddon, problem := c.AttachAddon(plan.Id, validPlanAddonRequest(plan.Phases[0].Key, addon.Id))
	require.Equal(t, http.StatusCreated, status, "attach: %+v", problem)

	return plan.Id, planAddon.Id
}

// Publishing a plan whose attached addon is archived should be rejected.
func TestV3PlanAddonPublishAgainstArchivedAddon(t *testing.T) {
	c := newV3Client(t)

	planID, _ := setupAttachedPlanAddon(t, c, "publish_vs_archived_addon")

	// Archive the attached addon. We need the addon ID — grab it from the
	// plan's addon list.
	status, list, problem := c.ListPlanAddons(planID)
	require.Equal(t, http.StatusOK, status, "%+v", problem)
	require.Len(t, list.Data, 1)
	addonID := list.Data[0].Addon.Id

	status, _, problem = c.ArchiveAddon(addonID)
	require.Equal(t, http.StatusOK, status, "archive addon: %+v", problem)

	status, _, problem = c.PublishPlan(planID)
	assert.Equal(t, http.StatusBadRequest, status, "publish against archived addon should reject: %+v", problem)
	require.NotNil(t, problem)
}

// Attaching the same addon twice is rejected.
func TestV3PlanAddonDuplicateAttachmentRejected(t *testing.T) {
	c := newV3Client(t)

	status, plan, problem := c.CreatePlan(validPlanRequest("dup_attach_plan"))
	require.Equal(t, http.StatusCreated, status, "%+v", problem)

	status, addon, problem := c.CreateAddon(validAddonRequest("dup_attach_addon"))
	require.Equal(t, http.StatusCreated, status, "%+v", problem)
	status, _, problem = c.PublishAddon(addon.Id)
	require.Equal(t, http.StatusOK, status, "%+v", problem)

	body := validPlanAddonRequest(plan.Phases[0].Key, addon.Id)

	status, _, problem = c.AttachAddon(plan.Id, body)
	require.Equal(t, http.StatusCreated, status, "first attach: %+v", problem)

	status, _, problem = c.AttachAddon(plan.Id, body)
	assert.Equal(t, http.StatusConflict, status, "duplicate attach should be rejected: %+v", problem)
	require.NotNil(t, problem)
}

// Publishing a plan with an attached addon succeeds and the junction
// survives the status transition.
func TestV3PlanAddonPublishWithAttachedAddon(t *testing.T) {
	c := newV3Client(t)

	status, plan, problem := c.CreatePlan(validPlanRequest("publish_attached_plan"))
	require.Equal(t, http.StatusCreated, status, "create plan: %+v", problem)
	require.NotNil(t, plan)

	status, addon, problem := c.CreateAddon(validAddonRequest("publish_attached_addon"))
	require.Equal(t, http.StatusCreated, status, "create addon: %+v", problem)
	require.NotNil(t, addon)

	status, _, problem = c.PublishAddon(addon.Id)
	require.Equal(t, http.StatusOK, status, "publish addon: %+v", problem)

	status, planAddon, problem := c.AttachAddon(plan.Id, validPlanAddonRequest(plan.Phases[0].Key, addon.Id))
	require.Equal(t, http.StatusCreated, status, "attach: %+v", problem)
	require.NotNil(t, planAddon)
	planAddonID := planAddon.Id

	status, published, problem := c.PublishPlan(plan.Id)
	require.Equal(t, http.StatusOK, status, "publish plan: %+v", problem)
	require.NotNil(t, published)
	assert.Equal(t, apiv3.BillingPlanStatusActive, published.Status)

	status, page, problem := c.ListPlanAddons(plan.Id)
	require.Equal(t, http.StatusOK, status, "list plan addons: %+v", problem)
	require.NotNil(t, page)

	found := false
	for _, pa := range page.Data {
		if pa.Id == planAddonID {
			found = true
			assert.Equal(t, addon.Id, pa.Addon.Id)
			break
		}
	}
	assert.True(t, found, "attached plan-addon missing after plan publish")
}

// makePlanWithStatus creates a plan and advances it to the requested lifecycle
// state. Scheduled is intentionally unsupported: see TestV3PlanAddonAttachStatusMatrix.
func makePlanWithStatus(t *testing.T, c *v3Client, keyPrefix string, target apiv3.BillingPlanStatus) *apiv3.BillingPlan {
	t.Helper()

	status, plan, problem := c.CreatePlan(validPlanRequest(keyPrefix))
	require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
	require.NotNil(t, plan)

	switch target {
	case apiv3.BillingPlanStatusDraft:
		return plan
	case apiv3.BillingPlanStatusActive:
		status, plan, problem = c.PublishPlan(plan.Id)
		require.Equal(t, http.StatusOK, status, "publish plan: %+v", problem)
		return plan
	case apiv3.BillingPlanStatusArchived:
		status, plan, problem = c.PublishPlan(plan.Id)
		require.Equal(t, http.StatusOK, status, "publish plan: %+v", problem)
		status, plan, problem = c.ArchivePlan(plan.Id)
		require.Equal(t, http.StatusOK, status, "archive plan: %+v", problem)
		return plan
	default:
		t.Fatalf("unsupported plan target status %q", target)
		return nil
	}
}

// Attach with currency mismatch: plan in EUR, addon in USD
// → 400 with plan_addon_currency_mismatch.
func TestV3PlanAddonCurrencyMismatch(t *testing.T) {
	c := newV3Client(t)

	planBody := validPlanRequest("currency_mismatch_plan")
	planBody.Currency = "EUR"

	status, plan, problem := c.CreatePlan(planBody)
	require.Equal(t, http.StatusCreated, status, "create plan: %+v", problem)
	require.NotNil(t, plan)

	// validAddonRequest defaults to USD.
	status, addon, problem := c.CreateAddon(validAddonRequest("currency_mismatch_addon"))
	require.Equal(t, http.StatusCreated, status, "create addon: %+v", problem)
	require.NotNil(t, addon)

	status, _, problem = c.PublishAddon(addon.Id)
	require.Equal(t, http.StatusOK, status, "publish addon: %+v", problem)

	status, _, problem = c.AttachAddon(plan.Id, validPlanAddonRequest(plan.Phases[0].Key, addon.Id))
	assert.Equal(t, http.StatusBadRequest, status, "attach response: %+v", problem)
	assertValidationCode(t, problem, "plan_addon_currency_mismatch")
}

// Attach with rate-card cadence unalignment.
//
// The domain alignment rule (openmeter/productcatalog/alignment.go) treats two
// cadences as aligned when one divides the other without remainder, so
// plan=P1M + addon=P1Y is aligned (12÷1=12) and won't trigger the check. To
// force an unalignment we pick cadences where neither divides the other:
// plan=P3M + addon=P2M (3÷2 and 2÷3 both fail).
//
// The error raised is `rate_card_billing_cadence_unaligned`, not
// `rate_card_billing_cadence_mismatch` (the mismatch code is for a different
// path — addon-vs-plan-rate-card direct comparison).
func TestV3PlanAddonCadenceMismatch(t *testing.T) {
	c := newV3Client(t)

	// Plan at P3M.
	planBody := validPlanRequest("cadence_mismatch_plan")
	planBody.BillingCadence = apiv3.ISO8601Duration("P3M")
	// The plan's rate card inherits from validFlatRateCard which sets P1M —
	// P1M divides P3M, so the plan itself stays aligned.

	status, plan, problem := c.CreatePlan(planBody)
	require.Equal(t, http.StatusCreated, status, "create plan: %+v", problem)
	require.NotNil(t, plan)

	// Addon rate card at P2M — neither divides P3M nor is divisible by it.
	cadence := apiv3.ISO8601Duration("P2M")
	term := apiv3.BillingPricePaymentTermInAdvance
	price := apiv3.BillingPrice{}
	require.NoError(t, price.FromBillingPriceFlat(apiv3.BillingPriceFlat{
		Type:   apiv3.BillingPriceFlatTypeFlat,
		Amount: "10",
	}))

	p2mRateCard := apiv3.BillingRateCard{
		Key:            uniqueKey("p2m_rc"),
		Name:           "P2M Rate Card",
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
	}

	addonBody := validAddonRequest("cadence_mismatch_addon")
	addonBody.RateCards = []apiv3.BillingRateCard{p2mRateCard}

	status, addon, problem := c.CreateAddon(addonBody)
	require.Equal(t, http.StatusCreated, status, "create addon: %+v", problem)
	require.NotNil(t, addon)

	status, _, problem = c.PublishAddon(addon.Id)
	require.Equal(t, http.StatusOK, status, "publish addon: %+v", problem)

	status, _, problem = c.AttachAddon(plan.Id, validPlanAddonRequest(plan.Phases[0].Key, addon.Id))
	assert.Equal(t, http.StatusBadRequest, status, "attach response: %+v", problem)
	assertValidationCode(t, problem, "rate_card_billing_cadence_unaligned")
}

// Rate-card shape collision on attach: when a plan phase and an addon share
// a rate-card key but have incompatible price shapes (flat vs graduated),
// attach is rejected with 400.
//
// The underlying error (rate_card_price_type_mismatch) currently lacks an
// HTTP status code attribute, so the response surfaces as a plain 400 with
// the reason text in Detail rather than as extensions.validationErrors[].code.
// Pinning the specific code via assertValidationCode is a follow-up.
func TestV3PlanAddonRateCardShapeCollision(t *testing.T) {
	c := newV3Client(t)

	sharedKey := uniqueKey("setup_fee")

	planFlat := validFlatRateCard("collision_plan_flat")
	planFlat.Key = sharedKey

	phase := validPlanPhase("collision_phase", true /* isLast */)
	phase.RateCards = []apiv3.BillingRateCard{planFlat}

	planBody := validPlanRequest("shape_collision_plan")
	planBody.Phases = []apiv3.BillingPlanPhase{phase}

	status, plan, problem := c.CreatePlan(planBody)
	require.Equal(t, http.StatusCreated, status, "create plan: %+v", problem)
	require.NotNil(t, plan)

	addonGraduated := validGraduatedRateCard("collision_addon_graduated")
	addonGraduated.Key = sharedKey

	addonBody := validAddonRequest("shape_collision_addon")
	addonBody.RateCards = []apiv3.BillingRateCard{addonGraduated}

	status, addon, problem := c.CreateAddon(addonBody)
	require.Equal(t, http.StatusCreated, status, "create addon: %+v", problem)
	require.NotNil(t, addon)

	status, _, problem = c.PublishAddon(addon.Id)
	require.Equal(t, http.StatusOK, status, "publish addon: %+v", problem)

	status, _, problem = c.AttachAddon(plan.Id, validPlanAddonRequest(plan.Phases[0].Key, addon.Id))
	assert.Equal(t, http.StatusBadRequest, status, "attach response: %+v", problem)
	require.NotNil(t, problem, "expected problem response")
	assertProblemDetail(t, problem, "price type must match")
}

// makeAddonWithStatus creates an addon and advances it to the requested
// lifecycle state.
func makeAddonWithStatus(t *testing.T, c *v3Client, keyPrefix string, target apiv3.AddonStatus) *apiv3.Addon {
	t.Helper()

	status, addon, problem := c.CreateAddon(validAddonRequest(keyPrefix))
	require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
	require.NotNil(t, addon)

	switch target {
	case apiv3.AddonStatusDraft:
		return addon
	case apiv3.AddonStatusActive:
		status, addon, problem = c.PublishAddon(addon.Id)
		require.Equal(t, http.StatusOK, status, "publish addon: %+v", problem)
		return addon
	case apiv3.AddonStatusArchived:
		status, addon, problem = c.PublishAddon(addon.Id)
		require.Equal(t, http.StatusOK, status, "publish addon: %+v", problem)
		status, addon, problem = c.ArchiveAddon(addon.Id)
		require.Equal(t, http.StatusOK, status, "archive addon: %+v", problem)
		return addon
	default:
		t.Fatalf("unsupported addon target status %q", target)
		return nil
	}
}
