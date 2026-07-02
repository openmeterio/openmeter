package clickhouse

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/alpacahq/alpacadecimal"
	"github.com/huandu/go-sqlbuilder"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/filter"
)

// queryCachedMeter builds the single-statement merge that serves a cacheable
// meter query. It reads settled hourly rollups from the cache table for
// [from, cutoff) and aggregates raw events for the fresh tail [cutoff, to),
// UNIONs them at hourly grain, then re-windows and re-aggregates to the
// requested window size and group-by subset.
//
// Structure:
//
//	SELECT <window cols>, <subject?>, <group subset cols>, <recombine>(value) AS value
//	FROM (
//	  -- cache leg: settled whole hours, collapsed to dedupe concurrent populates (newest wins)
//	  SELECT windowstart AS windowstart_hourly, subject, group_by, argMax(<col>, created_at) AS value
//	  FROM cache WHERE ns/type/meter/window range GROUP BY windowstart, subject, group_by
//	  UNION ALL
//	  -- live legs (sub-hour head + fresh tail): raw events -> hourly rollup, full group-by set
//	  SELECT tumbleStart(time,1h,UTC) AS windowstart_hourly, subject, [json paths] AS group_by, <agg> AS value
//	  FROM events WHERE ns/type/time range GROUP BY windowstart, subject, group exprs
//	)
//	WHERE <subject filter> AND <group-by filters>
//	GROUP BY <window cols>, <subject?>, <group subset cols>
//	ORDER BY windowstart
//
// The cache leg's collapse (inner argMax-over-created_at GROUP BY) is the
// read-time half of the dedup-before-re-aggregation fix (§4.2): even before
// ReplacingMergeTree merges asynchronously, duplicate settled-window rows
// collapse to one row here — deterministically the NEWEST — so the outer sum()
// cannot double-count and a stale row from a populate/invalidation race loses
// to the corrected one.
type queryCachedMeter struct {
	Database        string
	CacheTableName  string
	EventsTableName string
	query           queryMeter
	// Cutoff is the freshness boundary (hour-aligned): [from, cutoff) is served
	// from the cache, [cutoff, to) is scanned live.
	Cutoff time.Time
}

