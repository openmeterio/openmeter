package plan

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func AssertPlanCreateInputEqual(t *testing.T, i CreatePlanInput, p Plan) {
	t.Helper()

	assert.Equalf(t, i.Namespace, p.Namespace, "create input: namespace mismatch")
	assert.Equalf(t, i.Key, p.Key, "create input: key mismatch")
	assert.Equalf(t, i.Name, p.Name, "create input: name mismatch")
	assert.Equalf(t, i.Description, p.Description, "create input: description mismatch")
	assert.Equalf(t, i.Currency, p.Currency, "create input: currency mismatch")
	assert.Equalf(t, i.Metadata, p.Metadata, "metadata mismatch")

	AssertPlanPhasesEqual(t, i.Phases, p.Phases)
}

func AssertPlanUpdateInputEqual(t *testing.T, i UpdatePlanInput, p Plan) {
	t.Helper()

	assert.Equalf(t, i.Namespace, p.Namespace, "update input: namespace mismatch")

	if i.Name != nil {
		assert.Equalf(t, *i.Name, p.Name, "update input: name mismatch")
	}

	if i.Description != nil {
		assert.Equalf(t, lo.FromPtr(i.Description), lo.FromPtr(p.Description), "update input: description mismatch")
	}

	if i.Metadata != nil {
		assert.Equalf(t, *i.Metadata, p.Metadata, "metadata mismatch")
	}

	if i.Phases != nil {
		AssertPlanPhasesEqual(t, *i.Phases, p.Phases)
	}
}

func AssertPlanEqual(t *testing.T, expected, actual Plan) {
	t.Helper()

	assert.Equal(t, expected.Key, actual.Key)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.Currency, actual.Currency)

	AssertPlanPhasesEqual(t, expected.Phases, actual.Phases)
}

func AssertPlanPhasesEqual[E interface{ productcatalog.Phase | Phase }](t *testing.T, expected []E, actual []Phase) {
	t.Helper()

	assert.Equalf(t, len(expected), len(actual), "number of PlanPhases mismatch")

	expectedMap := func() map[string]E {
		m := make(map[string]E, len(expected))
		for _, v := range expected {
			var meta productcatalog.PhaseMeta
			switch vv := any(v).(type) {
			case productcatalog.Phase:
				meta = vv.PhaseMeta
			case Phase:
				meta = vv.PhaseMeta
			}

			require.NotEmptyf(t, meta.Key, "Phase key must not be empty")

			m[meta.Key] = v
		}

		return m
	}()

	actualMap := func() map[string]Phase {
		m := make(map[string]Phase, len(actual))
		for _, v := range actual {
			m[v.Key] = v
		}

		return m
	}()

	actualVisited := make(map[string]struct{})
	for key, expectedPhase := range expectedMap {
		actualPhase, ok := actualMap[key]
		require.Truef(t, ok, "missing PlanPhase key")

		AssertPlanPhaseEqual(t, expectedPhase, actualPhase)

		actualVisited[key] = struct{}{}
	}

	for key := range actualMap {
		_, ok := actualVisited[key]
		require.Truef(t, ok, "missing PlanPhase key")
	}
}

func AssertPlanPhaseEqual[E interface{ productcatalog.Phase | Phase }](t *testing.T, in E, actual Phase) {
	t.Helper()

	if managed, ok := any(in).(ManagedPhase); ok {
		expectedManagedFields := managed.ManagedFields()
		assert.Equalf(t, expectedManagedFields.PlanID, actual.PlanID, "planId mismatch")
		assert.Equalf(t, expectedManagedFields.ID, actual.ID, "id mismatch")
		assert.Equalf(t, expectedManagedFields.Namespace, actual.Namespace, "namespace mismatch")
	}

	var expected productcatalog.Phase
	switch v := any(in).(type) {
	case productcatalog.Phase:
		expected = v
	case Phase:
		expected = v.Phase
	}

	assert.Equalf(t, expected.Key, actual.Key, "key mismatch")
	assert.Equalf(t, expected.Name, actual.Name, "name mismatch")
	assert.Equalf(t, expected.Description, actual.Description, "description mismatch")
	assert.Equalf(t, expected.Metadata, actual.Metadata, "metadata mismatch")
	assert.Equalf(t, expected.Duration, actual.Duration, "duration mismatch")

	AssertPlanRateCardsEqual(t, expected.RateCards, actual.RateCards)
}

func AssertPlanRateCardsEqual(t *testing.T, r1, r2 productcatalog.RateCards) {
	t.Helper()

	assert.Equalf(t, len(r1), len(r2), "number of RateCards mismatch")

	r1Map := func() map[string]productcatalog.RateCard {
		m := make(map[string]productcatalog.RateCard, len(r1))
		for _, v := range r1 {
			m[v.Key()] = v
		}

		return m
	}()

	r2Map := func() map[string]productcatalog.RateCard {
		m := make(map[string]productcatalog.RateCard, len(r2))
		for _, v := range r2 {
			m[v.Key()] = v
		}

		return m
	}()

	visited := make(map[string]struct{})
	for phase1Key, rateCard1 := range r1Map {
		rateCard2, ok := r2Map[phase1Key]
		require.Truef(t, ok, "missing RateCard key")

		AssertRateCardEqual(t, rateCard1, rateCard2)

		visited[phase1Key] = struct{}{}
	}

	for phase2Key := range r2Map {
		_, ok := visited[phase2Key]
		require.Truef(t, ok, "missing RateCard key")
	}
}

func AssertRateCardEqual(t *testing.T, r1, r2 productcatalog.RateCard) {
	t.Helper()

	assert.Equalf(t, r1.Type(), r2.Type(), "type mismatch")

	m1 := r1.AsMeta()
	m2 := r2.AsMeta()

	assert.Equalf(t, m1.Key, m2.Key, "key mismatch")
	assert.Equalf(t, m1.Name, m2.Name, "name mismatch")
	assert.Equalf(t, lo.FromPtr(m1.Description), lo.FromPtr(m2.Description), "description mismatch")

	assert.Truef(t, m1.Metadata.Equal(m2.Metadata), "metadata mismatch")

	assert.Equalf(t, m1.FeatureKey, m2.FeatureKey, "feature key mismatch")
	assert.Equalf(t, m1.FeatureID, m2.FeatureID, "feature id mismatch")

	assert.Truef(t, m1.EntitlementTemplate.Equal(m2.EntitlementTemplate), "entitlement template mismatch")

	assert.Truef(t, m1.TaxConfig.Equal(m2.TaxConfig), "tax config mismatch")

	assert.Truef(t, m1.Price.Equal(m2.Price), "price mismatch")

	billingCadence1 := r1.GetBillingCadence().ISOStringPtrOrNil()
	billingCadence2 := r2.GetBillingCadence().ISOStringPtrOrNil()

	assert.Equal(t, billingCadence1, billingCadence2)
}
