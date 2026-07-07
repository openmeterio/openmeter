package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// cacheRejectReason enumerates why a meter query is not served from the cache. Every
// rejection falls back to the untouched live query path; the reasons exist so tests can
// assert the exact gate decision and operators can see why a query stayed live.
type cacheRejectReason string

const (
	cacheRejectReasonNone                 cacheRejectReason = ""
	cacheRejectReasonNotOptedIn           cacheRejectReason = "not_opted_in"
	cacheRejectReasonCacheDisabled        cacheRejectReason = "cache_disabled"
	cacheRejectReasonLatestAggregation    cacheRejectReason = "latest_aggregation"
	cacheRejectReasonDecimalDisabled      cacheRejectReason = "decimal_precision_disabled"
	cacheRejectReasonNoTo                 cacheRejectReason = "to_required"
	cacheRejectReasonTotalWithoutFrom     cacheRejectReason = "total_without_from"
	cacheRejectReasonWindowBelowGrain     cacheRejectReason = "window_size_below_grain"
	cacheRejectReasonInvalidGrain         cacheRejectReason = "invalid_grain"
	cacheRejectReasonTimezone             cacheRejectReason = "timezone_not_cacheable"
	cacheRejectReasonDayGrainTimezone     cacheRejectReason = "day_grain_requires_utc"
	cacheRejectReasonFilterCustomer       cacheRejectReason = "filter_customer"
	cacheRejectReasonCustomerIDGroupBy    cacheRejectReason = "customer_id_group_by"
	cacheRejectReasonClientID             cacheRejectReason = "client_id"
	cacheRejectReasonFilterStoredAt       cacheRejectReason = "filter_stored_at"
	cacheRejectReasonGroupByUnknownKey    cacheRejectReason = "group_by_unknown_key"
	cacheRejectReasonFilterGroupByUnknown cacheRejectReason = "filter_group_by_unknown_key"
	cacheRejectReasonReservedAlias        cacheRejectReason = "reserved_alias"
	cacheRejectReasonViewMissing          cacheRejectReason = "view_missing"
	cacheRejectReasonViewForeign          cacheRejectReason = "view_foreign"
	cacheRejectReasonBackfillUnstamped    cacheRejectReason = "backfill_unstamped"
	cacheRejectReasonViewException        cacheRejectReason = "view_exception"
	cacheRejectReasonViewStale            cacheRejectReason = "view_stale"
	cacheRejectReasonEmptyCacheRange      cacheRejectReason = "empty_cache_range"
	cacheRejectReasonUnhealedMarkers      cacheRejectReason = "unhealed_markers"
)

// meterCacheViewStateTTL memoizes per-view system.tables/system.view_refreshes lookups
// (G13). The window is deliberately short (the design allows 5-15s): a stale snapshot only
// delays noticing a new stamp or a fresh refresh by a few seconds — refreshStart moving
// backward in the reader's view is safe because it shrinks cacheHi and heals fewer markers
// — while an unmemoized gate would pay two system-table lookups on every meter query.
const meterCacheViewStateTTL = 10 * time.Second

// meterCacheStaleRefreshFactor: a view whose last successful refresh is older than this
// many refresh intervals is treated as unhealthy and the query served live. Scheduled
// refreshes run every interval with up to interval/3 randomization, so three intervals
// without a success means refreshes are failing or stopped and the cache tail is growing
// stale beyond what the read bounds assume.
const meterCacheStaleRefreshFactor = 3

// meterCacheHealBound is the G1-tightened marker heal window: an invalidation marker is
// healed by a refresh only when the refresh started after the marker AND
// refreshStart − marker.created_at < healBound. The bound is dirtyWindow −
// minimumUsageAge − refreshInterval because the refresh's stored_at lookback is measured
// from the refresh's own now() (≥ refreshStart), the late events' stored_at is at most a
// beat before the marker's created_at, and the affected buckets additionally had to be
// past the settled bound for the refresh to recompute them at all. Anything looser would
// declare markers healed by refreshes whose lookback provably could not have covered the
// late events, silently serving stale buckets; markers older than any refresh can heal
// are repaired by the reconciler's re-backfill, never assumed healed.
func meterCacheHealBound(minimumUsageAge, refreshInterval time.Duration) time.Duration {
	return meterCacheDirtyWindow(minimumUsageAge, refreshInterval) - minimumUsageAge - refreshInterval
}

