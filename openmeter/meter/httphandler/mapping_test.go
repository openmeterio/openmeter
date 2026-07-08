package httpdriver

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
)

// TestToQueryParamsFromRequestOptsIntoMeterCache pins the v1 meter query endpoints as a
// designated meter cache opt-in call site: every v1 query (GET, POST and CSV all funnel
// through toQueryParamsFromRequest) must hand Cachable=true to the streaming connector,
// while billing paths construct their own params and must never set it.
func TestToQueryParamsFromRequestOptsIntoMeterCache(t *testing.T) {
	h := &handler{}

	params, err := h.toQueryParamsFromRequest(t.Context(), meter.Meter{}, api.QueryMeterPostJSONRequestBody{})
	require.NoError(t, err)
	require.True(t, params.Cachable)
}
