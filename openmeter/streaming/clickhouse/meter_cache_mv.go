package clickhouse

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

// grainSpec describes how a cache grain is expressed in generated SQL.
type grainSpec struct {
	windowSize meterpkg.WindowSize
	// intervalUnit is used in INTERVAL 1 <unit> / toStartOfInterval expressions.
	intervalUnit string
	// tumbleInterval is the toIntervalX(1) form used in tumbleStart expressions.
	tumbleInterval string
	// seconds is the fixed bucket width. Day is a constant 86400 because cache buckets
	// are always UTC-aligned and UTC has no DST transitions.
	seconds int64
}

func grainSpecFor(grain CacheGrain) (grainSpec, error) {
	switch grain {
	case CacheGrainMinute:
		return grainSpec{windowSize: meterpkg.WindowSizeMinute, intervalUnit: "MINUTE", tumbleInterval: "toIntervalMinute(1)", seconds: 60}, nil
	case CacheGrainHour:
		return grainSpec{windowSize: meterpkg.WindowSizeHour, intervalUnit: "HOUR", tumbleInterval: "toIntervalHour(1)", seconds: 3600}, nil
	case CacheGrainDay:
		return grainSpec{windowSize: meterpkg.WindowSizeDay, intervalUnit: "DAY", tumbleInterval: "toIntervalDay(1)", seconds: 86400}, nil
	default:
		return grainSpec{}, fmt.Errorf("invalid meter cache grain: %s", grain)
	}
}

// meterCacheDirtyWindow is the stored_at lookback a scheduled refresh scans for
// recently-touched buckets. Buckets first become cacheable minimumUsageAge after their
// events arrive, so the lookback must exceed age + one refresh interval; the extra
// intervals and the one-hour floor absorb refresh scheduling jitter. The reader's marker
// heal rule (meterCacheHealBound) is derived from the same value, which is why it is a
// shared helper instead of arithmetic inlined at each site.
func meterCacheDirtyWindow(minimumUsageAge, refreshInterval time.Duration) time.Duration {
	dirtyWindow := minimumUsageAge + 3*refreshInterval
	if dirtyWindow < time.Hour {
		dirtyWindow = time.Hour
	}

	return dirtyWindow
}

// sqlStringLiteral renders s as a single-quoted ClickHouse string literal, escaping
// backslashes and single quotes. Generated cache SQL cannot use query arguments (a
// materialized view definition is stored verbatim), so every user-influenced string that
// ends up in cache DDL must go through this.
func sqlStringLiteral(s string) string {
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)

	return fmt.Sprintf("'%s'", escaped)
}

// settledBoundExpr is the exclusive upper event-time bound of cacheable data: the start
// of the grain bucket that lies minimumUsageAge before now. Events at or above it are in
// buckets that may still receive on-time events, so caching them would freeze a partial
// aggregate; the reader serves that tail from the live table instead.
//
// The bucket alignment is explicitly 'UTC' to match the windowstart expression
// (tumbleStart(..., 'UTC')): with the server default timezone, a non-UTC or
// non-whole-hour-offset server would align the bound differently than the buckets it is
// supposed to guard, letting a partially settled bucket slip into the cache.
func settledBoundExpr(minimumUsageAge time.Duration, spec grainSpec) string {
	return fmt.Sprintf(
		"toStartOfInterval(now() - INTERVAL %d SECOND, INTERVAL 1 %s, 'UTC')",
		int64(minimumUsageAge/time.Second),
		spec.intervalUnit,
	)
}

// meterCacheSelectParams parameterizes the single SELECT shape shared by the cache MV
// definition and the backfill INSERT. Both must aggregate identically — the backfill is
// simply the same query over full settled history — so they are rendered by one builder
// and only differ in their time bounds and the dirty-bucket restriction.
type meterCacheSelectParams struct {
	Database        string
	EventsTableName string
	Namespace       string
	Meter           meterpkg.Meter
	Grain           CacheGrain
	MinimumUsageAge time.Duration

	// From is the inclusive lower event-time bound (already resolved against the meter's
	// EventFrom by the caller); nil means unbounded history.
	From *time.Time
	// To is an exclusive upper event-time bound applied in addition to the settled bound;
	// backfill chunking uses it, the MV never does.
	To *time.Time

	// DirtyBucketsOnly restricts the query to recently-touched buckets plus the
	// newly-settled strip. The MV sets it so scheduled refreshes only recompute buckets
	// that can have changed; the backfill leaves it unset to cover full settled history.
	DirtyBucketsOnly bool
	// RefreshInterval sizes the dirty stored_at lookback and the newly-settled strip;
	// required when DirtyBucketsOnly is set.
	RefreshInterval time.Duration
}

