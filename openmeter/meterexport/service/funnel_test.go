package meterexportservice

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TestFunnelStaysOnRawMeterQueries pins the meter export funnel as a never-cached meter
// consumer: exports feed external billing/analytics pipelines, so every windowed query
// the funnel issues must have Cachable=false (the zero value) — only the read-only call
// sites opt in (see streaming.QueryParams.Cachable).
func TestFunnelStaysOnRawMeterQueries(t *testing.T) {
	m := meter.Meter{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: "test-ns"},
		},
		Key:         "meter-1",
		EventType:   "api-calls",
		Aggregation: meter.MeterAggregationCount,
	}

	from := time.Now().Add(-3 * time.Hour).Truncate(time.Hour)
	to := from.Add(2 * time.Hour)

	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
	streamingConnector.AddSimpleEvent(m.Key, 10, from.Add(time.Minute))

	svc := &service{Config: Config{StreamingConnector: streamingConnector}}

	resultCh := make(chan meter.MeterQueryRow, 1024)
	errCh := make(chan error, 16)

	err := svc.funnel(t.Context(), funnelParams{
		meter: m,
		queryParams: streaming.QueryParams{
			From:       &from,
			To:         &to,
			WindowSize: lo.ToPtr(meter.WindowSizeHour),
		},
	}, resultCh, errCh)
	require.NoError(t, err)

	for err := range errCh {
		require.NoError(t, err)
	}

	captured := streamingConnector.CapturedQueryMeterParams()
	require.NotEmpty(t, captured)

	for _, params := range captured {
		require.False(t, params.Cachable)
	}
}
