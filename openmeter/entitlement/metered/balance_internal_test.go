package meteredentitlement

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
)

// TestQueryMeterOptsIntoMeterCache pins the entitlement balance queryMeter wrapper as a
// designated meter cache opt-in call site: every balance/history usage read going through
// it must hand Cachable=true to the streaming connector. The zero-length-period shortcut
// must keep answering locally without touching the connector at all.
func TestQueryMeterOptsIntoMeterCache(t *testing.T) {
	m := meter.Meter{
		Key:           "meter-1",
		EventType:     "api-calls",
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
	streamingConnector.AddSimpleEvent(m.Key, 10, time.Now().Add(-time.Hour))

	e := &connector{streamingConnector: streamingConnector}

	from := time.Now().Add(-2 * time.Hour)
	to := time.Now()

	_, err := e.queryMeter(t.Context(), "test-ns", m, streaming.QueryParams{From: &from, To: &to})
	require.NoError(t, err)

	captured := streamingConnector.CapturedQueryMeterParams()
	require.Len(t, captured, 1)
	require.True(t, captured[0].Cachable)

	// Zero-length period: answered locally, no connector call recorded.
	rows, err := e.queryMeter(t.Context(), "test-ns", m, streaming.QueryParams{From: &from, To: &from})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, float64(0), rows[0].Value)
	require.Len(t, streamingConnector.CapturedQueryMeterParams(), 1)
}