// meterCacheSelectSQL renders the aggregation SELECT that populates om_meter_cache rows
// for one meter. Output columns, in order: namespace, meter_key, meter_hash, windowstart,
// subject, group_by, created_at, then the meter's combine columns (valueExprsCombine).
func meterCacheSelectSQL(p meterCacheSelectParams) (string, error) {
	groupByKeys := slices.Sorted(maps.Keys(p.Meter.GroupBy))

	// G9: a group-by key shadowing a source column or a generated alias would silently
	// neutralize the WHERE filters of this very query, so such meters must not get cache
	// SQL generated at all. Callers treat the error as "skip this meter, read it live".
	if err := reservedAliasCheck(groupByKeys); err != nil {
		return "", err
	}

	spec, err := grainSpecFor(p.Grain)
	if err != nil {
		return "", err
	}

	getColumn := columnFactory(p.EventsTableName)
	timeColumn := getColumn("time")
	dataColumn := getColumn("data")

	// Cache buckets are always UTC: the reader re-windows them into the query's timezone,
	// and a single canonical alignment is what lets every query share the same rows.
	windowColumns, err := windowExprs(spec.windowSize, timeColumn, "UTC")
	if err != nil {
		return "", err
	}

	combineColumns, err := valueExprsCombine(p.Meter, dataColumn, timeColumn)
	if err != nil {
		return "", err
	}

	// The meter's group-by dimension values in sorted key order; the same order the
	// reader uses to decode the array positionally.
	groupByElements := make([]string, 0, len(groupByKeys))
	for _, key := range groupByKeys {
		groupByElements = append(groupByElements, groupByJSONExpr(dataColumn, p.Meter.GroupBy[key]))
	}

	// A meter without group-by dimensions must still produce an Array(String) column: the
	// bare empty literal [] types as Array(Nothing), which ClickHouse rejects when the
	// SELECT feeds a materialized view or table column.
	groupByArrayExpr := "emptyArrayString() AS group_by"
	if len(groupByElements) > 0 {
		groupByArrayExpr = fmt.Sprintf("[%s] AS group_by", strings.Join(groupByElements, ", "))
	}

	selectColumns := []string{
		"namespace",
		fmt.Sprintf("%s AS meter_key", sqlStringLiteral(p.Meter.Key)),
		fmt.Sprintf("%d AS meter_hash", meterHash(p.Meter, p.Grain)),
		windowColumns[0],
		"subject",
		groupByArrayExpr,
		"now64(3) AS created_at",
	}
	selectColumns = append(selectColumns, combineColumns...)

	wheres := []string{
		fmt.Sprintf("%s = %s", getColumn("namespace"), sqlStringLiteral(p.Namespace)),
		fmt.Sprintf("%s = %s", getColumn("type"), sqlStringLiteral(p.Meter.EventType)),
	}

	if p.From != nil {
		wheres = append(wheres, fmt.Sprintf("%s >= %d", timeColumn, p.From.Unix()))
	}

	if p.To != nil {
		wheres = append(wheres, fmt.Sprintf("%s < %d", timeColumn, p.To.Unix()))
	}

	wheres = append(wheres, fmt.Sprintf("%s < %s", timeColumn, settledBoundExpr(p.MinimumUsageAge, spec)))

	if p.DirtyBucketsOnly {
		dirtyFilter, err := dirtyBucketFilterExpr(p, spec, timeColumn)
		if err != nil {
			return "", err
		}

		wheres = append(wheres, dirtyFilter)
	}

	return fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s GROUP BY namespace, windowstart, subject, group_by",
		strings.Join(selectColumns, ", "),
		getTableName(p.Database, p.EventsTableName),
		strings.Join(wheres, " AND "),
	), nil
}

// dirtyBucketFilterExpr restricts a scheduled refresh to the buckets that can have
// changed since recent refreshes, as the union of two bucket sets:
//
//   - Recently-touched buckets: any bucket containing an event whose stored_at falls in
//     the dirty window (minimumUsageAge + 3 refresh intervals, floored at 1h). Buckets
//     first become cacheable minimumUsageAge after their events arrive, so the lookback
//     must exceed age + one interval; the extra intervals and the floor absorb refresh
//     scheduling jitter. Any late arrival carries a fresh stored_at, so it lands here on
//     the next refresh.
//   - The newly-settled strip (G2): the buckets whose event time crossed the settled
//     bound since up to 3 refresh intervals ago. Future-dated events and events ingested
//     while their bucket was still unsettled have an old stored_at by the time the bucket
//     settles, so the stored_at lookback alone would never cache them; unconditionally
//     recomputing the strip just below the settled bound closes that gap at a bounded
//     cost. At least one strip bucket is always recomputed even when the refresh interval
//     is shorter than the grain, because the bound advances one whole grain at a time.
func dirtyBucketFilterExpr(p meterCacheSelectParams, spec grainSpec, timeColumn string) (string, error) {
	if p.RefreshInterval < time.Second {
		return "", fmt.Errorf("meter cache refresh interval must be at least one second, got %s", p.RefreshInterval)
	}

	dirtyWindow := meterCacheDirtyWindow(p.MinimumUsageAge, p.RefreshInterval)

	stripSeconds := int64((3 * p.RefreshInterval) / time.Second)
	stripBuckets := (stripSeconds + spec.seconds - 1) / spec.seconds
	if stripBuckets < 1 {
		stripBuckets = 1
	}

	return fmt.Sprintf(
		"toStartOfInterval(%s, INTERVAL 1 %s, 'UTC') IN ("+
			"SELECT DISTINCT toStartOfInterval(time, INTERVAL 1 %s, 'UTC') FROM %s WHERE namespace = %s AND type = %s AND stored_at >= now() - INTERVAL %d SECOND"+
			" UNION DISTINCT "+
			"SELECT subtractSeconds(%s, (number + 1) * %d) FROM numbers(%d))",
		timeColumn,
		spec.intervalUnit,
		spec.intervalUnit,
		getTableName(p.Database, p.EventsTableName),
		sqlStringLiteral(p.Namespace),
		sqlStringLiteral(p.Meter.EventType),
		int64(dirtyWindow/time.Second),
		settledBoundExpr(p.MinimumUsageAge, spec),
		spec.seconds,
		stripBuckets,
	), nil
}

