package adapter

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
)

func assertPhaseCreateInputEqual(t *testing.T, i createPhaseInput, p plan.Phase) {
	t.Helper()

	assert.Equalf(t, i.Namespace, p.Namespace, "create input: namespace mismatch")
	assert.Equalf(t, i.Key, p.Key, "create input: key mismatch")
	assert.Equalf(t, i.Name, p.Name, "create input: name mismatch")
	assert.Equalf(t, i.Description, p.Description, "create input: description mismatch")
	assert.Equalf(t, i.Metadata, p.Metadata, "create input: metadata mismatch")
	assert.Equalf(t, i.StartAfter.ISOString(), p.StartAfter.ISOString(), "create input: startAfter mismatch")

	plan.AssertPlanRateCardsEqual(t, i.RateCards, p.RateCards)
}

func assertPhaseUpdateInputEqual(t *testing.T, i updatePhaseInput, p plan.Phase) {
	t.Helper()

	assert.Equalf(t, i.Namespace, p.Namespace, "update input: namespace mismatch")

	assert.Equalf(t, i.Key, p.Key, "update input: key mismatch")

	if i.Name != nil {
		assert.Equalf(t, *i.Name, p.Name, "update input: name mismatch")
	}

	if i.Description != nil {
		assert.Equalf(t, lo.FromPtrOr(i.Description, ""), lo.FromPtrOr(p.Description, ""), "update input: description mismatch")
	}

	if i.Metadata != nil {
		assert.Equalf(t, *i.Metadata, p.Metadata, "update input: metadata mismatch")
	}

	if i.StartAfter != nil {
		assert.Equalf(t, *i.StartAfter, p.StartAfter, "update input: startAfter mismatch")
	}

	assert.Equalf(t, i.PlanID, p.PlanID, "update input: planID mismatch")

	if i.RateCards != nil {
		plan.AssertPlanRateCardsEqual(t, *i.RateCards, p.RateCards)
	}
}