func (q queryCachedMeter) toSQL() (string, []interface{}, error) {
	cacheTable := getTableName(q.Database, q.CacheTableName)
	eventsTable := getTableName(q.Database, q.EventsTableName)
	getColumn := columnFactory(q.EventsTableName)
	timeColumn := getColumn("time")

	column, recombine, ok := aggCacheColumn(q.query.Meter.Aggregation)
	if !ok {
		return "", nil, fmt.Errorf("aggregation %q is not cacheable", q.query.Meter.Aggregation)
	}

	paths := cacheGroupByPaths(q.query.Meter)
	pathIndex := make(map[string]int, len(paths))
	for i, p := range paths {
		pathIndex[p] = i
	}

	// All legs emit an hourly windowstart under a DISTINCT alias
	// (windowstart_hourly) so the outer layer can derive the requested window
	// (hour/day/month) and its GROUP BY from it without the output alias
	// windowstart shadowing the column it is computed from. Using windowstart on
	// both sides risks a cyclic-alias resolution that groups at the wrong grain.
	//
	// The query range [from, to) is partitioned into non-overlapping legs. The
	// cache serves only COMPLETE hour windows [cacheLo, cacheHi):
	//   cacheLo = ceil(from, 1h)     — partial FIRST hour excluded (served live)
	//   cacheHi = floor(cutoff, 1h)  — partial LAST hour excluded (served live)
	// Both boundaries are hour-aligned so every stored window spans a full hour of
	// the query. If that interval is empty (cacheLo >= cacheHi, e.g. a sub-hour or
	// fully-fresh range) there is NO cache leg and the whole range is scanned live.
	// When it is non-empty the live legs are the sub-hour head [from, cacheLo) and
	// the tail [cacheHi, to). These three ranges always tile [from, to) with no gap
	// and no overlap, so no event is dropped or double-counted.
	//
	// Storing only complete hour windows is the write-once invariant that makes
	// concurrent/repeated population byte-identical: a partial window would store a
	// different value for the same key depending on the query's `from`/`to`, and
	// the read-time collapse would then serve the wrong total to a query with
	// different bounds.
	from := q.cacheFrom()
	to := q.freshTo()
	cacheLo := q.headCeil()
	cacheHi := q.cacheHi()

	// --- Live legs (raw events -> hourly rollup, FULL group-by set) ------------
	liveValueExpr, err := q.freshLegValueExpr()
	if err != nil {
		return "", nil, err
	}

	liveGroupExprs := make([]string, len(paths))
	for i, path := range paths {
		liveGroupExprs[i] = q.query.groupByValueExpr(path)
	}
	liveGroupArray := "[]"
	if len(liveGroupExprs) > 0 {
		liveGroupArray = "[" + strings.Join(liveGroupExprs, ", ") + "]"
	}

	liveWindowStart := fmt.Sprintf("tumbleStart(%s, %s, 'UTC')", timeColumn, cacheGrainInterval)
	liveGroupByTerms := append([]string{liveWindowStart, "subject"}, liveGroupExprs...)

	// liveLeg produces a raw-event hourly rollup over the half-open [lo, hi) time
	// range, reused for every live segment.
	liveLegSQL := fmt.Sprintf(`SELECT %[1]s AS windowstart_hourly, subject, %[2]s AS group_by, %[3]s AS value
	FROM %[4]s
	WHERE namespace = ? AND type = ? AND %[5]s >= ? AND %[5]s < ?
	GROUP BY %[6]s`,
		liveWindowStart, liveGroupArray, liveValueExpr, eventsTable, timeColumn, strings.Join(liveGroupByTerms, ", "))

	var legs []string
	var cacheArgs, liveArgs []interface{}

	if cacheLo.Before(cacheHi) {
		// Cache leg (settled WHOLE hours), collapsed per key BEFORE the union so
		// the outer recombine cannot double-count duplicate stored rows.
		// - Filtered by meter_slug (distinct meters can share an event type) AND
		//   meter_hash (a meter definition change orphans old-shape rows instead of
		//   letting two incompatible group_by shapes combine into one aggregate).
		// - The collapse is argMax over created_at — NEWEST Wins — wrapped in
		//   tuple() because bare argMax skips NULL args (a stale non-NULL row would
		//   beat a newer legitimately-NULL rollup, and an all-NULL window would
		//   yield a spurious zero). Under the write-once invariant duplicates are
		//   byte-identical and this equals any(); when a populate races the
		//   late-event invalidation DELETE and persists a stale row, newest-wins
		//   makes reads deterministic and self-healing (the next populate writes a
		//   corrected row with a later created_at).
		legs = append(legs, fmt.Sprintf(`SELECT windowstart AS windowstart_hourly, subject, group_by, tupleElement(argMax(tuple(%[1]s), created_at), 1) AS value
	FROM %[2]s
	WHERE namespace = ? AND type = ? AND meter_slug = ? AND meter_hash = ? AND windowstart >= toDateTime(?) AND windowstart < toDateTime(?)
	GROUP BY windowstart, subject, group_by`,
			column, cacheTable))
		cacheArgs = []interface{}{q.query.Namespace, q.query.Meter.EventType, q.query.Meter.Key, meterShapeHash(q.query.Meter), cacheLo.Unix(), cacheHi.Unix()}

		// Head leg: sub-hour partial first window [from, cacheLo).
		if from.Before(cacheLo) {
			legs = append(legs, liveLegSQL)
			liveArgs = append(liveArgs, q.query.Namespace, q.query.Meter.EventType, from.Unix(), cacheLo.Unix())
		}
		// Tail leg: fresh tail [cacheHi, to).
		if cacheHi.Before(to) {
			legs = append(legs, liveLegSQL)
			liveArgs = append(liveArgs, q.query.Namespace, q.query.Meter.EventType, cacheHi.Unix(), to.Unix())
		}
	} else {
		// No cacheable whole-hour range: scan the entire query range live.
		legs = append(legs, liveLegSQL)
		liveArgs = append(liveArgs, q.query.Namespace, q.query.Meter.EventType, from.Unix(), to.Unix())
	}

	inner := strings.Join(legs, "\n\tUNION ALL\n\t")

	// --- Outer layer: filter, re-window, project subset, re-aggregate ---------
	outerSelect, outerGroupBy, err := q.outerWindowColumns()
	if err != nil {
		return "", nil, err
	}

	// Group-by subset projection + subject column.
	for _, key := range q.query.GroupBy {
		switch key {
		case "subject":
			outerSelect = append(outerSelect, "subject")
			outerGroupBy = append(outerGroupBy, "subject")
		case "customer_id":
			// customer_id is derived from subject and is not cacheable; the gate
			// rejects it. Defensive: refuse rather than emit a wrong column.
			return "", nil, fmt.Errorf("customer_id group by is not cacheable")
		default:
			idx, ok := pathIndex[key]
			if !ok {
				return "", nil, fmt.Errorf("group by %q is not a stored meter dimension", key)
			}
			col := sqlbuilder.Escape(key)
			// ClickHouse arrays are 1-based.
			outerSelect = append(outerSelect, fmt.Sprintf("group_by[%d] AS %s", idx+1, col))
			outerGroupBy = append(outerGroupBy, col)
		}
	}

	outerSelect = append(outerSelect, fmt.Sprintf("%s(value) AS value", recombine))

	// Read-time filters (applied identically to both legs via the UNION output).
	var whereClauses []string
	var whereArgs []interface{}

	// Subject allow-list: both legs carry all subjects, so restrict here.
	if len(q.query.FilterSubject) > 0 {
		subjects := append([]string(nil), q.query.FilterSubject...)
		placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(subjects)), ", ")
		whereClauses = append(whereClauses, fmt.Sprintf("subject IN (%s)", placeholders))
		for _, s := range subjects {
			whereArgs = append(whereArgs, s)
		}
	}

	// FilterGroupBy on stored dimensions.
	if len(q.query.FilterGroupBy) > 0 {
		filterKeys := make([]string, 0, len(q.query.FilterGroupBy))
		for k := range q.query.FilterGroupBy {
			filterKeys = append(filterKeys, k)
		}
		sort.Strings(filterKeys)

		for _, key := range filterKeys {
			fs := q.query.FilterGroupBy[key]
			if fs.IsEmpty() {
				continue
			}
			if err := fs.Validate(); err != nil {
				return "", nil, fmt.Errorf("invalid filter for group by %s: %w", key, err)
			}

			var column string
			switch key {
			case "subject":
				column = "subject"
			case "customer_id":
				return "", nil, fmt.Errorf("customer_id filter is not cacheable")
			default:
				idx, ok := pathIndex[key]
				if !ok {
					return "", nil, fmt.Errorf("filter group by %q is not a stored meter dimension", key)
				}
				column = fmt.Sprintf("group_by[%d]", idx+1)
			}

			expr, exprArgs := filterStringWhere(fs, column)
			whereClauses = append(whereClauses, expr)
			whereArgs = append(whereArgs, exprArgs...)
		}
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "\nWHERE " + strings.Join(whereClauses, " AND ")
	}

	orderBySQL := ""
	if q.query.WindowSize != nil {
		orderBySQL = "\nORDER BY windowstart"
	}

	groupBySQL := ""
	if len(outerGroupBy) > 0 {
		groupBySQL = "\nGROUP BY " + strings.Join(outerGroupBy, ", ")
	}

	sql := fmt.Sprintf(`SELECT %s
FROM (
	%s
)%s%s%s`,
		strings.Join(outerSelect, ", "),
		inner,
		whereSQL,
		groupBySQL,
		orderBySQL,
	)

	// Arg order matches left-to-right placeholder order in the final SQL: outer
	// window args (total: from/to), cache leg, live legs (head then tail), where.
	args := append([]interface{}{}, q.outerWindowArgs()...)
	args = append(args, cacheArgs...)
	args = append(args, liveArgs...)
	args = append(args, whereArgs...)

	// Query settings, mirroring the live path.
	settings := make([]string, 0, len(q.query.QuerySettings))
	for key, value := range q.query.QuerySettings {
		settings = append(settings, fmt.Sprintf("%s = %s", key, value))
	}
	if len(settings) > 0 {
		sql += fmt.Sprintf("\nSETTINGS %s", strings.Join(settings, ", "))
	}

	return sql, args, nil
}

