package clickhouse

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/huandu/go-sqlbuilder"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// meterCacheMVListingTTL bounds how often the invalidator rescans system.tables during a
// late-event burst (G7). The deployed MV set only changes when the lifecycle reconciler
// creates or drops views, so a short TTL trades at most a few seconds of trigger lag on a
// brand-new view for not paying a system.tables scan on every ingest batch.
const meterCacheMVListingTTL = 15 * time.Second

// invalidationWindow is one late-event invalidation marker payload: the half-open
// [WindowLo, WindowHi) range of UTC-aligned cache grain buckets in one (namespace, event
// type) pair that received events after those buckets had already settled. Readers treat
// cached buckets overlapping an unhealed marker as untrustworthy and serve them live.
type invalidationWindow struct {
	Namespace string
	EventType string
	WindowLo  time.Time
	WindowHi  time.Time
}

// invalidationKey groups late events into one marker per (namespace, event type): markers
// gate reads per meter event type, so finer per-bucket markers would only grow the table
// without changing reader behavior.
type invalidationKey struct {
	namespace string
	eventType string
}

// lateEventWindows computes the invalidation markers a freshly ingested batch requires:
// one (namespace, event type) marker spanning the min..max grain buckets that received
// late events. An event is late when its event time is older than the freshness horizon
// (now − minimumUsageAge): its bucket is already settled — scheduled refreshes have
// published it as final and readers serve it from the cache — so without a marker cached
// reads would silently miss the new event until a refresh happens to recompute the bucket.
//
// The lateness cutoff is deliberately the raw horizon, not the grain-aligned settled bound
// the MVs use: the app clock here and the ClickHouse clock deciding settlement can differ,
// and over-marking a not-yet-settled bucket only costs a spurious live fallback, while
// under-marking a settled one would leave cached reads stale. Bucket bounds are truncated
// in UTC to match om_meter_cache windowstart alignment.
func lateEventWindows(events []streaming.RawEvent, now time.Time, minimumUsageAge time.Duration, grain CacheGrain) ([]invalidationWindow, error) {
	spec, err := grainSpecFor(grain)
	if err != nil {
		return nil, err
	}

	grainDuration := time.Duration(spec.seconds) * time.Second
	cutoff := now.Add(-minimumUsageAge)

	merged := map[invalidationKey]invalidationWindow{}

	for _, event := range events {
		if !event.Time.Before(cutoff) {
			continue
		}

		bucketLo := event.Time.UTC().Truncate(grainDuration)
		bucketHi := bucketLo.Add(grainDuration)

		key := invalidationKey{namespace: event.Namespace, eventType: event.Type}

		window, ok := merged[key]
		if !ok {
			merged[key] = invalidationWindow{
				Namespace: event.Namespace,
				EventType: event.Type,
				WindowLo:  bucketLo,
				WindowHi:  bucketHi,
			}

			continue
		}

		if bucketLo.Before(window.WindowLo) {
			window.WindowLo = bucketLo
		}

		if bucketHi.After(window.WindowHi) {
			window.WindowHi = bucketHi
		}

		merged[key] = window
	}

	windows := slices.Collect(maps.Values(merged))

	// Sorted so marker inserts and tests are deterministic regardless of map iteration
	slices.SortFunc(windows, func(a, b invalidationWindow) int {
		if c := strings.Compare(a.Namespace, b.Namespace); c != 0 {
			return c
		}

		return strings.Compare(a.EventType, b.EventType)
	})

	return windows, nil
}

// insertInvalidationMarkers renders the INSERT persisting late-event invalidation markers.
//
// created_at is deliberately absent from the column list so ClickHouse fills it from the
// table's DEFAULT now64(3) — server time, never the app clock (G6). The reader's heal rule
// compares marker time against refresh times reported by system.view_refreshes; if one
// side of that comparison came from a skewed app clock, a marker could be judged healed by
// a refresh that never saw the late events, silently serving stale buckets.
type insertInvalidationMarkers struct {
	Database string
	Windows  []invalidationWindow
}

func (q insertInvalidationMarkers) toSQL() (string, []interface{}) {
	query := sqlbuilder.ClickHouse.NewInsertBuilder()
	query.InsertInto(getTableName(q.Database, meterCacheInvalidationsTableName))
	query.Cols("namespace", "event_type", "window_lo", "window_hi")

	for _, window := range q.Windows {
		query.Values(window.Namespace, window.EventType, window.WindowLo, window.WindowHi)
	}

	return query.Build()
}

// refreshThrottler rate limits best-effort SYSTEM REFRESH VIEW triggers per view. Late
// events tend to arrive in bursts (a delayed producer flushing its backlog), and each
// batch would otherwise fire another refresh at the same MV; one trigger per interval is
// enough because a refresh recomputes every dirty bucket, not just one event's.
//
// The state is in-process only: sink replicas throttle independently, and concurrent
// triggers from different replicas are absorbed server-side (a refresh request on an
// already-refreshing view is a no-op).
type refreshThrottler struct {
	minInterval time.Duration

	mu          sync.Mutex
	lastTrigger map[string]time.Time
}

