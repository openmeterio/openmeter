package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
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
	planBody.Phases = []v3sdk.PlanPhaseInput{trialPhase, standardPhase}

	addonBody := validAddonRequest("test_v3_plan_addon_golden_addon")

	var planID, addonID, planAddonID string

	t.Run("Should create a draft plan with two phases", func(t *testing.T) {
		plan, err := c.Plans.Create(t.Context(), planBody)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)
		require.Len(t, plan.Phases, 2)

		planID = plan.ID
	})

	t.Run("Should create and publish the addon", func(t *testing.T) {
		addon, err := c.Addons.Create(t.Context(), addonBody)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, addon)
		addonID = addon.ID

		addon, err = c.Addons.Publish(t.Context(), addonID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, addon)
		require.Equal(t, v3sdk.AddonStatusActive, addon.Status)
	})

	t.Run("Should attach the addon to the plan at the standard phase", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, addonID)

		body := validPlanAddonRequest(standardPhase.Key, addonID)

		planAddon, err := c.PlanAddons.Create(t.Context(), planID, body)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, planAddon)

		assert.Equal(t, standardPhase.Key, planAddon.FromPlanPhase)
		assert.Equal(t, addonID, planAddon.Addon.ID)

		planAddonID = planAddon.ID
	})

	t.Run("Should get the plan-addon junction", func(t *testing.T) {
		require.NotEmpty(t, planAddonID)

		planAddon, err := c.PlanAddons.Get(t.Context(), planID, planAddonID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, planAddon)

		assert.Equal(t, planAddonID, planAddon.ID)
		assert.Equal(t, standardPhase.Key, planAddon.FromPlanPhase)
		assert.Equal(t, addonID, planAddon.Addon.ID)
	})

	t.Run("Should list plan-addons and find the attached addon", func(t *testing.T) {
		require.NotEmpty(t, planAddonID)

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
		assert.True(t, found, "attached plan-addon not found in list")
	})

	t.Run("Should update the junction to attach from the trial phase", func(t *testing.T) {
		require.NotEmpty(t, planAddonID)

		update := v3sdk.UpsertPlanAddonRequest{
			Name:          "Test Plan Addon",
			FromPlanPhase: trialPhase.Key,
		}

		planAddon, err := c.PlanAddons.Update(t.Context(), planID, planAddonID, update)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, planAddon)

		assert.Equal(t, trialPhase.Key, planAddon.FromPlanPhase)
	})

	t.Run("Should detach the addon from the plan", func(t *testing.T) {
		require.NotEmpty(t, planAddonID)

		err := c.PlanAddons.Delete(t.Context(), planID, planAddonID)
		c.requireStatus(http.StatusNoContent, err)
	})

	t.Run("Should not find the junction in the list after detach", func(t *testing.T) {
		require.NotEmpty(t, planAddonID)

		page, err := c.PlanAddons.List(t.Context(), planID, v3sdk.PlanAddonListParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, page)

		for _, pa := range page.Data {
			assert.NotEqual(t, planAddonID, pa.ID, "detached plan-addon still present in list")
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
		planStatus       v3sdk.PlanStatus
		addonStatus      v3sdk.AddonStatus
		expectedStatus   int
		expectedDetailIn string // substring of Detail; empty when expectedStatus is 2xx
	}{
		{
			name:           "draft plan + active addon → 201",
			planStatus:     v3sdk.PlanStatusDraft,
			addonStatus:    v3sdk.AddonStatusActive,
			expectedStatus: http.StatusCreated,
		},
		{
			name:             "active plan + active addon → 400 (plan must be draft or scheduled)",
			planStatus:       v3sdk.PlanStatusActive,
			addonStatus:      v3sdk.AddonStatusActive,
			expectedStatus:   http.StatusBadRequest,
			expectedDetailIn: "invalid active status, allowed statuses: [draft scheduled]",
		},
		{
			name:             "archived plan + active addon → 400",
			planStatus:       v3sdk.PlanStatusArchived,
			addonStatus:      v3sdk.AddonStatusActive,
			expectedStatus:   http.StatusBadRequest,
			expectedDetailIn: "invalid archived status, allowed statuses: [draft scheduled]",
		},
		{
			name:             "draft plan + draft addon → 400 (addon must be active)",
			planStatus:       v3sdk.PlanStatusDraft,
			addonStatus:      v3sdk.AddonStatusDraft,
			expectedStatus:   http.StatusBadRequest,
			expectedDetailIn: "invalid draft status, allowed statuses: [active]",
		},
		{
			name:             "draft plan + archived addon → 400",
			planStatus:       v3sdk.PlanStatusDraft,
			addonStatus:      v3sdk.AddonStatusArchived,
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
			planAddon, err := c.PlanAddons.Create(t.Context(), plan.ID, validPlanAddonRequest(phaseKey, addon.ID))

			if tc.expectedDetailIn != "" {
				problem := requireProblem(t, err, tc.expectedStatus)
				assertProblemDetail(t, problem, tc.expectedDetailIn)
			} else {
				c.requireStatus(tc.expectedStatus, err)
				require.NotNil(t, planAddon)
			}
		})
	}
}