// meterCacheWindowSizeRank orders query window sizes for the WindowSize >= grain gate
// rule; -1 means unknown (reject). SECOND ranks below every grain on purpose: the live
// path has no second-window support either, so the gate sends it live to fail there.
func meterCacheWindowSizeRank(w meterpkg.WindowSize) int {
	switch w {
	case meterpkg.WindowSizeSecond:
		return 0
	case meterpkg.WindowSizeMinute:
		return 1
	case meterpkg.WindowSizeHour:
		return 2
	case meterpkg.WindowSizeDay:
		return 3
	case meterpkg.WindowSizeMonth:
		return 4
	default:
		return -1
	}
}

// meterCacheStaticReject evaluates the query-shape half of the cache gate: everything
// decidable from the meter, the query params, and the configuration alone, without
// touching ClickHouse. It returns the first matching reject reason, or
// cacheRejectReasonNone when the dynamic checks (view state, markers) may proceed.
func meterCacheStaticReject(m meterpkg.Meter, params streaming.QueryParams, cache CacheConfig, enableDecimalPrecision bool) cacheRejectReason {
	if !params.Cachable {
		return cacheRejectReasonNotOptedIn
	}

	if !cache.Enabled {
		return cacheRejectReasonCacheDisabled
	}

	// LATEST only ever needs the single newest value in the queried window, so unlike the
	// other aggregations there is no re-aggregation of settled history for the cache to
	// save — a cached LATEST bucket costs a write and a read merge to save recomputing an
	// argMax ClickHouse would otherwise do once, live, over a narrow tail. Excluding it
	// also removes a real correctness surface: the 100M-row benchmark's cache/live parity
	// gate caught non-deterministic value mismatches on cached LATEST multi-subject
	// windowed reads (open finding at the time of this change) that the exclusion sidesteps
	// entirely rather than papering over. LATEST meters always take the live path.
	if m.Aggregation == meterpkg.MeterAggregationLatest {
		return cacheRejectReasonLatestAggregation
	}

	// The cache stores Decimal128 only; the float and decimal live legs have no UNION
	// supertype, so a float-mode deployment can never combine cache and live legs.
	if !enableDecimalPrecision {
		return cacheRejectReasonDecimalDisabled
	}

	if params.To == nil {
		return cacheRejectReasonNoTo
	}

	// A total (nil window) with a nil From keeps its computed WindowStart — the live
	// query derives it from the earliest event's timestamp, which cache buckets cannot
	// reproduce; with From set both window bounds are overwritten with the requested
	// period on every path, so parity holds.
	if params.WindowSize == nil && params.From == nil {
		return cacheRejectReasonTotalWithoutFrom
	}

	grainSpec, err := grainSpecFor(cache.WindowSize)
	if err != nil {
		return cacheRejectReasonInvalidGrain
	}

	// Query windows narrower than the cache grain cannot be reassembled from grain
	// buckets. Totals (nil window) are allowed: they aggregate whole buckets.
	if params.WindowSize != nil {
		rank := meterCacheWindowSizeRank(*params.WindowSize)
		if rank < 0 || rank < meterCacheWindowSizeRank(grainSpec.windowSize) {
			return cacheRejectReasonWindowBelowGrain
		}
	}

	if reason := meterCacheTimezoneReject(params.WindowTimeZone, params.From, *params.To, cache.WindowSize); reason != cacheRejectReasonNone {
		return reason
	}

	// Customer attribution resolves subjects through a query-time map that cache rows
	// do not carry, in both its filter and group-by forms.
	if len(params.FilterCustomer) > 0 {
		return cacheRejectReasonFilterCustomer
	}

	if slices.Contains(params.GroupBy, "customer_id") {
		return cacheRejectReasonCustomerIDGroupBy
	}

	// Progress tracking counts scanned event rows; a cache-leg query scans rollup rows,
	// so the reported progress would be meaningless.
	if params.ClientID != nil {
		return cacheRejectReasonClientID
	}

	// stored_at is not part of the cached rollup: rows aggregate away individual events'
	// storage times, so a stored-at cutoff can only be applied to the raw events table.
	if params.FilterStoredAt != nil && !params.FilterStoredAt.IsEmpty() {
		return cacheRejectReasonFilterStoredAt
	}

	for _, key := range params.GroupBy {
		if key == "subject" || key == "customer_id" {
			continue
		}

		if _, ok := m.GroupBy[key]; !ok {
			return cacheRejectReasonGroupByUnknownKey
		}
	}

	for key := range params.FilterGroupBy {
		if _, ok := m.GroupBy[key]; !ok {
			return cacheRejectReasonFilterGroupByUnknown
		}
	}

	// G9: such meters never get cache SQL generated (no MV exists either), and their
	// group-by keys would shadow columns the read legs depend on.
	if err := reservedAliasCheck(slices.Sorted(maps.Keys(m.GroupBy))); err != nil {
		return cacheRejectReasonReservedAlias
	}

	return cacheRejectReasonNone
}

