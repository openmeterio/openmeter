package plan

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/datex"
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
		assert.Equalf(t, lo.FromPtrOr(i.Description, ""), lo.FromPtrOr(p.Description, ""), "update input: description mismatch")
	}

	if i.Metadata != nil {
		assert.Equalf(t, *i.Metadata, p.Metadata, "metadata mismatch")
	}

	if i.Phases != nil {
		AssertPlanPhasesEqual(t, *i.Phases, p.Phases)
	}
}

func AssertPhaseCreateInputEqual(t *testing.T, i CreatePhaseInput, p Phase) {
	t.Helper()

	assert.Equalf(t, i.Namespace, p.Namespace, "create input: namespace mismatch")
	assert.Equalf(t, i.Key, p.Key, "create input: key mismatch")
	assert.Equalf(t, i.Name, p.Name, "create input: name mismatch")
	assert.Equalf(t, i.Description, p.Description, "create input: description mismatch")
	assert.Equalf(t, i.Metadata, p.Metadata, "create input: metadata mismatch")
	assert.Equalf(t, i.StartAfter.ISOString(), p.StartAfter.ISOString(), "create input: startAfter mismatch")

	AssertPlanRateCardsEqual(t, i.RateCards, p.RateCards)
}

func AssertPhaseUpdateInputEqual(t *testing.T, i UpdatePhaseInput, p Phase) {
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
		AssertPlanRateCardsEqual(t, *i.RateCards, p.RateCards)
	}
}

func AssertPlanEqual(t *testing.T, p1, p2 Plan) {
	t.Helper()

	assert.Equal(t, p1.Namespace, p2.Namespace)
	assert.Equal(t, p1.Key, p2.Key)
	assert.Equal(t, p1.Name, p2.Name)
	assert.Equal(t, p1.Description, p2.Description)
	assert.Equal(t, p1.Currency, p2.Currency)

	AssertPlanPhasesEqual(t, p1.Phases, p2.Phases)
}

func AssertPlanPhasesEqual(t *testing.T, p1, p2 []Phase) {
	t.Helper()

	assert.Equalf(t, len(p1), len(p2), "number of PlanPhases mismatch")

	p1Map := func() map[string]Phase {
		m := make(map[string]Phase, len(p1))
		for _, v := range p1 {
			m[v.Key] = v
		}

		return m
	}()

	p2Map := func() map[string]Phase {
		m := make(map[string]Phase, len(p2))
		for _, v := range p2 {
			m[v.Key] = v
		}

		return m
	}()

	visited := make(map[string]struct{})
	for phase1Key, phase1 := range p1Map {
		phase2, ok := p2Map[phase1Key]
		require.Truef(t, ok, "missing PlanPhase key")

		AssertPlanPhaseEqual(t, phase1, phase2)

		visited[phase1Key] = struct{}{}
	}

	for phase2Key := range p2Map {
		_, ok := visited[phase2Key]
		require.Truef(t, ok, "missing PlanPhase key")
	}
}

func AssertPlanPhaseEqual(t *testing.T, p1, p2 Phase) {
	t.Helper()

	assert.Equalf(t, p1.Key, p2.Key, "key mismatch")
	assert.Equalf(t, p1.Name, p2.Name, "name mismatch")
	assert.Equalf(t, p1.Description, p2.Description, "description mismatch")
	assert.Equalf(t, p1.Metadata, p2.Metadata, "metadata mismatch")
	assert.Equalf(t, p1.StartAfter, p2.StartAfter, "startAfter mismatch")

	AssertPlanRateCardsEqual(t, p1.RateCards, p2.RateCards)
}

func AssertPlanRateCardsEqual(t *testing.T, r1, r2 []RateCard) {
	t.Helper()

	assert.Equalf(t, len(r1), len(r2), "number of RateCards mismatch")

	r1Map := func() map[string]RateCard {
		m := make(map[string]RateCard, len(r1))
		for _, v := range r1 {
			m[v.Key()] = v
		}

		return m
	}()

	r2Map := func() map[string]RateCard {
		m := make(map[string]RateCard, len(r2))
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

func AssertRateCardEqual(t *testing.T, r1, r2 RateCard) {
	t.Helper()

	m1, err := r1.AsMeta()
	require.NoErrorf(t, err, "AsMeta must not fail")

	m2, err := r2.AsMeta()
	require.NoErrorf(t, err, "AsMeta must not fail")

	assert.Equalf(t, m1.Key, m2.Key, "key mismatch")
	assert.Equalf(t, m1.Type, m2.Type, "type mismatch")
	assert.Equalf(t, m1.Name, m2.Name, "name mismatch")
	assert.Equalf(t, lo.FromPtrOr(m1.Description, ""), lo.FromPtrOr(m2.Description, ""), "description mismatch")
	assert.Equalf(t, lo.FromPtrOr(m1.Description, ""), lo.FromPtrOr(m2.Description, ""), "description mismatch")

	assert.Truef(t, MetadataEqual(m1.Metadata, m2.Metadata), "metadata mismatch")

	f1 := lo.FromPtrOr(m1.Feature, feature.Feature{})
	f2 := lo.FromPtrOr(m2.Feature, feature.Feature{})

	assert.Equalf(t, f1.Key, f2.Key, "feature key mismatch")
	assert.Equalf(t, f1.ID, f2.ID, "feature id mismatch")
	assert.Equalf(t, f1.Namespace, f2.Namespace, "feature namespace mismatch")

	tmpl1, err := json.Marshal(m1.EntitlementTemplate)
	require.NoErrorf(t, err, "json marshal entitlement template must not fail")

	tmpl2, err := json.Marshal(m2.EntitlementTemplate)
	require.NoErrorf(t, err, "json marshal entitlement template must not fail")

	assert.Equalf(t, string(tmpl1), string(tmpl2), "entitlement template content mismatch")

	tax1, err := json.Marshal(m1.TaxConfig)
	require.NoErrorf(t, err, "json marshal tax config must not fail")

	tax2, err := json.Marshal(m2.TaxConfig)
	require.NoErrorf(t, err, "json marshal tax config must not fail")

	assert.Equalf(t, string(tax1), string(tax2), "tax config content mismatch")

	var billingCadence1, billingCadence2 datex.Period
	var taxConfig1, taxConfig2 []byte

	switch r1.Type() {
	case FlatFeeRateCardType:
		rc1, err := r1.AsFlatFee()
		require.NoError(t, err)

		rc2, err := r2.AsFlatFee()
		require.NoError(t, err)

		billingCadence1 = lo.FromPtrOr(rc1.BillingCadence, datex.Period{})
		billingCadence2 = lo.FromPtrOr(rc2.BillingCadence, datex.Period{})

		taxConfig1, err = json.Marshal(rc1.TaxConfig)
		require.NoError(t, err)

		taxConfig2, err = json.Marshal(rc2.TaxConfig)
		require.NoError(t, err)
	case UsageBasedRateCardType:
		rc1, err := r1.AsUsageBased()
		require.NoError(t, err)

		rc2, err := r2.AsUsageBased()
		require.NoError(t, err)

		billingCadence1 = rc1.BillingCadence
		billingCadence2 = rc2.BillingCadence

		taxConfig1, err = json.Marshal(rc1.TaxConfig)
		require.NoError(t, err)

		taxConfig2, err = json.Marshal(rc2.TaxConfig)
		require.NoError(t, err)
	}

	assert.Equal(t, billingCadence1, billingCadence2)
	assert.Equal(t, string(taxConfig1), string(taxConfig2))
}