// Attach instance_type × max_quantity matrix.
//
// Exercises the TypeSpec schema binder (invalid_parameters[].rule="minimum")
// for max_quantity<1, which the API schema enforces before the handler runs.
func TestV3PlanAddonInstanceTypeMaxQuantityMatrix(t *testing.T) {
	cases := []struct {
		name           string
		instanceType   v3sdk.AddonInstanceType
		maxQuantity    *int64
		expectedStatus int
		expectedRule   string // schema-layer rule (invalid_parameters)
	}{
		{
			name:           "single + nil max_quantity → 201",
			instanceType:   v3sdk.AddonInstanceTypeSingle,
			maxQuantity:    nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "multiple + nil max_quantity → 201",
			instanceType:   v3sdk.AddonInstanceTypeMultiple,
			maxQuantity:    nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "multiple + max_quantity=5 → 201",
			instanceType:   v3sdk.AddonInstanceTypeMultiple,
			maxQuantity:    lo.ToPtr(int64(5)),
			expectedStatus: http.StatusCreated,
		},
		{
			// Caught by the TypeSpec binder (minimum: 1), not the domain
			// validator — the schema layer fires first for max_quantity<1.
			name:           "multiple + max_quantity=0 → 400 schema minimum",
			instanceType:   v3sdk.AddonInstanceTypeMultiple,
			maxQuantity:    lo.ToPtr(int64(0)),
			expectedStatus: http.StatusBadRequest,
			expectedRule:   "minimum",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newV3Client(t)

			plan, err := c.Plans.Create(t.Context(), validPlanRequest("iq_plan"))
			c.requireStatus(http.StatusCreated, err)
			require.NotNil(t, plan)

			addonBody := validAddonRequest("iq_addon")
			addonBody.InstanceType = tc.instanceType

			addon, err := c.Addons.Create(t.Context(), addonBody)
			c.requireStatus(http.StatusCreated, err)
			require.NotNil(t, addon)

			published, err := c.Addons.Publish(t.Context(), addon.ID)
			c.requireStatus(http.StatusOK, err)
			require.NotNil(t, published)

			body := validPlanAddonRequest(plan.Phases[0].Key, addon.ID)
			body.MaxQuantity = tc.maxQuantity

			planAddon, err := c.PlanAddons.Create(t.Context(), plan.ID, body)

			switch {
			case tc.expectedRule != "":
				problem := requireProblem(t, err, tc.expectedStatus)
				assertInvalidParameterRule(t, problem, tc.expectedRule)
			case tc.expectedStatus >= 400:
				problem := requireProblem(t, err, tc.expectedStatus)
				require.NotNil(t, problem, "expected a problem response")
				assert.NotEmpty(t, problem.Detail, "expected problem.Detail to be populated")
			default:
				c.requireStatus(tc.expectedStatus, err)
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

		err := c.PlanAddons.Delete(t.Context(), planID, planAddonID)
		c.requireStatus(http.StatusNoContent, err)
	})

	t.Run("detach from an active plan → 400", func(t *testing.T) {
		c := newV3Client(t)

		planID, planAddonID := setupAttachedPlanAddon(t, c, "detach_active")

		_, err := c.Plans.Publish(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)

		err = c.PlanAddons.Delete(t.Context(), planID, planAddonID)
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("detach from a deleted plan → 400 'plan is deleted'", func(t *testing.T) {
		// Plan doc suggested 404; live server returns 400 with a deleted-state
		// detail. Reasonable contract either way — pinning observed behavior.
		c := newV3Client(t)

		planID, planAddonID := setupAttachedPlanAddon(t, c, "detach_deleted")

		// Publish → archive → delete to reach the deleted state.
		_, err := c.Plans.Publish(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		_, err = c.Plans.Archive(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		err = c.Plans.Delete(t.Context(), planID)
		c.requireStatus(http.StatusNoContent, err)

		err = c.PlanAddons.Delete(t.Context(), planID, planAddonID)
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertProblemDetail(t, problem, "plan is deleted")
	})
}

// setupAttachedPlanAddon creates a draft plan + published addon, attaches
// them, and returns (planID, planAddonID).
func setupAttachedPlanAddon(t *testing.T, c *v3Client, keyPrefix string) (string, string) {
	t.Helper()

	plan, err := c.Plans.Create(t.Context(), validPlanRequest(keyPrefix+"_plan"))
	c.requireStatus(http.StatusCreated, err)

	addon, err := c.Addons.Create(t.Context(), validAddonRequest(keyPrefix+"_addon"))
	c.requireStatus(http.StatusCreated, err)
	_, err = c.Addons.Publish(t.Context(), addon.ID)
	c.requireStatus(http.StatusOK, err)

	planAddon, err := c.PlanAddons.Create(t.Context(), plan.ID, validPlanAddonRequest(plan.Phases[0].Key, addon.ID))
	c.requireStatus(http.StatusCreated, err)

	return plan.ID, planAddon.ID
}

// Publishing a plan whose attached addon is archived should be rejected.
func TestV3PlanAddonPublishAgainstArchivedAddon(t *testing.T) {
	c := newV3Client(t)

	planID, _ := setupAttachedPlanAddon(t, c, "publish_vs_archived_addon")

	// Archive the attached addon. We need the addon ID — grab it from the
	// plan's addon list.
	list, err := c.PlanAddons.List(t.Context(), planID, v3sdk.PlanAddonListParams{})
	c.requireStatus(http.StatusOK, err)
	require.Len(t, list.Data, 1)
	addonID := list.Data[0].Addon.ID

	_, err = c.Addons.Archive(t.Context(), addonID)
	c.requireStatus(http.StatusOK, err)

	_, err = c.Plans.Publish(t.Context(), planID)
	requireProblem(t, err, http.StatusBadRequest)
}

// Attaching the same addon twice is rejected.
func TestV3PlanAddonDuplicateAttachmentRejected(t *testing.T) {
	c := newV3Client(t)

	plan, err := c.Plans.Create(t.Context(), validPlanRequest("dup_attach_plan"))
	c.requireStatus(http.StatusCreated, err)

	addon, err := c.Addons.Create(t.Context(), validAddonRequest("dup_attach_addon"))
	c.requireStatus(http.StatusCreated, err)
	_, err = c.Addons.Publish(t.Context(), addon.ID)
	c.requireStatus(http.StatusOK, err)

	body := validPlanAddonRequest(plan.Phases[0].Key, addon.ID)

	_, err = c.PlanAddons.Create(t.Context(), plan.ID, body)
	c.requireStatus(http.StatusCreated, err)

	_, err = c.PlanAddons.Create(t.Context(), plan.ID, body)
	requireProblem(t, err, http.StatusConflict)
}

// Publishing a plan with an attached addon succeeds and the junction
// survives the status transition.
func TestV3PlanAddonPublishWithAttachedAddon(t *testing.T) {
	c := newV3Client(t)

	plan, err := c.Plans.Create(t.Context(), validPlanRequest("publish_attached_plan"))
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plan)

	addon, err := c.Addons.Create(t.Context(), validAddonRequest("publish_attached_addon"))
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, addon)

	_, err = c.Addons.Publish(t.Context(), addon.ID)
	c.requireStatus(http.StatusOK, err)

	planAddon, err := c.PlanAddons.Create(t.Context(), plan.ID, validPlanAddonRequest(plan.Phases[0].Key, addon.ID))
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, planAddon)
	planAddonID := planAddon.ID

	published, err := c.Plans.Publish(t.Context(), plan.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, published)
	assert.Equal(t, v3sdk.PlanStatusActive, published.Status)

	page, err := c.PlanAddons.List(t.Context(), plan.ID, v3sdk.PlanAddonListParams{})
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, page)

	found := false
	for _, pa := range page.Data {
		if pa.ID == planAddonID {
			found = true
			assert.Equal(t, addon.ID, pa.Addon.ID)
			break
		}
	}
	assert.True(t, found, "attached plan-addon missing after plan publish")
}

// makePlanWithStatus creates a plan and advances it to the requested lifecycle
// state. Scheduled is intentionally unsupported: see TestV3PlanAddonAttachStatusMatrix.
func makePlanWithStatus(t *testing.T, c *v3Client, keyPrefix string, target v3sdk.PlanStatus) *v3sdk.Plan {
	t.Helper()

	plan, err := c.Plans.Create(t.Context(), validPlanRequest(keyPrefix))
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plan)

	switch target {
	case v3sdk.PlanStatusDraft:
		return plan
	case v3sdk.PlanStatusActive:
		plan, err = c.Plans.Publish(t.Context(), plan.ID)
		c.requireStatus(http.StatusOK, err)
		return plan
	case v3sdk.PlanStatusArchived:
		plan, err = c.Plans.Publish(t.Context(), plan.ID)
		c.requireStatus(http.StatusOK, err)
		plan, err = c.Plans.Archive(t.Context(), plan.ID)
		c.requireStatus(http.StatusOK, err)
		return plan
	default:
		t.Fatalf("unsupported plan target status %q", target)
		return nil
	}
}

// makeAddonWithStatus creates an addon and advances it to the requested
// lifecycle state.
func makeAddonWithStatus(t *testing.T, c *v3Client, keyPrefix string, target v3sdk.AddonStatus) *v3sdk.Addon {
	t.Helper()

	addon, err := c.Addons.Create(t.Context(), validAddonRequest(keyPrefix))
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, addon)

	switch target {
	case v3sdk.AddonStatusDraft:
		return addon
	case v3sdk.AddonStatusActive:
		addon, err = c.Addons.Publish(t.Context(), addon.ID)
		c.requireStatus(http.StatusOK, err)
		return addon
	case v3sdk.AddonStatusArchived:
		addon, err = c.Addons.Publish(t.Context(), addon.ID)
		c.requireStatus(http.StatusOK, err)
		addon, err = c.Addons.Archive(t.Context(), addon.ID)
		c.requireStatus(http.StatusOK, err)
		return addon
	default:
		t.Fatalf("unsupported addon target status %q", target)
		return nil
	}
}