// cacheFrom returns the effective query from (merging Meter.EventFrom). It is
// the lower bound of the sub-hour head leg, NOT the cache leg (which starts at
// headCeil so it only ever stores complete hour windows).
func (q queryCachedMeter) cacheFrom() time.Time {
	from := q.query.from()
	if from == nil {
		// canQueryBeCached requires a from; defensive fallback to cutoff.
		return q.Cutoff
	}
	return *from
}

// headCeil returns the effective from rounded UP to the next hour boundary — the
// start of the first COMPLETE hour window (cache lower bound). The sub-hour head
// [from, headCeil) is served live. If from is already hour-aligned, headCeil ==
// from and the head leg is empty.
func (q queryCachedMeter) headCeil() time.Time {
	from := q.cacheFrom()
	floored := from.Truncate(cacheGrain)
	if floored.Equal(from) {
		return from
	}
	return floored.Add(cacheGrain)
}

// cacheHi returns the cutoff rounded DOWN to the hour boundary — the end of the
// last COMPLETE hour window (cache upper bound, exclusive). The partial last hour
// [cacheHi, cutoff) and everything after is served live via the tail leg. This is
// the symmetric counterpart to headCeil: cutoff can be mid-hour (it is clamped to
// the query's arbitrary `to`), and caching a partial last hour under a whole-hour
// key would poison it exactly as a partial first hour would.
func (q queryCachedMeter) cacheHi() time.Time {
	return q.Cutoff.Truncate(cacheGrain)
}

