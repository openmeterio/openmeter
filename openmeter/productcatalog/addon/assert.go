package addon

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

func AssertAddonCreateInputEqual(t *testing.T, i CreateAddonInput, a Addon) {
	t.Helper()

	assert.Equalf(t, i.Namespace, a.Namespace, "create input: namespace mismatch")
	assert.Equalf(t, i.Key, a.Key, "create input: key mismatch")
	assert.Equalf(t, i.Name, a.Name, "create input: name mismatch")
	assert.Equalf(t, i.Description, a.Description, "create input: description mismatch")
	assert.Equalf(t, i.Currency, a.Currency, "create input: currency mismatch")
	assert.Equalf(t, i.Metadata, a.Metadata, "metadata mismatch")
	assert.Equalf(t, i.Annotations, a.Annotations, "annotations mismatch")

	AssertAddonRateCardsEqual(t, i.RateCards, a.RateCards)
}

func AssertAddonUpdateInputEqual(t *testing.T, i UpdateAddonInput, a Addon) {
	t.Helper()

	assert.Equalf(t, i.Namespace, a.Namespace, "update input: namespace mismatch")

	if i.Name != nil {
		assert.Equalf(t, *i.Name, a.Name, "update input: name mismatch")
	}

	if i.Description != nil {
		assert.Equalf(t, lo.FromPtrOr(i.Description, ""), lo.FromPtrOr(a.Description, ""), "update input: description mismatch")
	}

	if i.Metadata != nil {
		assert.Equalf(t, *i.Metadata, a.Metadata, "metadata mismatch")
	}

	if i.Annotations != nil {
		assert.Equalf(t, *i.Annotations, a.Annotations, "annotations mismatch")
	}

	if i.RateCards != nil {
		AssertAddonRateCardsEqual(t, *i.RateCards, a.RateCards)
	}
}

func AssertAddonEqual(t *testing.T, expected, actual Addon) {
	t.Helper()

	assert.Equalf(t, expected.Key, actual.Key, "key mismatch")
	assert.Equalf(t, expected.Name, actual.Name, "name mismatch")
	assert.Equalf(t, expected.Description, actual.Description, "description mismatch")
	assert.Equalf(t, expected.Currency, actual.Currency, "currency mismatch")
	assert.Equalf(t, expected.Metadata, actual.Metadata, "metadata mismatch")
	assert.Equalf(t, expected.Annotations, actual.Annotations, "annotations mismatch")

	AssertAddonRateCardsEqual(t, expected.RateCards, actual.RateCards)
}

func AssertAddonRateCardsEqual(t *testing.T, r1, r2 productcatalog.RateCards) {
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

// getBillingCadence extracts billing cadence from different RateCard types
func getBillingCadence(r productcatalog.RateCard) isodate.Period {
	switch vv := r.(type) {
	case *productcatalog.FlatFeeRateCard:
		return lo.FromPtr(vv.BillingCadence)
	case *FlatFeeRateCard:
		return lo.FromPtr(vv.FlatFeeRateCard.BillingCadence)
	case *productcatalog.UsageBasedRateCard:
		return vv.BillingCadence
	case *UsageBasedRateCard:
		return vv.UsageBasedRateCard.BillingCadence
	}
	return isodate.Period{}
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

	f1 := lo.FromPtr(m1.Feature)
	f2 := lo.FromPtr(m2.Feature)

	assert.Equalf(t, f1.Key, f2.Key, "feature key mismatch")
	assert.Equalf(t, f1.Namespace, f2.Namespace, "feature namespace mismatch")

	assert.Truef(t, m1.EntitlementTemplate.Equal(m2.EntitlementTemplate), "entitlement template mismatch")

	assert.Truef(t, m1.TaxConfig.Equal(m2.TaxConfig), "tax config mismatch")

	assert.Truef(t, m1.Price.Equal(m2.Price), "price mismatch")

	billingCadence1 := getBillingCadence(r1)
	billingCadence2 := getBillingCadence(r2)

	assert.Equal(t, billingCadence1, billingCadence2)
}
