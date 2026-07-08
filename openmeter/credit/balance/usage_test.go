package balance_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// TestUsageQuerierOptsIntoMeterCache pins the credit usage querier as a designated meter
// cache opt-in call site: every usage read it issues must hand Cachable=true to the
// streaming connector — including both legs of the UNIQUE_COUNT subtraction, whose
// staleness would be billing-visible through grant burn-down.
func TestUsageQuerierOptsIntoMeterCache(t *testing.T) {
	newQuerier := func(t *testing.T, m meter.Meter, periodStart time.Time) (balance.UsageQuerier, *streamingtestutils.MockStreamingConnector) {
		t.Helper()

		streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
		streamingConnector.AddSimpleEvent(m.Key, 10, periodStart.Add(time.Minute))

		querier := balance.NewUsageQuerier(balance.UsageQuerierConfig{
			StreamingConnector: streamingConnector,
			DescribeOwner: func(ctx context.Context, id models.NamespacedID) (grant.Owner, error) {
				return grant.Owner{
					NamespacedID: id,
					Meter:        m,
				}, nil
			},
			GetDefaultParams: func(ctx context.Context, ownerID models.NamespacedID) (streaming.QueryParams, error) {
				return streaming.QueryParams{}, nil
			},
			GetUsagePeriodStartAt: func(ctx context.Context, ownerID models.NamespacedID, at time.Time) (time.Time, error) {
				return periodStart, nil
			},
		})

		return querier, streamingConnector
	}

	ownerID := models.NamespacedID{Namespace: "test-ns", ID: "owner-1"}
	periodStart := time.Now().Add(-4 * time.Hour).Truncate(time.Minute)

	t.Run("sum single query", func(t *testing.T) {
		querier, streamingConnector := newQuerier(t, meter.Meter{
			Key:           "meter-1",
			EventType:     "api-calls",
			Aggregation:   meter.MeterAggregationSum,
			ValueProperty: lo.ToPtr("$.value"),
		}, periodStart)

		_, err := querier.QueryUsage(t.Context(), ownerID, timeutil.ClosedPeriod{
			From: periodStart,
			To:   periodStart.Add(3 * time.Hour),
		})
		require.NoError(t, err)

		captured := streamingConnector.CapturedQueryMeterParams()
		require.Len(t, captured, 1)
		require.True(t, captured[0].Cachable)
	})

	t.Run("unique count subtraction queries", func(t *testing.T) {
		querier, streamingConnector := newQuerier(t, meter.Meter{
			Key:           "meter-1",
			EventType:     "api-calls",
			Aggregation:   meter.MeterAggregationUniqueCount,
			ValueProperty: lo.ToPtr("$.value"),
		}, periodStart)

		// A period starting after the usage period start forces both subtraction legs:
		// [periodStart, period.To) and [periodStart, period.From).
		_, err := querier.QueryUsage(t.Context(), ownerID, timeutil.ClosedPeriod{
			From: periodStart.Add(time.Hour),
			To:   periodStart.Add(3 * time.Hour),
		})
		require.NoError(t, err)

		captured := streamingConnector.CapturedQueryMeterParams()
		require.Len(t, captured, 2)
		require.True(t, captured[0].Cachable)
		require.True(t, captured[1].Cachable)
	})
}