// meterCacheTimezoneReject decides whether the query's window timezone lets grain-aligned
// UTC cache buckets be re-windowed exactly. Buckets nest into the query's windows only if
// every window boundary in the queried range falls on a bucket boundary:
//
//   - day grain: only UTC itself is accepted (per design; even a zone that happens to sit
//     at offset 0 for the whole range is rejected to keep the rule trivially auditable),
//   - minute/hour grain: the zone's UTC offset must be a whole hour at every sampled
//     instant of [from, to]. Sampling is weekly plus both endpoints, which cannot miss a
//     DST period (real zones hold each offset for months at a time); zones with permanent
//     fractional offsets (Asia/Kathmandu +05:45) fail at the endpoints already,
//   - a nil From with a non-UTC zone is rejected because the window boundaries extend over
//     unbounded history where offsets cannot be sampled (many zones had fractional
//     offsets in the distant past).
func meterCacheTimezoneReject(loc *time.Location, from *time.Time, to time.Time, grain CacheGrain) cacheRejectReason {
	// The live query defaults a nil timezone to UTC.
	if loc == nil || loc == time.UTC || loc.String() == "UTC" {
		return cacheRejectReasonNone
	}

	if grain == CacheGrainDay {
		return cacheRejectReasonDayGrainTimezone
	}

	if from == nil {
		return cacheRejectReasonTimezone
	}

	for t := *from; t.Before(to); t = t.Add(7 * 24 * time.Hour) {
		if _, offset := t.In(loc).Zone(); offset%3600 != 0 {
			return cacheRejectReasonTimezone
		}
	}

	if _, offset := to.In(loc).Zone(); offset%3600 != 0 {
		return cacheRejectReasonTimezone
	}

	return cacheRejectReasonNone
}

// meterCacheViewState is the reader-relevant snapshot of one deployed cache MV, joined
// from system.tables (comment metadata) and system.view_refreshes (health).
type meterCacheViewState struct {
	Exists bool
	// MetadataOK is false when the comment did not parse as valid cache MV metadata; the
	// view must then be treated as foreign or corrupt, never served from.
	MetadataOK bool
	Metadata   meterCacheMVMetadata

	// LastSuccessTime/LastSuccessDurationMS come from system.view_refreshes and are nil
	// until the view's first successful refresh. refreshStart is derived from them.
	LastSuccessTime       *time.Time
	LastSuccessDurationMS *uint64
	Exception             string
}

type meterCacheViewStateEntry struct {
	fetchedAt time.Time
	state     meterCacheViewState
}

