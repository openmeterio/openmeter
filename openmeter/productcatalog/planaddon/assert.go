package planaddon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func AssertPlanAddonCreateInputEqual(t *testing.T, i CreatePlanAddonInput, a PlanAddon) {
	t.Helper()

	assert.Equalf(t, i.Namespace, a.Namespace, "create input: namespace mismatch")
	assert.Equalf(t, i.Metadata, a.Metadata, "metadata mismatch")
	assert.Equalf(t, i.Annotations, a.Annotations, "annotations mismatch")
	assert.Equalf(t, i.PlanID, a.Plan.ID, "plan id mismatch")
	assert.Equalf(t, i.AddonID, a.Addon.ID, "add-on id mismatch")
	assert.Equalf(t, i.FromPlanPhase, a.FromPlanPhase, "plan phase key mismatch")

	if i.MaxQuantity != nil {
		assert.Equalf(t, *i.MaxQuantity, *a.MaxQuantity, "max quantity mismatch")
	}
}

func AssertPlanAddonUpdateInputEqual(t *testing.T, i UpdatePlanAddonInput, a PlanAddon) {
	t.Helper()

	assert.Equalf(t, i.Namespace, a.Namespace, "update input: namespace mismatch")

	if i.Metadata != nil {
		assert.Equalf(t, *i.Metadata, a.Metadata, "metadata mismatch")
	}

	if i.Annotations != nil {
		assert.Equalf(t, *i.Annotations, a.Annotations, "annotations mismatch")
	}

	if i.ID != "" {
		assert.Equalf(t, i.ID, a.ID, "id mismatch")
	}

	if i.PlanID != "" {
		assert.Equalf(t, i.PlanID, a.Plan.ID, "plan id mismatch")
	}

	if i.AddonID != "" {
		assert.Equalf(t, i.AddonID, a.Addon.ID, "add-on id mismatch")
	}

	if i.FromPlanPhase != nil {
		assert.Equalf(t, *i.FromPlanPhase, a.FromPlanPhase, "plan phase key mismatch")
	}

	if i.MaxQuantity != nil {
		assert.Equalf(t, *i.MaxQuantity, *a.MaxQuantity, "max quality mismatch")
	}
}

func AssertPlanAddonEqual(t *testing.T, expected, actual PlanAddon) {
	t.Helper()

	assert.Equalf(t, expected.ID, actual.ID, "id mismatch")
	assert.Equalf(t, expected.FromPlanPhase, actual.FromPlanPhase, "plan phase key mismatch")
	assert.Equalf(t, expected.Metadata, actual.Metadata, "metadata mismatch")
	assert.Equalf(t, expected.Annotations, actual.Annotations, "annotations mismatch")
}
