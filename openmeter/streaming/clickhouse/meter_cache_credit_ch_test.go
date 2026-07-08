package clickhouse

import (
	"context"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// TestCreditUniqueCountThroughCache drives the real credit usage querier
// (balance.NewUsageQuerier — the exact code grant burn-down uses) against a cache-enabled
// connector and proves, for the UNIQUE_COUNT aggregation whose inexactness would be
// billing-visible:
//
//   - the subtraction path (two cached-leg queries whose results are subtracted) matches
//     a live-forced run of the same subtraction exactly,
//   - a poison unique value bypassing BatchInsert stays invisible to the querier (the
//     proof its reads came from the cache leg, not a silent live fallback),
//   - a marker + refresh + wait converges the querier to the live answer.
func (s *MeterCacheCHTestSuite) TestCreditUniqueCountThroughCache() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-credit-uniq"
		eventType = "api-calls"
	)

	c := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	m := meter.Meter{
		Key:           "meter-uniq",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationUniqueCount,
		ValueProperty: lo.ToPtr("$.user"),
	}

	now := time.Now().UTC()
	periodStart := now.Add(-6 * time.Hour).Truncate(time.Hour)
	cachedBucket := now.Add(-4 * time.Hour).Truncate(time.Hour)

	// given:
	// - two unique users in a fully settled (cached) bucket, one of them repeated (uniq
	//   must not double count), and a third unique user in the always-live tail
	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", cachedBucket.Add(5*time.Minute), `{"user": "u1"}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-1", cachedBucket.Add(10*time.Minute), `{"user": "u2"}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-1", cachedBucket.Add(15*time.Minute), `{"user": "u1"}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-1", now.Add(-30*time.Minute), `{"user": "u3"}`, now),
	)

	mv := createMeterCacheMV{
		Database:        s.Database,
		EventsTableName: e2eEventsTable,
		Namespace:       namespace,
		Meter:           m,
		Grain:           CacheGrainHour,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
	}
	s.deployMeterCacheMV(ctx, mv)

	newUsageQuerier := func(connector streaming.Connector) balance.UsageQuerier {
		return balance.NewUsageQuerier(balance.UsageQuerierConfig{
			StreamingConnector: connector,
			DescribeOwner: func(ctx context.Context, id models.NamespacedID) (grant.Owner, error) {
				return grant.Owner{
					NamespacedID: id,
					Meter:        m,
				}, nil
			},
			GetDefaultParams: func(ctx context.Context, ownerID models.NamespacedID) (streaming.QueryParams, error) {
				return streaming.QueryParams{
					FilterSubject: []string{"subject-1"},
				}, nil
			},
			GetUsagePeriodStartAt: func(ctx context.Context, ownerID models.NamespacedID, at time.Time) (time.Time, error) {
				return periodStart, nil
			},
		})
	}

	// QueryUsage opts into the cache itself (the WP7 credit call site sets Cachable=true
	// on every read), so the ground truth cannot be produced by the same querier with a
	// flag flipped — it comes from an identical querier bound to a cache-disabled
	// connector, whose gate rejects every query into the untouched live path.
	cachedQuerier := newUsageQuerier(c.Connector)
	liveQuerier := newUsageQuerier(s.newCacheConnector(ctx, CacheConfig{}).Connector)

	ownerID := models.NamespacedID{Namespace: namespace, ID: "owner-1"}

	queryBothUsage := func(period timeutil.ClosedPeriod) (liveUsage, cachedUsage float64) {
		liveUsage, err := liveQuerier.QueryUsage(ctx, ownerID, period)
		s.NoError(err)

		c.logs.Reset()

		cachedUsage, err = cachedQuerier.QueryUsage(ctx, ownerID, period)
		s.NoError(err)
		s.NotContains(c.logs.String(), "serving live", "credit querier fell back to the live path")

		return liveUsage, cachedUsage
	}

	// then:
	// - the two-leg subtraction (period.From after the usage period start forces both the
	//   uniq-at-To and uniq-at-From queries) matches live exactly: u1..u3 by To, u1+u2 by
	//   From → 1
	subtractionPeriod := timeutil.ClosedPeriod{From: now.Add(-2 * time.Hour), To: now}

	liveUsage, cachedUsage := queryBothUsage(subtractionPeriod)
	s.Equal(float64(1), liveUsage)
	s.Equal(liveUsage, cachedUsage)

	// given:
	// - a poison unique user lands in the cached bucket bypassing BatchInsert (no marker,
	//   no refresh)
	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", cachedBucket.Add(20*time.Minute), `{"user": "u4"}`, time.Now().UTC()),
	)

	// then:
	// - a single-leg usage read over the whole period (period.From equals the usage
	//   period start, so only the uniq-at-To query runs) serves the stale cached state:
	//   live counts u4, cached does not — the proof the querier's reads are cache-served
	fullPeriod := timeutil.ClosedPeriod{From: periodStart, To: now}

	liveUsage, cachedUsage = queryBothUsage(fullPeriod)
	s.Equal(float64(4), liveUsage)
	s.Equal(float64(3), cachedUsage)

	// when:
	// - the marker the ingestion hook would have written lands together with a refresh
	//   (modeling the BatchInsert direction through the credit path)
	windows, err := lateEventWindows([]streaming.RawEvent{
		rawCacheTestEvent(namespace, eventType, "subject-1", cachedBucket.Add(20*time.Minute), `{"user": "u4"}`, time.Now().UTC()),
	}, time.Now().UTC(), time.Hour, CacheGrainHour)
	s.NoError(err)
	s.Len(windows, 1)

	markerSQL, markerArgs := insertInvalidationMarkers{Database: s.Database, Windows: windows}.toSQL()
	s.NoError(s.ClickHouse.Exec(ctx, markerSQL, markerArgs...))

	s.refreshViewUntilMarkersHealed(ctx, mv.name())

	// then:
	// - the refresh healed the marker and recomputed the bucket: the cached usage
	//   converges to live, u4 included
	liveUsage, cachedUsage = queryBothUsage(fullPeriod)
	s.Equal(float64(4), liveUsage)
	s.Equal(liveUsage, cachedUsage)
}