// meterCacheGate decides per query whether the cached read path may serve it. All state
// it keeps is a short-TTL memo of per-view system-table lookups; every decision it makes
// falls back to the live path on rejection or error.
type meterCacheGate struct {
	logger     *slog.Logger
	clickhouse clickhouse.Conn
	database   string
	cache      CacheConfig

	enableDecimalPrecision bool

	// fetchViewState is indirect so unit tests can exercise the memoization and the
	// dynamic reject rules without a ClickHouse connection.
	fetchViewState func(ctx context.Context, viewName string) (meterCacheViewState, error)

	viewStateTTL time.Duration
	viewsMu      sync.Mutex
	viewStates   map[string]meterCacheViewStateEntry
}

func newMeterCacheGate(config Config) *meterCacheGate {
	gate := &meterCacheGate{
		logger:                 config.Logger,
		clickhouse:             config.ClickHouse,
		database:               config.Database,
		cache:                  config.Cache,
		enableDecimalPrecision: config.EnableDecimalPrecision,
		viewStateTTL:           meterCacheViewStateTTL,
		viewStates:             map[string]meterCacheViewStateEntry{},
	}

	gate.fetchViewState = gate.fetchViewStateFromClickHouse

	return gate
}

// cacheEligibility runs the full cache gate for one meter query: the static shape checks,
// then the view-state checks (existence, metadata, backfill stamp, refresh health), the
// cache leg bounds, and the invalidation marker scan. It returns the leg bounds and
// cacheRejectReasonNone when the query may be served from the cache; a non-empty reason
// when it must be served live; and an error only for infrastructure failures (which
// callers also treat as "serve live").
//
// The marker scan deliberately happens on every request while the view state is memoized:
// markers are the correctness signal for buckets the cache already published, so a stale
// marker view could serve poisoned buckets, whereas a stale refresh state only makes the
// gate more conservative.
func (g *meterCacheGate) cacheEligibility(ctx context.Context, query queryMeter, params streaming.QueryParams) (meterCacheLegBounds, cacheRejectReason, error) {
	if reason := meterCacheStaticReject(query.Meter, params, g.cache, g.enableDecimalPrecision); reason != cacheRejectReasonNone {
		return meterCacheLegBounds{}, reason, nil
	}

	hash := meterHash(query.Meter, g.cache.WindowSize)

	state, err := g.viewState(ctx, mvName(query.Namespace, hash))
	if err != nil {
		return meterCacheLegBounds{}, cacheRejectReasonNone, fmt.Errorf("meter cache view state: %w", err)
	}

	if !state.Exists {
		return meterCacheLegBounds{}, cacheRejectReasonViewMissing, nil
	}

	if !state.MetadataOK {
		return meterCacheLegBounds{}, cacheRejectReasonViewForeign, nil
	}

	// Two meters with identical shape but different keys share a meter hash and thus an
	// MV name; the MV can only serve the meter key it was generated for, because its
	// SELECT stamps that key into every row. The namespace must match exactly as well:
	// the MV name only folds the namespace to 8 hex chars, so a colliding namespace with
	// a same-shape, same-key meter resolves to this very view while its cache leg
	// (filtered on the querying namespace) would match zero rows — the gate must send it
	// live, not serve empty settled history.
	if state.Metadata.Namespace != query.Namespace ||
		state.Metadata.MeterKey != query.Meter.Key ||
		state.Metadata.EventType != query.Meter.EventType ||
		state.Metadata.MeterHash != formatCacheHash(hash) {
		return meterCacheLegBounds{}, cacheRejectReasonViewForeign, nil
	}

	// G3: an unstamped MV only contains recently refreshed buckets; serving it would
	// silently drop all history older than its first refresh.
	if state.Metadata.BackfilledAt == nil {
		return meterCacheLegBounds{}, cacheRejectReasonBackfillUnstamped, nil
	}

	if state.Exception != "" {
		return meterCacheLegBounds{}, cacheRejectReasonViewException, nil
	}

	if state.LastSuccessTime == nil || state.LastSuccessDurationMS == nil {
		return meterCacheLegBounds{}, cacheRejectReasonViewStale, nil
	}

	if time.Since(*state.LastSuccessTime) > meterCacheStaleRefreshFactor*g.cache.RefreshInterval {
		return meterCacheLegBounds{}, cacheRejectReasonViewStale, nil
	}

	// refreshStart, not last_success_time: the refresh evaluated its settled bound at the
	// moment it started, so everything derived from it (cache horizon, marker healing)
	// must be anchored there.
	refreshStart := state.LastSuccessTime.Add(-time.Duration(*state.LastSuccessDurationMS) * time.Millisecond)

	bounds, ok, err := meterCacheBounds(query.from(), *query.To, refreshStart, g.cache.MinimumUsageAge, g.cache.WindowSize)
	if err != nil {
		return meterCacheLegBounds{}, cacheRejectReasonNone, fmt.Errorf("meter cache bounds: %w", err)
	}

	if !ok {
		return meterCacheLegBounds{}, cacheRejectReasonEmptyCacheRange, nil
	}

	markerSQL, markerArgs := meterCacheMarkerOverlapQuery{
		Database:     g.database,
		Namespace:    query.Namespace,
		EventType:    query.Meter.EventType,
		CacheLo:      bounds.CacheLo,
		CacheHi:      bounds.CacheHi,
		RefreshStart: refreshStart,
		HealBound:    meterCacheHealBound(g.cache.MinimumUsageAge, g.cache.RefreshInterval),
		BackfilledAt: *state.Metadata.BackfilledAt,
	}.toSQL()

	var unhealed uint64
	if err := g.clickhouse.QueryRow(ctx, markerSQL, markerArgs...).Scan(&unhealed); err != nil {
		return meterCacheLegBounds{}, cacheRejectReasonNone, fmt.Errorf("meter cache marker overlap: %w", err)
	}

	if unhealed > 0 {
		return meterCacheLegBounds{}, cacheRejectReasonUnhealedMarkers, nil
	}

	return bounds, cacheRejectReasonNone, nil
}

