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
