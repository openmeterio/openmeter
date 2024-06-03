package testutils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
)

func CreateFeature(t *testing.T, connector credit.Connector, feature credit.Feature) credit.Feature {
	ctx := context.Background()
	p, err := connector.CreateFeature(ctx, feature)
	if err != nil {
		t.Error(err)
	}
	return p
}

func RemoveTimestampsFromGrant(g credit.Grant) credit.Grant {
	g.CreatedAt = nil
	g.UpdatedAt = nil
	return g
}

func RemoveTimestampsFromGrants(gs []credit.Grant) []credit.Grant {
	for i := range gs {
		gs[i] = RemoveTimestampsFromGrant(gs[i])
	}
	return gs
}

func AssertGrantsEqual(t *testing.T, expected, actual credit.Grant) {
	assert.Equal(t, RemoveTimestampsFromGrant(expected), RemoveTimestampsFromGrant(actual))
}

func RemoveTimestampsFromBalance(balance credit.Balance) credit.Balance {
	balance.FeatureBalances = RemoveTimestampsFromFeatureBalances(balance.FeatureBalances)
	balance.GrantBalances = RemoveTimestampsFromGrantBalances(balance.GrantBalances)
	return balance
}

func RemoveTimestampsFromGrantBalances(grantBalances []credit.GrantBalance) []credit.GrantBalance {
	for i := range grantBalances {
		grantBalances[i].Grant.CreatedAt = nil
		grantBalances[i].Grant.UpdatedAt = nil
	}
	return grantBalances
}

func RemoveTimestampsFromFeatureBalances(featureBalances []credit.FeatureBalance) []credit.FeatureBalance {
	for i := range featureBalances {
		featureBalances[i].Feature.CreatedAt = nil
		featureBalances[i].Feature.UpdatedAt = nil
	}
	return featureBalances
}

func ToPtr[D any](s D) *D {
	return &s
}
