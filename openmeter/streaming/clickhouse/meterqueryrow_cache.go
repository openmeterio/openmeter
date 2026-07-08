package clickhouse

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

// meterQueryRowCacheTable is the name of the pre-aggregated rollup table that
// backs the optional meter query-result cache.
const meterQueryRowCacheTable = "meterqueryrow_cache"

// cacheGrainInterval is the tumble interval of the cache's hourly grain.
const cacheGrainInterval = "toIntervalHour(1)"

// createMeterQueryRowCacheTable builds the DDL for the rollup table.
//
// Storage decisions:
//   - ReplacingMergeTree(created_at): combined with the settled-window
//     write-once invariant, any duplicate produced by concurrent lazy
//     population is byte-identical, so replacing by the sort key is safe and
//     collapses them. Plain MergeTree would let the read-time UNION double-count
//     SUM/COUNT.
//   - ORDER BY includes group_by so distinct group-by combos in the same window
//     are distinct rows, not replacement collisions.
//   - sum_value/min_value/max_value are Nullable: a settled window whose events
//     all have a null value property yields a NULL sum/min/max in the live query
//     (ClickHouse sum/min/max over all-NULL is NULL, not 0), and the live scan
//     skips that row. A non-nullable column would store 0 there and both emit a
//     spurious row and corrupt the enclosing day's aggregate. count_value stays
//     non-nullable: count(*) counts rows regardless of value nullness, and a
//     COUNT meter emits that window as a row just as live does.
//   - meter_slug (the per-namespace unique meter key) is in the ORDER BY key:
//     the (namespace, event_type) pair does NOT identify a meter — many meters
//     can share an event type (uniqueness is on (namespace, key)). Two such
//     meters (e.g. SUM of tokens vs SUM of latency) would otherwise share a sort
//     key and one would clobber the other, so a query would read the other
//     meter's value. type is kept for locality/pruning but meter_slug is the
//     discriminator that makes rows meter-specific.
//   - meter_hash pins the meter's EXTRACTION SHAPE (aggregation, value property
//     and sorted group-by map). Meter definitions are mutable (UpdateMeter can add
//     or remove group-by dimensions), and the group_by array is aligned to the
//     CURRENT sorted paths — without the hash, rows written before and after a
//     definition change coexist under different sort keys and the cache leg
//     double-counts every settled hour. With the hash in the key and the read
//     filter, a shape change simply orphans the old rows (never read again).
//   - The query's WindowTimeZone is deliberately NOT part of the key: rows are
//     timezone-agnostic hourly-UTC partials (populate hard-codes UTC), and the
//     timezone only re-windows them at read time. The gate admits only
//     whole-hour-offset zones, where every tz-local window boundary — including
//     DST transition instants — falls on a UTC hour boundary, so one stored row
//     set serves every admitted timezone exactly (pinned by
//     TestQueryCacheCrossTimezoneSharing). Keying by timezone would store
//     byte-identical copies per zone for no correctness gain.
//   - created_at is DateTime64(3): it is both the ReplacingMergeTree version and
//     the argMax pick in the read-time collapse; second granularity would make
//     newest-wins ties far more likely under racing populates.
//   - TTL garbage-collects rows. Gap tracking means covered ranges are NOT
//     rewritten on read, so actively queried rows DO age: the coverage claim's
//     trust window (cacheCoverageTrustWindow, one day shorter than this TTL)
//     expires first and forces a full re-populate that rewrites the rows and
//     resets their clock before any can expire. Rows orphaned by a meter shape
//     change or an idle meter age out unreferenced.
type createMeterQueryRowCacheTable struct {
	Database  string
	TableName string
}

