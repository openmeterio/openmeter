package metercache

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	clickhousego "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	progressmanageradapter "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
)

// chTestEnv is the metercache package's ClickHouse harness: a temp database, a raw
// connection for assertions, and a cache-enabled connector. It mirrors the clickhouse
// package's CHTestSuite, which is not importable from here (test-file only).
type chTestEnv struct {
	conn      clickhousego.Conn
	database  string
	connector *clickhouse.Connector
	logs      *bytes.Buffer
}

func newCHTestEnv(t *testing.T) *chTestEnv {
	t.Helper()

	dsn := os.Getenv("TEST_CLICKHOUSE_DSN")
	if dsn == "" {
		t.Skip("TEST_CLICKHOUSE_DSN is not set; skipping integration tests")
	}

	opts, err := clickhousego.ParseDSN(dsn)
	require.NoError(t, err, "failed to parse ClickHouse DSN")

	conn, err := clickhousego.Open(opts)
	require.NoError(t, err, "failed to open ClickHouse connection")

	database := fmt.Sprintf("test_%s", ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String())
	require.NoError(t, conn.Exec(t.Context(), fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", database)))

	t.Cleanup(func() {
		if !t.Failed() {
			_ = conn.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s SYNC", database))
			_ = conn.Close()
		}
	})

	// The connector logs "serving live" on every cache fallback; capturing the logs lets
	// cached queries assert they were really served from the cache leg.
	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	connector, err := clickhouse.New(t.Context(), clickhouse.Config{
		Logger:                 logger,
		ClickHouse:             conn,
		Database:               database,
		EventsTableName:        "om_events",
		EnableDecimalPrecision: true,
		ProgressManager:        progressmanageradapter.NewMockProgressManager(),
		Cache: clickhouse.CacheConfig{
			Enabled:         true,
			RefreshInterval: 10 * time.Minute,
			MinimumUsageAge: time.Hour,
			WindowSize:      clickhouse.CacheGrainHour,
		},
	})
	require.NoError(t, err)

	return &chTestEnv{
		conn:      conn,
		database:  database,
		connector: connector,
		logs:      logs,
	}
}

func (e *chTestEnv) cachedRowCount(t *testing.T, ctx context.Context, formattedMeterHash string) uint64 {
	t.Helper()

	hash, err := strconv.ParseUint(formattedMeterHash, 16, 64)
	require.NoError(t, err)

	// FINAL folds re-appended bucket versions (backfill overlapping the initial refresh)
	// into the newest-wins view the readers see, so counts are version-independent.
	var count uint64
	require.NoError(t, e.conn.QueryRow(ctx,
		fmt.Sprintf("SELECT count() FROM %s.om_meter_cache FINAL WHERE meter_hash = ?", e.database), hash,
	).Scan(&count))

	return count
}

func (e *chTestEnv) totalCachedRowCount(t *testing.T, ctx context.Context) uint64 {
	t.Helper()

	var count uint64
	require.NoError(t, e.conn.QueryRow(ctx, fmt.Sprintf("SELECT count() FROM %s.om_meter_cache FINAL", e.database)).Scan(&count))

	return count
}

// TestReconcilerLifecycle drives the reconciler's full MV lifecycle against a real
// ClickHouse through sequential phases sharing one database: create, leader crash
// mid-backfill repair, meter shape change swap, and meter deletion. Phases build on each
// other on purpose — each starts from the state the previous one converged to, exactly
// like consecutive reconciliation passes in production.
//
// Watched RED with guards reverted: making planViewAction return viewActionNone for the
// unstamped case fails the crash-repair phase (stamp never restored), and dropping the
// DeleteMeterCacheOrphanRows call from reconcile fails the shape-change phase on the
// old-hash row count.
func TestReconcilerLifecycle(t *testing.T) {
	env := newCHTestEnv(t)
	ctx := t.Context()

	const (
		namespace = "cache-lifecycle"
		eventType = "api-calls"
	)

	now := time.Now().UTC()
	bucketA := now.Add(-4 * time.Hour).Truncate(time.Hour)
	bucketB := now.Add(-3 * time.Hour).Truncate(time.Hour)

	// given:
	// - settled events in two hour buckets (two subjects, two dimension values) plus one
	//   event in the always-live tail, inserted directly so no invalidation markers exist
	newEvent := func(subject string, at time.Time, data string) streaming.RawEvent {
		return streaming.RawEvent{
			Namespace:  namespace,
			ID:         ulid.Make().String(),
			Type:       eventType,
			Source:     "test-source",
			Subject:    subject,
			Time:       at,
			Data:       data,
			IngestedAt: now,
			StoredAt:   now,
		}
	}

	insertSQL, insertArgs := clickhouse.InsertEventsQuery{
		Database:        env.database,
		EventsTableName: "om_events",
		Events: []streaming.RawEvent{
			newEvent("subject-1", bucketA.Add(5*time.Minute), `{"value": 2, "group1": "a"}`),
			newEvent("subject-1", bucketA.Add(10*time.Minute), `{"value": 7, "group1": "a"}`),
			newEvent("subject-2", bucketA.Add(20*time.Minute), `{"value": 5, "group1": "b"}`),
			newEvent("subject-1", bucketB.Add(15*time.Minute), `{"value": 3, "group1": "b"}`),
			newEvent("subject-2", now.Add(-20*time.Minute), `{"value": 11, "group1": "a"}`),
		},
	}.ToSQL()
	require.NoError(t, env.conn.Exec(ctx, insertSQL, insertArgs...))

	meterService := &fakeMeterService{}
	reconciler := newTestReconciler(env.connector, meterService)

	from := now.Add(-6 * time.Hour).Truncate(time.Hour)
	to := now.Truncate(time.Hour).Add(time.Hour)
	windowSizeHour := meter.WindowSizeHour

	queryParams := streaming.QueryParams{
		From:       &from,
		To:         &to,
		WindowSize: &windowSizeHour,
		GroupBy:    []string{"subject", "group1"},
	}

	// requireCachedReadParity proves cached rows are what reads actually serve: the cached
	// run must not log a live fallback and must return exactly the live path's rows.
	requireCachedReadParity := func(m meter.Meter) {
		t.Helper()

		params := queryParams
		params.Cachable = false
		live, err := env.connector.QueryMeter(ctx, namespace, m, params)
		require.NoError(t, err)
		require.NotEmpty(t, live, "parity would be trivial on an empty result")

		env.logs.Reset()

		params.Cachable = true
		cached, err := env.connector.QueryMeter(ctx, namespace, m, params)
		require.NoError(t, err)
		require.NotContains(t, env.logs.String(), "serving live", "cached query fell back to the live path")

		require.ElementsMatch(t, live, cached)
	}

	// the capability probe must pass on the test deployment before anything else would run
	version, err := env.connector.ProbeMeterCacheCapabilities(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, version)

	meterV1 := newTestMeter(namespace, "meter-sum", eventType, map[string]string{"group1": "$.group1"})
	desiredV1, err := env.connector.DesiredMeterCacheView(namespace, meterV1)
	require.NoError(t, err)

	t.Run("CreateDeploysBackfillsAndStamps", func(t *testing.T) {
		// when:
		// - the meter appears and a pass runs
		meterService.meters = []meter.Meter{meterV1}
		require.NoError(t, reconciler.reconcile(ctx))

		// then:
		// - the view is deployed with converged metadata and a backfill stamp
		views, err := env.connector.ListActualViews(ctx)
		require.NoError(t, err)
		require.Len(t, views, 1)
		require.Equal(t, desiredV1.Name, views[0].Name)
		require.True(t, views[0].MetadataOK)
		require.Equal(t, meterV1.Key, views[0].MeterKey)
		require.Equal(t, desiredV1.MeterHash, views[0].MeterHash)
		require.Equal(t, desiredV1.DDLHash, views[0].DDLHash)
		require.NotNil(t, views[0].BackfilledAt)

		// then:
		// - the backfill populated the settled buckets (3 subject × dimension rows)
		require.Equal(t, uint64(3), env.cachedRowCount(t, ctx, desiredV1.MeterHash))

		// then:
		// - a converged view is left alone by the next pass
		require.NoError(t, reconciler.reconcile(ctx))
		require.Equal(t, uint64(3), env.cachedRowCount(t, ctx, desiredV1.MeterHash))

		// then:
		// - once the view has a successful refresh, cached reads serve live-equal rows
		require.NoError(t, env.conn.Exec(ctx, fmt.Sprintf("SYSTEM WAIT VIEW %s.%s", env.database, desiredV1.Name)))
		requireCachedReadParity(meterV1)
	})

	t.Run("CrashMidBackfillIsRepairedByNextPass", func(t *testing.T) {
		// given:
		// - the exact state a leader crash between CREATE and backfill leaves behind: the
		//   view exists with an unstamped comment and the cache holds none of its rows
		var comment string
		require.NoError(t, env.conn.QueryRow(ctx,
			"SELECT comment FROM system.tables WHERE database = ? AND name = ?", env.database, desiredV1.Name,
		).Scan(&comment))

		var metadata map[string]any
		require.NoError(t, json.Unmarshal([]byte(comment), &metadata))
		delete(metadata, "backfilled_at")

		unstamped, err := json.Marshal(metadata)
		require.NoError(t, err)

		escaped := strings.ReplaceAll(string(unstamped), `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `'`, `\'`)
		require.NoError(t, env.conn.Exec(ctx, fmt.Sprintf("ALTER TABLE %s.%s MODIFY COMMENT '%s'", env.database, desiredV1.Name, escaped)))

		require.NoError(t, env.conn.Exec(ctx, fmt.Sprintf("DELETE FROM %s.om_meter_cache WHERE true", env.database)))
		require.Equal(t, uint64(0), env.totalCachedRowCount(t, ctx))

		// when:
		// - the next pass runs
		require.NoError(t, reconciler.reconcile(ctx))

		// then:
		// - the pass re-backfilled in place (no drop, so the same view object) and stamped
		views, err := env.connector.ListActualViews(ctx)
		require.NoError(t, err)
		require.Len(t, views, 1)
		require.True(t, views[0].MetadataOK)
		require.NotNil(t, views[0].BackfilledAt)
		require.Equal(t, uint64(3), env.cachedRowCount(t, ctx, desiredV1.MeterHash))
	})

	meterV2 := newTestMeter(namespace, "meter-sum", eventType, map[string]string{"group2": "$.group1"})
	desiredV2, err := env.connector.DesiredMeterCacheView(namespace, meterV2)
	require.NoError(t, err)

	t.Run("ShapeChangeSwapsViewAndGCsOldRows", func(t *testing.T) {
		// when:
		// - the meter's group-by dimension is renamed (new meter hash, new view name)
		meterService.meters = []meter.Meter{meterV2}
		require.NoError(t, reconciler.reconcile(ctx))

		// then:
		// - the old view is gone, the new one is deployed and stamped
		views, err := env.connector.ListActualViews(ctx)
		require.NoError(t, err)
		require.Len(t, views, 1)
		require.Equal(t, desiredV2.Name, views[0].Name)
		require.NotNil(t, views[0].BackfilledAt)

		// then:
		// - the old shape's rows were GC'd in the same pass, the new shape's rows exist
		require.Equal(t, uint64(0), env.cachedRowCount(t, ctx, desiredV1.MeterHash))
		require.Equal(t, uint64(3), env.cachedRowCount(t, ctx, desiredV2.MeterHash))

		// then:
		// - reads under the new shape serve live-equal rows (the meter_hash filter keeps any
		//   old-shape leftovers out of results even before GC lands)
		require.NoError(t, env.conn.Exec(ctx, fmt.Sprintf("SYSTEM WAIT VIEW %s.%s", env.database, desiredV2.Name)))

		params := queryParams
		params.GroupBy = []string{"subject", "group2"}

		paramsLive := params
		paramsLive.Cachable = false
		live, err := env.connector.QueryMeter(ctx, namespace, meterV2, paramsLive)
		require.NoError(t, err)
		require.NotEmpty(t, live)

		env.logs.Reset()

		paramsCached := params
		paramsCached.Cachable = true
		cached, err := env.connector.QueryMeter(ctx, namespace, meterV2, paramsCached)
		require.NoError(t, err)
		require.NotContains(t, env.logs.String(), "serving live", "cached query fell back to the live path")
		require.ElementsMatch(t, live, cached)
	})

	t.Run("MeterDeletionDropsViewAndRows", func(t *testing.T) {
		// when:
		// - the meter disappears from the desired set (soft delete excludes it from listing)
		meterService.meters = nil
		require.NoError(t, reconciler.reconcile(ctx))

		// then:
		// - no cache view remains and every cached row was GC'd
		views, err := env.connector.ListActualViews(ctx)
		require.NoError(t, err)
		require.Empty(t, views)
		require.Equal(t, uint64(0), env.totalCachedRowCount(t, ctx))
	})
}

// rewriteViewMetadata mutates a deployed view's comment metadata JSON in place, modeling
// stamps a past deployment would carry (backdated backfilled_at, adjusted covered_at)
// without waiting real time. Keys mapped to nil are removed.
func rewriteViewMetadata(t *testing.T, ctx context.Context, env *chTestEnv, viewName string, fields map[string]any) {
	t.Helper()

	var comment string
	require.NoError(t, env.conn.QueryRow(ctx,
		"SELECT comment FROM system.tables WHERE database = ? AND name = ?", env.database, viewName,
	).Scan(&comment))

	var metadata map[string]any
	require.NoError(t, json.Unmarshal([]byte(comment), &metadata))

	for key, value := range fields {
		if value == nil {
			delete(metadata, key)

			continue
		}

		metadata[key] = value
	}

	rewritten, err := json.Marshal(metadata)
	require.NoError(t, err)

	escaped := strings.ReplaceAll(string(rewritten), `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	require.NoError(t, env.conn.Exec(ctx, fmt.Sprintf("ALTER TABLE %s.%s MODIFY COMMENT '%s'", env.database, viewName, escaped)))
}

// queryCachedIntent runs one meter query with Cachable set and reports whether the cache
// path actually served it (no "serving live" fallback logged).
func queryCachedIntent(t *testing.T, ctx context.Context, env *chTestEnv, namespace string, m meter.Meter, params streaming.QueryParams) ([]meter.MeterQueryRow, bool) {
	t.Helper()

	env.logs.Reset()

	params.Cachable = true
	rows, err := env.connector.QueryMeter(ctx, namespace, m, params)
	require.NoError(t, err)

	return rows, !strings.Contains(env.logs.String(), "serving live")
}

// TestWatermarkRepairsRestartOutageGap is the extended-outage regression the durable
// coverage watermark exists for: buckets that settled while refreshes were absent longer
// than the dirty-window slack are recomputed by no future refresh — their events'
// stored_at has aged out of the lookback and the newly-settled strip has moved past them
// — and system.view_refreshes cannot reveal the gap after a ClickHouse restart (it is
// wiped: first nil, then fresh again the moment one refresh succeeds). Only the durable
// watermark stamped in the view's comment still shows how old the last covered refresh
// was, and the reconciler must react with a re-backfill.
//
// The outage is modeled without waiting: the gap event's stored_at is backdated beyond
// the dirty window and the deployed view's stamps are rewritten to the pre-outage past,
// exactly the state a restart-after-outage leaves behind, while the refresh state is
// genuinely fresh.
//
// Watched RED with the guard reverted: making planViewAction consult only
// now − LastSuccessTime (the pre-watermark rule) lets the fresh post-outage refresh mask
// the gap — the repair pass becomes a no-op and the final cached read returns 2 instead
// of 102, permanently missing the gap bucket's usage.
func TestWatermarkRepairsRestartOutageGap(t *testing.T) {
	env := newCHTestEnv(t)
	ctx := t.Context()

	const (
		namespace = "cache-restart-outage"
		eventType = "api-calls"
	)

	now := time.Now().UTC()
	bucketCovered := now.Add(-4 * time.Hour).Truncate(time.Hour)
	bucketGap := now.Add(-6 * time.Hour).Truncate(time.Hour)

	newEvent := func(at time.Time, data string, storedAt time.Time) streaming.RawEvent {
		return streaming.RawEvent{
			Namespace:  namespace,
			ID:         ulid.Make().String(),
			Type:       eventType,
			Source:     "test-source",
			Subject:    "subject-1",
			Time:       at,
			Data:       data,
			IngestedAt: storedAt,
			StoredAt:   storedAt,
		}
	}

	insertEvents := func(events ...streaming.RawEvent) {
		insertSQL, insertArgs := clickhouse.InsertEventsQuery{
			Database:        env.database,
			EventsTableName: "om_events",
			Events:          events,
		}.ToSQL()
		require.NoError(t, env.conn.Exec(ctx, insertSQL, insertArgs...))
	}

	// given:
	// - one settled event covered by the initial deploy
	insertEvents(newEvent(bucketCovered.Add(5*time.Minute), `{"value": 2}`, now))

	m := newTestMeter(namespace, "meter-sum", eventType, nil)

	meterService := &fakeMeterService{meters: []meter.Meter{m}}
	reconciler := newTestReconciler(env.connector, meterService)

	require.NoError(t, reconciler.reconcile(ctx))

	desired, err := env.connector.DesiredMeterCacheView(namespace, m)
	require.NoError(t, err)
	require.NoError(t, env.conn.Exec(ctx, fmt.Sprintf("SYSTEM WAIT VIEW %s.%s", env.database, desired.Name)))

	// given:
	// - a gap event: stored before the modeled outage (stored_at aged beyond the 1h30m
	//   dirty lookback), in a bucket the newly-settled strip has long moved past — the
	//   state an on-time event ends up in when its bucket settles mid-outage. It lands
	//   after the deploy backfill, so only a repair re-backfill can ever cache it.
	insertEvents(newEvent(bucketGap.Add(10*time.Minute), `{"value": 100}`, now.Add(-3*time.Hour)))

	require.NoError(t, env.conn.Exec(ctx, fmt.Sprintf("SYSTEM REFRESH VIEW %s.%s", env.database, desired.Name)))
	require.NoError(t, env.conn.Exec(ctx, fmt.Sprintf("SYSTEM WAIT VIEW %s.%s", env.database, desired.Name)))

	from := bucketGap
	to := now.Truncate(time.Hour).Add(time.Hour)
	params := streaming.QueryParams{From: &from, To: &to}

	// then:
	// - the refresh proved unable to recover the gap: the cached read really serves the
	//   cache leg and undercounts by the gap bucket
	rows, servedCached := queryCachedIntent(t, ctx, env, namespace, m, params)
	require.True(t, servedCached, "cached read fell back to the live path")
	require.Len(t, rows, 1)
	require.Equal(t, float64(2), rows[0].Value)

	// given:
	// - the durable stamps say coverage was last provably continuous two hours ago, while
	//   the refresh state is fresh — the exact post-restart shape
	rewriteViewMetadata(t, ctx, env, desired.Name, map[string]any{
		"backfilled_at": now.Add(-2 * time.Hour).Format(time.RFC3339),
		"covered_at":    nil,
	})

	// when:
	// - the next reconciliation pass runs
	require.NoError(t, reconciler.reconcile(ctx))

	// then:
	// - the pass re-backfilled (fresh stamp) and the cached read converges to live,
	//   gap bucket included
	views, err := env.connector.ListActualViews(ctx)
	require.NoError(t, err)
	require.Len(t, views, 1)
	require.NotNil(t, views[0].BackfilledAt)
	require.True(t, views[0].BackfilledAt.After(now.Add(-time.Hour)), "repair must re-stamp the backfill")

	paramsLive := params
	paramsLive.Cachable = false
	live, err := env.connector.QueryMeter(ctx, namespace, m, paramsLive)
	require.NoError(t, err)
	require.Len(t, live, 1)
	require.Equal(t, float64(102), live[0].Value)

	rows, servedCached = queryCachedIntent(t, ctx, env, namespace, m, params)
	require.True(t, servedCached, "cached read fell back to the live path")
	require.Len(t, rows, 1)
	require.Equal(t, float64(102), rows[0].Value)
}

// TestExpiredUnhealedMarkersRepairAndGC drives the reconciler's marker maintenance end to
// end: a marker whose heal window expired unhealed (its late events aged out of every
// future refresh's lookback during a refresh gap) forces a re-backfill of the serving
// view, after which the marker counts as healed-by-backfill and is deleted — so the
// marked range returns to cache-serving instead of staying live until the marker's 7 day
// TTL.
//
// covered_at is pinned fresh while backfilled_at is backdated so the marker path is
// isolated from the watermark outage rule: only the marker report can trigger the repair
// here.
//
// Watched RED with the guard reverted: dropping the ReconcileMeterCacheMarkers call from
// reconcile (returning nil) leaves the marker in place forever — the final marker-count
// assertion reads 1 and the last cached read falls back live (unhealed_markers).
func TestExpiredUnhealedMarkersRepairAndGC(t *testing.T) {
	env := newCHTestEnv(t)
	ctx := t.Context()

	const (
		namespace = "cache-marker-repair"
		eventType = "api-calls"
	)

	now := time.Now().UTC()
	bucket := now.Add(-5 * time.Hour).Truncate(time.Hour)

	newEvent := func(at time.Time, data string, storedAt time.Time) streaming.RawEvent {
		return streaming.RawEvent{
			Namespace:  namespace,
			ID:         ulid.Make().String(),
			Type:       eventType,
			Source:     "test-source",
			Subject:    "subject-1",
			Time:       at,
			Data:       data,
			IngestedAt: storedAt,
			StoredAt:   storedAt,
		}
	}

	insertEvents := func(events ...streaming.RawEvent) {
		insertSQL, insertArgs := clickhouse.InsertEventsQuery{
			Database:        env.database,
			EventsTableName: "om_events",
			Events:          events,
		}.ToSQL()
		require.NoError(t, env.conn.Exec(ctx, insertSQL, insertArgs...))
	}

	insertEvents(
		newEvent(bucket.Add(5*time.Minute), `{"value": 2}`, now),
		newEvent(bucket.Add(10*time.Minute), `{"value": 7}`, now),
	)

	m := newTestMeter(namespace, "meter-sum", eventType, nil)

	meterService := &fakeMeterService{meters: []meter.Meter{m}}
	reconciler := newTestReconciler(env.connector, meterService)

	require.NoError(t, reconciler.reconcile(ctx))

	desired, err := env.connector.DesiredMeterCacheView(namespace, m)
	require.NoError(t, err)
	require.NoError(t, env.conn.Exec(ctx, fmt.Sprintf("SYSTEM WAIT VIEW %s.%s", env.database, desired.Name)))

	// given:
	// - a late event whose stored_at already aged out of the dirty lookback (no refresh
	//   will ever recompute its bucket) and its marker, created 25 minutes ago — past the
	//   20m heal bound, so no future refresh can heal it either,
	// - stamps rewritten so the backfill predates the marker (the deploy really ran before
	//   the modeled outage) while covered_at stays fresh (isolates the marker repair from
	//   the watermark rule)
	insertEvents(newEvent(bucket.Add(20*time.Minute), `{"value": 100}`, now.Add(-3*time.Hour)))

	require.NoError(t, env.conn.Exec(ctx,
		fmt.Sprintf("INSERT INTO %s.om_meter_cache_invalidations (namespace, event_type, window_lo, window_hi, created_at) VALUES (?, ?, ?, ?, ?)", env.database),
		namespace, eventType, bucket, bucket.Add(time.Hour), now.Add(-25*time.Minute),
	))

	rewriteViewMetadata(t, ctx, env, desired.Name, map[string]any{
		"backfilled_at": now.Add(-2 * time.Hour).Format(time.RFC3339),
		"covered_at":    now.Format(time.RFC3339),
	})

	from := bucket
	to := now.Truncate(time.Hour).Add(time.Hour)
	params := streaming.QueryParams{From: &from, To: &to}

	// then:
	// - while the marker is unhealed the reader stays conservative: the query is served
	//   live and therefore already sees the late event
	rows, servedCached := queryCachedIntent(t, ctx, env, namespace, m, params)
	require.False(t, servedCached, "unhealed marker must force the live path")
	require.Contains(t, env.logs.String(), "unhealed_markers")
	require.Len(t, rows, 1)
	require.Equal(t, float64(109), rows[0].Value)

	// when:
	// - the next pass runs: marker maintenance reports the view, the repair re-backfills
	//   (picking up the late event, stamping a fresh backfilled_at past the marker)
	require.NoError(t, reconciler.reconcile(ctx))

	// - and the pass after that deletes the marker, now healed by the fresh backfill
	require.NoError(t, reconciler.reconcile(ctx))

	// then:
	// - the marker is gone and the marked range serves from the cache again, late event
	//   included
	var markerCount uint64
	require.NoError(t, env.conn.QueryRow(ctx,
		fmt.Sprintf("SELECT count() FROM %s.om_meter_cache_invalidations", env.database),
	).Scan(&markerCount))
	require.Equal(t, uint64(0), markerCount)

	rows, servedCached = queryCachedIntent(t, ctx, env, namespace, m, params)
	require.True(t, servedCached, "cached read fell back to the live path")
	require.Len(t, rows, 1)
	require.Equal(t, float64(109), rows[0].Value)
}