// freshTo returns the upper bound of the fresh tail leg.
func (q queryCachedMeter) freshTo() time.Time {
	if q.query.To != nil {
		return *q.query.To
	}
	return time.Now().UTC()
}

// freshLegValueExpr returns the fresh-tail aggregate that produces the same
// per-window partial the cache column stores, so both legs are recombined
// identically. COUNT is count(*); SUM/MIN/MAX wrap rawValueExpr.
func (q queryCachedMeter) freshLegValueExpr() (string, error) {
	switch q.query.Meter.Aggregation {
	case meterpkg.MeterAggregationCount:
		return "count(*)", nil
	case meterpkg.MeterAggregationSum:
		return fmt.Sprintf("sum(%s)", q.query.rawValueExpr()), nil
	case meterpkg.MeterAggregationMin:
		return fmt.Sprintf("min(%s)", q.query.rawValueExpr()), nil
	case meterpkg.MeterAggregationMax:
		return fmt.Sprintf("max(%s)", q.query.rawValueExpr()), nil
	default:
		return "", fmt.Errorf("aggregation %q is not cacheable", q.query.Meter.Aggregation)
	}
}

// outerWindowColumns returns the windowstart/windowend SELECT expressions and
// their GROUP BY terms for the requested window size, mirroring meter_query.go
// EXACTLY. The input is the inner hourly windowstart (UTC DateTime). The gate
// guarantees a whole-hour timezone offset, so re-tumbling hourly rows into
// day/month windows in the target tz is exact.
//
// For WindowSize == nil (total) there is no window grouping; windowstart and
// windowend are placeholder From/To constants that QueryMeter overwrites.
func (q queryCachedMeter) outerWindowColumns() (selectCols []string, groupBy []string, err error) {
	if q.query.WindowSize == nil {
		// Total: emit From/To placeholders (overwritten by QueryMeter), no window
		// grouping.
		return []string{"toDateTime(?) AS windowstart", "toDateTime(?) AS windowend"}, nil, nil
	}

	tz := "UTC"
	if q.query.WindowTimeZone != nil {
		tz = q.query.WindowTimeZone.String()
	}

	// The window start/end are derived from windowstart_hourly (the inner hourly
	// grain) and GROUP BY the derived windowstart so distinct output windows
	// collapse. windowend is derived from the same tumbled start, mirroring
	// meter_query.go exactly for each size.
	switch *q.query.WindowSize {
	case meterpkg.WindowSizeHour:
		return []string{
			fmt.Sprintf("tumbleStart(windowstart_hourly, toIntervalHour(1), '%s') AS windowstart", tz),
			fmt.Sprintf("tumbleEnd(windowstart_hourly, toIntervalHour(1), '%s') AS windowend", tz),
		}, []string{"windowstart", "windowend"}, nil

	case meterpkg.WindowSizeDay:
		return []string{
			fmt.Sprintf("tumbleStart(windowstart_hourly, toIntervalDay(1), '%s') AS windowstart", tz),
			fmt.Sprintf("tumbleStart(windowstart_hourly, toIntervalDay(1), '%s') + toIntervalDay(1) AS windowend", tz),
		}, []string{"windowstart", "windowend"}, nil

	case meterpkg.WindowSizeMonth:
		return []string{
			fmt.Sprintf("toDateTime(tumbleStart(windowstart_hourly, toIntervalMonth(1), '%s'), '%s') AS windowstart", tz, tz),
			fmt.Sprintf("toDateTime(tumbleEnd(windowstart_hourly, toIntervalMonth(1), '%s'), '%s') AS windowend", tz, tz),
		}, []string{"windowstart", "windowend"}, nil

	case meterpkg.WindowSizeMinute:
		// Minute windows cannot be reconstructed from hourly rollups; the gate
		// routes these to the live path.
		return nil, nil, fmt.Errorf("minute window size is not cacheable")

	default:
		return nil, nil, fmt.Errorf("invalid window size type: %s", *q.query.WindowSize)
	}
}

