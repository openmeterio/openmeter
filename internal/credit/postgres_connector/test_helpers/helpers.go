package test_helpers

import (
	"context"
	"testing"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/stretchr/testify/assert"
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