func newRefreshThrottler(minInterval time.Duration) *refreshThrottler {
	return &refreshThrottler{
		minInterval: minInterval,
		lastTrigger: map[string]time.Time{},
	}
}

// allow reports whether a refresh of view may fire at now, and when it may, records now as
// the view's last trigger so further calls within minInterval are suppressed. The slot is
// consumed by the decision, not by trigger success: callers deliberately do not roll back
// on a failed trigger so persistent errors (e.g. a missing SYSTEM REFRESH grant) surface
// once per interval instead of once per ingest batch.
func (t *refreshThrottler) allow(view string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if last, ok := t.lastTrigger[view]; ok && now.Sub(last) < t.minInterval {
		return false
	}

	t.lastTrigger[view] = now

	return true
}

// deployedCacheMV is one cache MV discovered in system.tables: its object name and the
// parsed comment metadata.
type deployedCacheMV struct {
	Name     string
	Metadata meterCacheMVMetadata
}

// affectedViewNames returns the deployed cache MVs serving any of the invalidated
// (namespace, event type) pairs, matched against the exact namespace and event_type
// recorded in the MV comment metadata — the folded name prefix is not consulted, so a
// namespace collision on the 8-hex fold never nudges another namespace's view.
// Correctness never depends on this matching — markers gate the reads — it only decides
// which views get refreshed early.
func affectedViewNames(views []deployedCacheMV, windows []invalidationWindow) []string {
	var names []string

	for _, window := range windows {
		for _, view := range views {
			if view.Metadata.Namespace == window.Namespace && view.Metadata.EventType == window.EventType {
				names = append(names, view.Name)
			}
		}
	}

	slices.Sort(names)

	return slices.Compact(names)
}

// meterCacheInvalidator reacts to late events observed at ingestion: it writes
// invalidation markers so cached reads stop trusting the affected buckets, then
// best-effort triggers the affected MVs' refresh so those buckets converge sooner than the
// next scheduled refresh.
//
// Everything here is strictly best-effort with respect to ingestion: the events are
// already durably stored when this runs, so no failure below may reach the ingest caller —
// a lost marker degrades cache freshness (bounded by the scheduled refresh dirty window
// and the reconciler), while a failed ingest response would make the caller re-send events
// that were already written. Failures are logged and counted instead.
type meterCacheInvalidator struct {
	logger     *slog.Logger
	clickhouse clickhouse.Conn
	database   string
	cache      CacheConfig

	observability *meterCacheObservability

	throttler *refreshThrottler

	viewListingTTL time.Duration
	viewsMu        sync.Mutex
	viewsFetchedAt time.Time
	views          []deployedCacheMV

	// Process-local counters kept alongside the OTel instruments in observability: tests
	// assert on these directly without needing a metrics reader, while observability
	// carries the same signal to operators.
	// markerInsertFailures is the one to alert on (G11): a lost marker is the only failure
	// mode in this pipeline that can leave cached reads silently stale.
	markerInsertFailures   atomic.Uint64
	refreshTriggerFailures atomic.Uint64
	refreshTriggersFired   atomic.Uint64
}

func newMeterCacheInvalidator(config Config, observability *meterCacheObservability) *meterCacheInvalidator {
	return &meterCacheInvalidator{
		logger:         config.Logger,
		clickhouse:     config.ClickHouse,
		database:       config.Database,
		cache:          config.Cache,
		observability:  observability,
		throttler:      newRefreshThrottler(config.Cache.RefreshInterval),
		viewListingTTL: meterCacheMVListingTTL,
	}
}

// meterCacheMarkerFailureStage is the streaming.meter_cache.marker_failures counter's stage
// attribute, identifying which of the two failure modes in the invalidation pipeline
// occurred (G11): classification never reaching a marker, or a computed marker failing to
// persist.
type meterCacheMarkerFailureStage string

const (
	meterCacheMarkerFailureStageClassify meterCacheMarkerFailureStage = "classify"
	meterCacheMarkerFailureStageInsert   meterCacheMarkerFailureStage = "insert"
)

// meterCacheRefreshTriggerOutcome is the streaming.meter_cache.refresh_triggers counter's
// outcome attribute for one candidate view's best-effort refresh trigger.
type meterCacheRefreshTriggerOutcome string

const (
	meterCacheRefreshTriggerOutcomeOK        meterCacheRefreshTriggerOutcome = "ok"
	meterCacheRefreshTriggerOutcomeError     meterCacheRefreshTriggerOutcome = "error"
	meterCacheRefreshTriggerOutcomeThrottled meterCacheRefreshTriggerOutcome = "throttled"
	meterCacheRefreshTriggerOutcomeListError meterCacheRefreshTriggerOutcome = "list_error"
)