func (t createMeterQueryRowCacheTable) toSQL() string {
	tableName := getTableName(t.Database, t.TableName)

	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	namespace String,
	type LowCardinality(String),
	meter_slug String,
	meter_hash String,
	windowstart DateTime,
	subject String,
	group_by Array(String),
	sum_value Nullable(Decimal128(19)),
	count_value UInt64,
	min_value Nullable(Decimal128(19)),
	max_value Nullable(Decimal128(19)),
	created_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = ReplacingMergeTree(created_at)
PARTITION BY toYYYYMM(windowstart)
ORDER BY (namespace, type, meter_slug, meter_hash, windowstart, subject, group_by)
TTL toDateTime(created_at) + toIntervalDay(90)`, tableName)
}

// meterShapeHash fingerprints everything that determines what a cache row's
// value and group_by array MEAN for a meter: the event type, the aggregation,
// the value property, and the sorted group-by map (keys and JSON paths). It
// deliberately excludes fields that don't change row semantics (name,
// description, EventFrom). Rows are written and read pinned to this hash, so
// any meter definition change makes old rows unreachable instead of silently
// combining two incompatible shapes into one aggregate.
//
// EventType is included even though rollup rows also carry a type column:
// the coverage claim's identity is only (namespace, slug, hash), and a meter
// deleted and recreated under the same slug with a different event type (the
// slug unique index is scoped to non-deleted meters) but identical
// aggregation/value/group-by would otherwise inherit the old claim while the
// cache leg's type filter matches none of the claimed rows — the whole
// settled range would read as empty.
func meterShapeHash(meter meterpkg.Meter) string {
	h := fnv.New64a()

	_, _ = h.Write([]byte(meter.EventType))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(meter.Aggregation))
	_, _ = h.Write([]byte{0})
	if meter.ValueProperty != nil {
		_, _ = h.Write([]byte(*meter.ValueProperty))
	}
	_, _ = h.Write([]byte{0})

	for _, key := range cacheGroupByPaths(meter) {
		_, _ = h.Write([]byte(key))
		_, _ = h.Write([]byte{1})
		_, _ = h.Write([]byte(meter.GroupBy[key]))
		_, _ = h.Write([]byte{2})
	}

	return fmt.Sprintf("%016x", h.Sum64())
}

// cacheGroupByPaths returns the meter's group-by JSON-path dimension keys in
// sorted order. This is the order the group_by Array(String) is aligned to,
// both when populating the cache and when reading it back, so array index i
// always corresponds to the same meter dimension. Subject (a top-level column)
// and customer_id (derived from subject) are NOT stored in the array — subject
// has its own column and customer_id queries are not cached.
func cacheGroupByPaths(meter meterpkg.Meter) []string {
	paths := make([]string, 0, len(meter.GroupBy))
	for key := range meter.GroupBy {
		paths = append(paths, key)
	}
	sort.Strings(paths)
	return paths
}

// aggCacheColumn maps a meter aggregation to the cache column that stores its
// per-window partial and the SQL function that recombines partials across
// windows/groups. Only the four mergeable aggregations are supported; callers
// must gate on canQueryBeCached before reaching here.
func aggCacheColumn(agg meterpkg.MeterAggregation) (column string, recombine string, ok bool) {
	switch agg {
	case meterpkg.MeterAggregationSum:
		return "sum_value", "sum", true
	case meterpkg.MeterAggregationCount:
		// Counts recombine by summing the per-window counts.
		return "count_value", "sum", true
	case meterpkg.MeterAggregationMin:
		return "min_value", "min", true
	case meterpkg.MeterAggregationMax:
		return "max_value", "max", true
	default:
		return "", "", false
	}
}

// populateMeterQueryRowCache rolls up hourly windows in [From, Cutoff) into the
// cache table for a single meter, storing one row per (hourly window, subject,
// full group-by combo). From MUST be hour-aligned (the caller passes the
// head-ceiled boundary) so every stored window is COMPLETE: an incomplete window
// would store a different value for the same key depending on the query's `from`,
// and the read-time collapse would then serve the wrong total.
//
// The filter is (namespace, type, time range) ONLY — deliberately NOT the
// query's subject/group-by filters — so the rollup is subject- and group-
// complete for the meter and can serve any later subject/group subset. The
// value/group extraction reuses queryMeter.rawValueExpr / groupByValueExpr so
// stored values are byte-identical to the live query.
//
// Only the aggregate column matching the meter's aggregation carries a real
// value; the others are filler (NULL, or 0 for the non-nullable count_value).
// A meter has exactly one aggregation, so every query over this meter's type
// reads the same column and the filler columns are never read back. This also
// avoids dereferencing a nil ValueProperty for COUNT meters.
type populateMeterQueryRowCache struct {
	Database        string
	CacheTableName  string
	EventsTableName string
	query           queryMeter
	From            time.Time
	Cutoff          time.Time
}

func (p populateMeterQueryRowCache) toSQL() (string, []interface{}) {
	cacheTable := getTableName(p.Database, p.CacheTableName)
	eventsTable := getTableName(p.Database, p.EventsTableName)
	getColumn := columnFactory(p.EventsTableName)
	timeColumn := getColumn("time")

	paths := cacheGroupByPaths(p.query.Meter)

	groupExprs := make([]string, len(paths))
	for i, path := range paths {
		groupExprs[i] = p.query.groupByValueExpr(path)
	}
	groupArray := "[]"
	if len(groupExprs) > 0 {
		groupArray = "[" + strings.Join(groupExprs, ", ") + "]"
	}

	windowStartExpr := fmt.Sprintf("tumbleStart(%s, %s, 'UTC')", timeColumn, cacheGrainInterval)

	// Only the meter's own aggregation column gets a real value; the others are
	// filler — NULL for the Nullable sum/min/max, 0 for the non-nullable
	// count_value. Filler is constant per meter so it compresses to almost
	// nothing, whereas storing the real count(*) on SUM/MIN/MAX rows costs
	// roughly a fifth of the table for data no read ever touches
	// (aggCacheColumn routes each meter's reads to its owning column only).
	// COUNT meters have no ValueProperty, so we must not emit a value
	// aggregate for them.
	sumExpr, countExpr, minExpr, maxExpr := "CAST(NULL AS Nullable(Decimal128(19)))", "0", "CAST(NULL AS Nullable(Decimal128(19)))", "CAST(NULL AS Nullable(Decimal128(19)))"
	if p.query.Meter.Aggregation == meterpkg.MeterAggregationCount {
		countExpr = "count(*)"
	}
	if p.query.Meter.ValueProperty != nil {
		rawValue := p.query.rawValueExpr()
		switch p.query.Meter.Aggregation {
		case meterpkg.MeterAggregationSum:
			sumExpr = fmt.Sprintf("sum(%s)", rawValue)
		case meterpkg.MeterAggregationMin:
			minExpr = fmt.Sprintf("min(%s)", rawValue)
		case meterpkg.MeterAggregationMax:
			maxExpr = fmt.Sprintf("max(%s)", rawValue)
		}
	}

	groupByTerms := append([]string{"namespace", "type", windowStartExpr, "subject"}, groupExprs...)

	sql := fmt.Sprintf(`INSERT INTO %s (namespace, type, meter_slug, meter_hash, windowstart, subject, group_by, sum_value, count_value, min_value, max_value)
SELECT
	namespace,
	type,
	? AS meter_slug,
	? AS meter_hash,
	%s AS windowstart,
	subject,
	%s AS group_by,
	%s AS sum_value,
	%s AS count_value,
	%s AS min_value,
	%s AS max_value
FROM %s
WHERE namespace = ? AND type = ? AND %s >= ? AND %s < ?
GROUP BY %s`,
		cacheTable,
		windowStartExpr,
		groupArray,
		sumExpr,
		countExpr,
		minExpr,
		maxExpr,
		eventsTable,
		timeColumn, timeColumn,
		strings.Join(groupByTerms, ", "),
	)

	args := []interface{}{
		p.query.Meter.Key,
		meterShapeHash(p.query.Meter),
		p.query.Namespace,
		p.query.Meter.EventType,
		p.From.Unix(),
		p.Cutoff.Unix(),
	}

	return sql, args
}

// cachedMeterRow is one merged result row as returned from the cache/live merge,
// carried as exact decimals end-to-end (never float) until the final
// MeterQueryRow conversion. Duplicate keys cannot occur in the merged output:
// the outer GROUP BY collapses every (window, subject, group-by) tuple to one
// row, and the cache leg's argMax-over-created_at collapse deterministically
// picks the newest stored rollup for each key before recombination.
type cachedMeterRow struct {
	WindowStart time.Time
	WindowEnd   time.Time
	Subject     string
	GroupBy     []string
	Value       alpacadecimal.Decimal
	// ValueValid is false when the recombined value is NULL
	ValueValid bool
}

// deleteMeterQueryRowCacheForNamespaces builds a DELETE that wipes all cached
// rows for the given namespaces.
type deleteMeterQueryRowCacheForNamespaces struct {
	Database   string
	TableName  string
	Namespaces []string
}

func (d deleteMeterQueryRowCacheForNamespaces) toSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.TableName)

	placeholders := strings.Repeat("?, ", len(d.Namespaces))
	placeholders = strings.TrimSuffix(placeholders, ", ")

	sql := fmt.Sprintf("DELETE FROM %s WHERE namespace IN (%s)", tableName, placeholders)

	args := make([]any, 0, len(d.Namespaces))
	for _, ns := range d.Namespaces {
		args = append(args, ns)
	}

	return sql, args
}