// viewState returns the memoized view snapshot, refreshing it when older than the TTL
// (G13). The lock is not held across the fetch: a burst of queries on an expired entry
// may fetch redundantly, which is cheaper than serializing every meter's gate behind one
// ClickHouse round-trip.
func (g *meterCacheGate) viewState(ctx context.Context, viewName string) (meterCacheViewState, error) {
	g.viewsMu.Lock()
	entry, ok := g.viewStates[viewName]
	g.viewsMu.Unlock()

	if ok && time.Since(entry.fetchedAt) < g.viewStateTTL {
		return entry.state, nil
	}

	state, err := g.fetchViewState(ctx, viewName)
	if err != nil {
		return meterCacheViewState{}, err
	}

	g.viewsMu.Lock()
	g.viewStates[viewName] = meterCacheViewStateEntry{fetchedAt: time.Now(), state: state}
	g.viewsMu.Unlock()

	return state, nil
}

func (g *meterCacheGate) fetchViewStateFromClickHouse(ctx context.Context, viewName string) (meterCacheViewState, error) {
	rows, err := g.clickhouse.Query(ctx,
		"SELECT t.comment, r.last_success_time, r.last_success_duration_ms, r.exception "+
			"FROM system.tables AS t "+
			"LEFT JOIN system.view_refreshes AS r ON r.database = t.database AND r.view = t.name "+
			"WHERE t.database = ? AND t.name = ?",
		g.database, viewName,
	)
	if err != nil {
		return meterCacheViewState{}, err
	}

	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return meterCacheViewState{}, err
		}

		return meterCacheViewState{Exists: false}, nil
	}

	state := meterCacheViewState{Exists: true}

	var comment string
	if err := rows.Scan(&comment, &state.LastSuccessTime, &state.LastSuccessDurationMS, &state.Exception); err != nil {
		return meterCacheViewState{}, err
	}

	if metadata, err := parseMeterCacheMVMetadata(comment); err == nil {
		state.MetadataOK = true
		state.Metadata = metadata
	}

	return state, nil
}