// createMeterCacheMV generates the per-meter refreshable materialized view that maintains
// om_meter_cache rows for one meter shape.
type createMeterCacheMV struct {
	Database        string
	EventsTableName string
	Namespace       string
	Meter           meterpkg.Meter
	Grain           CacheGrain
	RefreshInterval time.Duration
	MinimumUsageAge time.Duration
}

func (d createMeterCacheMV) name() string {
	return mvName(d.Namespace, meterHash(d.Meter, d.Grain))
}

func (d createMeterCacheMV) selectSQL() (string, error) {
	return meterCacheSelectSQL(meterCacheSelectParams{
		Database:         d.Database,
		EventsTableName:  d.EventsTableName,
		Namespace:        d.Namespace,
		Meter:            d.Meter,
		Grain:            d.Grain,
		MinimumUsageAge:  d.MinimumUsageAge,
		From:             d.Meter.EventFrom,
		DirtyBucketsOnly: true,
		RefreshInterval:  d.RefreshInterval,
	})
}

// metadata returns the comment metadata the MV is created with. BackfilledAt is always
// nil here: the stamp is only added (via ALTER ... MODIFY COMMENT) after the backfill
// completes, so a leader crash between CREATE and backfill leaves a visibly unstamped MV
// that readers refuse and the reconciler re-backfills.
func (d createMeterCacheMV) metadata() (meterCacheMVMetadata, error) {
	selectSQL, err := d.selectSQL()
	if err != nil {
		return meterCacheMVMetadata{}, err
	}

	return meterCacheMVMetadata{
		Namespace: d.Namespace,
		MeterKey:  d.Meter.Key,
		EventType: d.Meter.EventType,
		MeterHash: formatCacheHash(meterHash(d.Meter, d.Grain)),
		DDLHash:   formatCacheHash(ddlHash(d.Grain, d.RefreshInterval, d.MinimumUsageAge, d.Meter.EventFrom, selectSQL)),
	}, nil
}

func (d createMeterCacheMV) toSQL() (string, error) {
	if d.RefreshInterval < time.Second {
		return "", fmt.Errorf("meter cache refresh interval must be at least one second, got %s", d.RefreshInterval)
	}

	selectSQL, err := d.selectSQL()
	if err != nil {
		return "", err
	}

	metadata, err := d.metadata()
	if err != nil {
		return "", err
	}

	comment, err := metadata.marshal()
	if err != nil {
		return "", err
	}

	refreshSeconds := int64(d.RefreshInterval / time.Second)
	refreshClause := fmt.Sprintf("REFRESH EVERY %d SECOND", refreshSeconds)

	// RANDOMIZE FOR spreads the refresh load of many MVs sharing the same interval; it is
	// omitted below one second because ClickHouse rejects a zero interval.
	if randomizeSeconds := refreshSeconds / 3; randomizeSeconds >= 1 {
		refreshClause += fmt.Sprintf(" RANDOMIZE FOR %d SECOND", randomizeSeconds)
	}

	// IF NOT EXISTS keeps concurrent reconciler passes benign: the diff decided this MV
	// is missing, and losing a create race must not surface as an error.
	sql := fmt.Sprintf(
		"CREATE MATERIALIZED VIEW IF NOT EXISTS %s %s APPEND TO %s AS %s COMMENT %s",
		getTableName(d.Database, d.name()),
		refreshClause,
		getTableName(d.Database, meterCacheTableName),
		selectSQL,
		sqlStringLiteral(comment),
	)

	// APPEND is load-bearing for the shared target: a non-APPEND refresh atomically
	// replaces the entire om_meter_cache table with this one meter's rows, wiping every
	// other meter's cache. This guard exists so no future edit (e.g. making the refresh
	// mode conditional) can ever emit a non-APPEND cache MV.
	if !strings.Contains(sql, " APPEND TO ") {
		return "", errors.New("generated meter cache MV DDL must contain APPEND TO: a non-APPEND refresh would wipe the shared cache table")
	}

	return sql, nil
}