// invalidateLateEvents inspects a just-inserted batch for late events and, when found,
// persists invalidation markers and nudges the affected MVs to refresh. It never returns
// an error: see the meterCacheInvalidator contract.
//
// Classification runs before the span starts: BatchInsert calls this on every batch
// regardless of whether the batch contains late events, and the zero-late-events case is
// the steady-state majority. Starting the span only once there is something to report
// (a classification error, or at least one marker window) keeps streaming.meter_cache.invalidate
// from emitting a span per insert when the cache is enabled and no late events occur.
func (i *meterCacheInvalidator) invalidateLateEvents(ctx context.Context, events []streaming.RawEvent) {
	windows, err := lateEventWindows(events, time.Now().UTC(), i.cache.MinimumUsageAge, i.cache.WindowSize)
	if err == nil && len(windows) == 0 {
		return
	}

	ctx, span := i.observability.tracer.Start(ctx, "streaming.meter_cache.invalidate")
	defer span.End()

	if err != nil {
		i.markerInsertFailures.Add(1)
		i.recordMarkerFailure(ctx, meterCacheMarkerFailureStageClassify)
		i.logger.Error("meter cache: late event classification failed, cached reads may serve stale buckets", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "late event classification failed")

		return
	}

	span.SetAttributes(attribute.Int("markers", len(windows)))

	sql, args := insertInvalidationMarkers{Database: i.database, Windows: windows}.toSQL()

	if err := i.clickhouse.Exec(ctx, sql, args...); err != nil {
		i.markerInsertFailures.Add(1)
		i.recordMarkerFailure(ctx, meterCacheMarkerFailureStageInsert)
		i.logger.Error("meter cache: invalidation marker insert failed, cached reads may serve stale buckets", "error", err, "markers", len(windows))
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalidation marker insert failed")
	}

	// Refresh triggering proceeds even when the marker insert failed: a completed refresh
	// re-appends the affected buckets, shrinking the very staleness window the lost marker
	// would have guarded.
	i.triggerRefreshes(ctx, windows)
}

// recordMarkerFailure emits the streaming.meter_cache.marker_failures counter (G11): this is
// the silent-staleness alert signal, so every classify/insert failure must reach it, not
// just the process-local atomic counters.
func (i *meterCacheInvalidator) recordMarkerFailure(ctx context.Context, stage meterCacheMarkerFailureStage) {
	i.observability.markerFailures.Add(ctx, 1, metric.WithAttributes(attribute.String("stage", string(stage))))
}

func (i *meterCacheInvalidator) recordRefreshTrigger(ctx context.Context, outcome meterCacheRefreshTriggerOutcome) {
	i.observability.refreshTriggers.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", string(outcome))))
}

func (i *meterCacheInvalidator) triggerRefreshes(ctx context.Context, windows []invalidationWindow) {
	views, err := i.listDeployedCacheMVs(ctx, time.Now())
	if err != nil {
		i.refreshTriggerFailures.Add(1)
		i.recordRefreshTrigger(ctx, meterCacheRefreshTriggerOutcomeListError)
		i.logger.Warn("meter cache: listing cache views for refresh triggering failed", "error", err)

		return
	}

	for _, view := range affectedViewNames(views, windows) {
		if !i.throttler.allow(view, time.Now()) {
			i.recordRefreshTrigger(ctx, meterCacheRefreshTriggerOutcomeThrottled)

			continue
		}

		if err := i.clickhouse.Exec(ctx, "SYSTEM REFRESH VIEW "+getTableName(i.database, view)); err != nil {
			i.refreshTriggerFailures.Add(1)
			i.recordRefreshTrigger(ctx, meterCacheRefreshTriggerOutcomeError)
			i.logger.Warn("meter cache: refresh trigger failed", "view", view, "error", err)

			continue
		}

		i.refreshTriggersFired.Add(1)
		i.recordRefreshTrigger(ctx, meterCacheRefreshTriggerOutcomeOK)
	}
}

// listDeployedCacheMVs returns the cache MVs deployed in the database, served from an
// in-process snapshot refreshed at most every viewListingTTL (G7). The mutex is held
// across the rescan on purpose: concurrent ingest batches hitting an expired snapshot
// should share one system.tables scan, not race to repeat it.
func (i *meterCacheInvalidator) listDeployedCacheMVs(ctx context.Context, now time.Time) ([]deployedCacheMV, error) {
	i.viewsMu.Lock()
	defer i.viewsMu.Unlock()

	if !i.viewsFetchedAt.IsZero() && now.Sub(i.viewsFetchedAt) < i.viewListingTTL {
		return i.views, nil
	}

	rows, err := i.clickhouse.Query(ctx,
		"SELECT name, comment FROM system.tables WHERE database = ? AND engine = 'MaterializedView' AND startsWith(name, ?)",
		i.database, meterCacheMVNamePrefix,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var views []deployedCacheMV

	for rows.Next() {
		var name, comment string

		if err := rows.Scan(&name, &comment); err != nil {
			return nil, err
		}

		metadata, err := parseMeterCacheMVMetadata(comment)
		if err != nil {
			// A prefix-matching view without valid metadata is foreign or corrupt; refresh
			// triggering must not touch it, and repairing it is the reconciler's job.
			i.logger.Debug("meter cache: skipping view with unparseable comment metadata", "view", name, "error", err)

			continue
		}

		views = append(views, deployedCacheMV{Name: name, Metadata: metadata})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	i.views = views
	i.viewsFetchedAt = now

	return views, nil
}