// outerWindowArgs returns the args consumed by outerWindowColumns (the From/To
// placeholders for the total case, none otherwise).
func (q queryCachedMeter) outerWindowArgs() []interface{} {
	if q.query.WindowSize != nil {
		return nil
	}

	// Total: placeholders overwritten by QueryMeter, but must be valid times.
	from := q.cacheFrom()
	to := q.freshTo()
	return []interface{}{from.Unix(), to.Unix()}
}

// scanRows scans the merge result into cached rows (exact decimals), then maps
// them to MeterQueryRow honoring the query's requested group-by subset. It
// mirrors meter_query.go's scanRows: first three columns are windowstart,
// windowend, value; the remaining columns are the group-by subset (subject /
// customer_id top-level, or a meter dimension).
func (q queryCachedMeter) scanRows(rows driver.Rows) ([]meterpkg.MeterQueryRow, error) {
	values := []meterpkg.MeterQueryRow{}

	columns := rows.Columns()
	if len(columns) < 3 {
		return nil, fmt.Errorf("cache query returned %d columns, expected at least 3", len(columns))
	}
	if columns[0] != "windowstart" {
		return nil, fmt.Errorf("first column is not windowstart")
	}
	if columns[1] != "windowend" {
		return nil, fmt.Errorf("second column is not windowend")
	}
	// value is last (SELECT appends it after the group columns).
	valueIdx := len(columns) - 1
	if columns[valueIdx] != "value" {
		return nil, fmt.Errorf("last column is not value")
	}

	// Rows are collected as exact decimals; duplicate keys cannot occur (the
	// outer GROUP BY emits one row per key, and the cache leg's argMax collapse
	// deterministically resolves duplicate stored rollups before recombination).
	var cached []cachedMeterRow

	for rows.Next() {
		var (
			windowStart time.Time
			windowEnd   time.Time
			// Reuse the live path's NullDecimal so COUNT (UInt64) and Decimal128
			// values scan through exactly the same code as meter_query.go.
			value NullDecimal
		)

		// Group columns sit between windowend (idx 1) and value (valueIdx).
		groupColumns := columns[2:valueIdx]
		groupDest := make([]string, len(groupColumns))

		dest := []interface{}{&windowStart, &windowEnd}
		for i := range groupDest {
			dest = append(dest, &groupDest[i])
		}
		dest = append(dest, &value)

		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("cache query row scan: %w", err)
		}

		row := cachedMeterRow{
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
			GroupBy:     append([]string(nil), groupDest...),
			ValueValid:  value.Valid,
		}
		if value.Valid {
			row.Value = alpacadecimal.NewFromDecimal(value.Decimal)
		}
		cached = append(cached, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cache query rows error: %w", err)
	}

	groupColumns := columns[2:valueIdx]
	for _, r := range cached {
		// Skip null-value rows, matching the live scan.
		if !r.ValueValid {
			continue
		}

		mrow := meterpkg.MeterQueryRow{
			WindowStart: r.WindowStart,
			WindowEnd:   r.WindowEnd,
			GroupBy:     map[string]*string{},
			// Match live scanRows: convert to float64 only at this boundary.
			Value: r.Value.InexactFloat64(),
		}

		// Attach group-by subset values by column name (mirrors the mapping in
		// meter_query.go's scanRows: subject/customer_id are top-level).
		for i, column := range groupColumns {
			s := r.GroupBy[i]
			switch {
			case column == "subject":
				mrow.Subject = &s
			case column == "customer_id":
				mrow.CustomerID = &s
			case slices.Contains(q.query.GroupBy, column):
				mrow.GroupBy[column] = &s
			default:
				return nil, fmt.Errorf("column %s is not a valid group by", column)
			}
		}

		values = append(values, mrow)
	}

	return values, nil
}

// filterStringWhere renders a filter.FilterString into a WHERE fragment and its
// args, using the connector's sqlbuilder for parameter placeholders. It mirrors
// how meter_query.go applies FilterString via SelectWhereExpr, but produces a
// standalone fragment usable inside the merge's outer WHERE.
func filterStringWhere(fs filter.FilterString, column string) (string, []interface{}) {
	// Build against a throwaway select builder to reuse the filter's own `?`
	// placeholder generation and positional args, then extract the fragment
	// after WHERE. sqlbuilder.ClickHouse emits positional `?` placeholders whose
	// args are returned in order, so splicing the fragment into the merge and
	// appending its args in the same left-to-right order preserves binding.
	sb := sqlbuilder.ClickHouse.NewSelectBuilder()
	expr := fs.SelectWhereExpr(column, sb)
	sb.Select("1").From("_").Where(expr)
	built, args := sb.Build()

	_, frag, _ := strings.Cut(built, "WHERE ")
	return frag, args
}
