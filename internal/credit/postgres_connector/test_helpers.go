package postgres_connector

import (
	"context"
	"testing"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/stretchr/testify/assert"
)

//lint:ignore U1000 https://github.com/dominikh/go-tools/issues/633
var th = testHelpers{}

type testHelpers struct{}

//lint:ignore U1000 https://github.com/dominikh/go-tools/issues/633
func (th testHelpers) createFeature(t *testing.T, connector credit.Connector, feature credit.Feature) credit.Feature {
	ctx := context.Background()
	p, err := connector.CreateFeature(ctx, feature)
	if err != nil {
		t.Error(err)
	}
	return p
}

//lint:ignore U1000 https://github.com/dominikh/go-tools/issues/633
func (th testHelpers) removeTimestampsFromGrant(g credit.Grant) credit.Grant {
	g.CreatedAt = nil
	g.UpdatedAt = nil
	return g
}

//lint:ignore U1000 https://github.com/dominikh/go-tools/issues/633
func (th testHelpers) removeTimestampsFromGrants(gs []credit.Grant) []credit.Grant {
	for i := range gs {
		gs[i] = th.removeTimestampsFromGrant(gs[i])
	}
	return gs
}

//lint:ignore U1000 https://github.com/dominikh/go-tools/issues/633
func (th testHelpers) assertGrantsEqual(t *testing.T, expected, actual credit.Grant) {
	assert.Equal(t, th.removeTimestampsFromGrant(expected), th.removeTimestampsFromGrant(actual))
}
