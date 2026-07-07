package clickhouse

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/huandu/go-sqlbuilder"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

// meterCacheBucketAlias is the column name under which every leg subquery of the cached
// read emits its grain-aligned UTC bucket start. It is deliberately distinct from
// windowstart: the outer query re-windows buckets into the query's window size and
// timezone under the windowstart alias, and ClickHouse resolving that alias against a
// same-named source column would be ambiguous. The name is in reservedMeterSQLAliases so
// meter group-by keys can never shadow it.
const meterCacheBucketAlias = "windowstart_bucket"

// meterCacheLegBounds is the half-open bucket range [CacheLo, CacheHi) the cache leg of a
// query serves; everything outside it is served by live legs over raw events.
type meterCacheLegBounds struct {
	// CacheLo is the first cached bucket; nil means unbounded below (the query has no
	// lower bound and neither does the meter, so the backfilled cache covers all history).
	CacheLo *time.Time
	CacheHi time.Time
}

// meterCacheBounds computes the cache leg bounds for a query over [from, to) given the
// serving view's last refresh start. from is the query's effective lower bound (already
// merged with the meter's EventFrom, see queryMeter.from); nil means unbounded history.
//
//	cacheLo = ceil(from, grain)
//	cacheHi = floor(min(to, refreshStart − minimumUsageAge), grain) − one grain
//
// The trailing one-grain subtraction is the G5 epsilon: refreshStart is reconstructed
// from second-resolution last_success_time minus a millisecond duration, so it can
// overestimate the instant at which the refresh evaluated its settled bound; backing off
// one whole grain guarantees the reader never serves a bucket the refresh had not
// computed yet. The epsilon is applied unconditionally (even when to is far in the past
// and the min is not the horizon) to keep the tiling arithmetic uniform.
//
// ok is false when the range contains no whole cached bucket; the caller must then run
// the entire query live.
func meterCacheBounds(from *time.Time, to time.Time, refreshStart time.Time, minimumUsageAge time.Duration, grain CacheGrain) (meterCacheLegBounds, bool, error) {
	spec, err := grainSpecFor(grain)
	if err != nil {
		return meterCacheLegBounds{}, false, err
	}

	grainDuration := time.Duration(spec.seconds) * time.Second

	hi := to
	if horizon := refreshStart.Add(-minimumUsageAge); horizon.Before(hi) {
		hi = horizon
	}

	cacheHi := hi.Truncate(grainDuration).Add(-grainDuration)

	bounds := meterCacheLegBounds{CacheHi: cacheHi.UTC()}

	if from == nil {
		return bounds, true, nil
	}

	cacheLo := from.Truncate(grainDuration)
	if cacheLo.Before(*from) {
		cacheLo = cacheLo.Add(grainDuration)
	}

	cacheLo = cacheLo.UTC()
	bounds.CacheLo = &cacheLo

	return bounds, bounds.CacheHi.After(cacheLo), nil
}

// meterCacheNewestWinsExprs returns the cache leg SELECT expressions picking the newest
// version of each combine column per stored bucket. om_meter_cache is a
// ReplacingMergeTree: refreshes and backfills re-append full bucket recomputations, so
// the reader must pick the row with the greatest created_at per key instead of trusting
// background merges.
//
// Every pick is argMax over a single tuple of all the aggregation's columns:
//   - the tuple wrap makes argMax keep NULL values (bare argMax skips NULL arguments, so
//     a newer recompute that turned sum_value NULL would lose to an older non-NULL row),
//   - one shared tuple per aggregation makes multi-column picks (AVG's sum + count)
//     always come from the same row version, never mixing versions on created_at ties.
//
// The picks are aliased picked_<column>, never the source column name: ClickHouse
// substitutes SELECT-list aliases into sibling expressions, so aliasing a pick AS
// sum_value would turn a sibling's sum_value argument into the pick expression itself and
// fail with an aggregate-inside-aggregate error (hit live with AVG's two picks). The
// picked_* names are in reservedMeterSQLAliases so group-by keys cannot shadow them.
//
// LATEST has no case here: it is excluded from the cache entirely (see
// meterCacheStaticReject), so this function is never called for it.
func meterCacheNewestWinsExprs(aggregation meterpkg.MeterAggregation) ([]string, error) {
	switch aggregation {
	case meterpkg.MeterAggregationSum:
		return []string{"tupleElement(argMax(tuple(sum_value), created_at), 1) AS picked_sum_value"}, nil
	case meterpkg.MeterAggregationAvg:
		return []string{
			"tupleElement(argMax(tuple(sum_value, value_count), created_at), 1) AS picked_sum_value",
			"tupleElement(argMax(tuple(sum_value, value_count), created_at), 2) AS picked_value_count",
		}, nil
	case meterpkg.MeterAggregationMin:
		return []string{"tupleElement(argMax(tuple(min_value), created_at), 1) AS picked_min_value"}, nil
	case meterpkg.MeterAggregationMax:
		return []string{"tupleElement(argMax(tuple(max_value), created_at), 1) AS picked_max_value"}, nil
	case meterpkg.MeterAggregationCount:
		return []string{"tupleElement(argMax(tuple(count_value), created_at), 1) AS picked_count_value"}, nil
	case meterpkg.MeterAggregationUniqueCount:
		return []string{"tupleElement(argMax(tuple(uniq_state), created_at), 1) AS picked_uniq_state"}, nil
	default:
		return nil, models.NewGenericValidationError(
			fmt.Errorf("invalid aggregation type: %s", aggregation),
		)
	}
}

// meterCacheCombinedValueExpr returns the outer SELECT expression combining the legs'
// combine columns into the final value, matching the live query's result exactly:
//
//   - SUM/MIN/MAX/COUNT re-aggregate with themselves across legs and buckets,
//   - AVG divides the summed numerators by the summed non-null value counts; the numerator
//     is converted to Float64 before the division because that is precisely how ClickHouse
//     computes avg() over Decimal inputs, keeping the result bit-identical to live,
//   - UNIQUE_COUNT merges the uniqExact states — distinct counts of two legs can never be
//     summed, shared values would double count.
//
// LATEST has no case here: it is excluded from the cache entirely (see
// meterCacheStaticReject), so this function is never called for it.
//
// NULL propagation is load-bearing for row-emission parity: sum/min/max/avg over
// all-NULL inputs yield NULL and scanRows drops the row exactly like the live query,
// while uniqExactMerge and sum(count_value) yield 0 and the row is emitted with 0, again
// like live.
// The combiners reference the picked_* column names because UNION ALL takes its column
// names from the first (cache) leg; live legs align positionally.
func meterCacheCombinedValueExpr(aggregation meterpkg.MeterAggregation) (string, error) {
	switch aggregation {
	case meterpkg.MeterAggregationSum:
		return "sum(picked_sum_value) AS value", nil
	case meterpkg.MeterAggregationAvg:
		return "toFloat64(sum(picked_sum_value)) / sum(picked_value_count) AS value", nil
	case meterpkg.MeterAggregationMin:
		return "min(picked_min_value) AS value", nil
	case meterpkg.MeterAggregationMax:
		return "max(picked_max_value) AS value", nil
	case meterpkg.MeterAggregationCount:
		return "sum(picked_count_value) AS value", nil
	case meterpkg.MeterAggregationUniqueCount:
		return "uniqExactMerge(picked_uniq_state) AS value", nil
	default:
		return "", models.NewGenericValidationError(
			fmt.Errorf("invalid aggregation type: %s", aggregation),
		)
	}
}

// meterCacheReadQuery assembles the cached meter query: up to three legs tiling
// [from, to) exactly — a live leg over raw events for [from, cacheLo), the cache leg for
// [cacheLo, cacheHi), and a live leg for [cacheHi, to) — UNION ALL'd in combine form and
// re-aggregated by an outer query into the exact column layout the live meter query
// produces (windowstart, windowend, value, group-by dimensions), so queryMeter.scanRows
// consumes both paths' results identically.
type meterCacheReadQuery struct {
	queryMeter

	Grain   CacheGrain
	CacheLo *time.Time
	CacheHi time.Time
}

func (d meterCacheReadQuery) toSQL() (string, []interface{}, error) {
	if d.To == nil {
		return "", nil, fmt.Errorf("meter cache read requires an upper bound")
	}

	if d.WindowSize == nil && d.From == nil {
		return "", nil, fmt.Errorf("meter cache read requires a lower bound for total queries")
	}

	spec, err := grainSpecFor(d.Grain)
	if err != nil {
		return "", nil, err
	}

	cacheLeg, err := d.cacheLeg()
	if err != nil {
		return "", nil, err
	}

	legs := []sqlbuilder.Builder{cacheLeg}

	// The pre leg only exists when the query's effective lower bound lies strictly below
	// the first cached bucket; a grain-aligned bound makes the cache leg start exactly
	// there. An unbounded query (CacheLo nil) has no pre leg by construction.
	effectiveFrom := d.from()
	if effectiveFrom != nil && d.CacheLo != nil && effectiveFrom.Before(*d.CacheLo) {
		preLeg, err := d.liveLeg(spec, *effectiveFrom, *d.CacheLo)
		if err != nil {
			return "", nil, err
		}

		legs = append(legs, preLeg)
	}

	// The post leg always exists: cacheHi is strictly below to by construction (G5
	// epsilon), so the freshest tail is always served live.
	postLeg, err := d.liveLeg(spec, d.CacheHi, *d.To)
	if err != nil {
		return "", nil, err
	}

	legs = append(legs, postLeg)

	valueExpr, err := meterCacheCombinedValueExpr(d.Meter.Aggregation)
	if err != nil {
		return "", nil, err
	}

	outer := sqlbuilder.ClickHouse.NewSelectBuilder()

	tz := "UTC"
	if d.WindowTimeZone != nil {
		tz = d.WindowTimeZone.String()
	}

	var selectColumns, groupByColumns []string

	if d.WindowSize != nil {
		windowColumns, err := windowExprs(*d.WindowSize, meterCacheBucketAlias, tz)
		if err != nil {
			return "", nil, err
		}

		selectColumns = append(selectColumns, windowColumns...)
		groupByColumns = append(groupByColumns, "windowstart", "windowend")
	} else {
		// Total queries: the live path derives these bounds from min/max event times, but
		// every consumer sees them overwritten with the requested period by
		// Connector.QueryMeter (the gate requires From and To for totals), so constants
		// carrying the right type are sufficient here.
		selectColumns = append(selectColumns,
			fmt.Sprintf("toDateTime(%d) AS windowstart", d.From.Unix()),
			fmt.Sprintf("toDateTime(%d) AS windowend", d.To.Unix()),
		)
	}

	selectColumns = append(selectColumns, valueExpr)

	for _, key := range d.GroupBy {
		if key == "subject" {
			selectColumns = append(selectColumns, "subject")
			groupByColumns = append(groupByColumns, "subject")

			continue
		}

		column := sqlbuilder.Escape(key)
		selectColumns = append(selectColumns, column)
		groupByColumns = append(groupByColumns, column)
	}

	outer.Select(selectColumns...)
	outer.From(outer.BuilderAs(sqlbuilder.UnionAll(legs...), "legs"))

	if len(groupByColumns) > 0 {
		outer.GroupBy(groupByColumns...)
	}

	if d.WindowSize != nil {
		outer.OrderBy("windowstart")
	}

	sql, args := outer.Build()

	if len(d.QuerySettings) > 0 {
		settings := make([]string, 0, len(d.QuerySettings))
		for _, key := range slices.Sorted(maps.Keys(d.QuerySettings)) {
			settings = append(settings, fmt.Sprintf("%s = %s", key, d.QuerySettings[key]))
		}

		sql += fmt.Sprintf(" SETTINGS %s", strings.Join(settings, ", "))
	}

	return sql, args, nil
}

// cacheLeg builds the leg reading [CacheLo, CacheHi) from om_meter_cache. It groups by
// the full stored row key (bucket, subject, full group_by array) so newest-wins picks one
// version per stored row; collapsing to the query's dimensionality happens in the outer
// query together with the cross-leg combine.
func (d meterCacheReadQuery) cacheLeg() (*sqlbuilder.SelectBuilder, error) {
	sb := sqlbuilder.ClickHouse.NewSelectBuilder()

	selectColumns := []string{fmt.Sprintf("windowstart AS %s", meterCacheBucketAlias)}

	meterGroupByKeys := slices.Sorted(maps.Keys(d.Meter.GroupBy))

	groupByArrayIndex := func(key string) (int, error) {
		// group_by array elements are stored in sorted meter group-by key order (see
		// meterCacheSelectSQL), which makes this positional decoding unambiguous.
		idx := slices.Index(meterGroupByKeys, key)
		if idx < 0 {
			return 0, fmt.Errorf("query group by %s is not a meter group by", key)
		}

		return idx + 1, nil
	}

	for _, key := range d.GroupBy {
		if key == "subject" {
			selectColumns = append(selectColumns, "subject")

			continue
		}

		idx, err := groupByArrayIndex(key)
		if err != nil {
			return nil, err
		}

		selectColumns = append(selectColumns, fmt.Sprintf("group_by[%d] AS %s", idx, sqlbuilder.Escape(key)))
	}

	pickExprs, err := meterCacheNewestWinsExprs(d.Meter.Aggregation)
	if err != nil {
		return nil, err
	}

	selectColumns = append(selectColumns, pickExprs...)

	sb.Select(selectColumns...)
	sb.From(getTableName(d.Database, meterCacheTableName))

	sb.Where(sb.Equal("namespace", d.Namespace))
	// meter_key on top of meter_hash: meters with identical shape share a hash, and rows
	// of a same-shape sibling meter must never be co-read.
	sb.Where(sb.Equal("meter_key", d.Meter.Key))
	sb.Where(sb.Equal("meter_hash", meterHash(d.Meter, d.Grain)))

	if d.CacheLo != nil {
		sb.Where(sb.GreaterEqualThan("windowstart", d.CacheLo.Unix()))
	}

	sb.Where(sb.LessThan("windowstart", d.CacheHi.Unix()))

	if len(d.FilterSubject) > 0 {
		sb.Where(sb.In("subject", d.FilterSubject))
	}

	err = d.applyGroupByFilters(sb, func(key string) (string, error) {
		idx, err := groupByArrayIndex(key)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("group_by[%d]", idx), nil
	})
	if err != nil {
		return nil, err
	}

	sb.GroupBy("windowstart", "subject", "group_by")

	return sb, nil
}

// liveLeg builds a leg aggregating raw events of [from, to) into grain buckets in combine
// form, using the exact value/group-by expressions the live query uses so leg rows are
// interchangeable with cache rows.
func (d meterCacheReadQuery) liveLeg(spec grainSpec, from, to time.Time) (*sqlbuilder.SelectBuilder, error) {
	sb := sqlbuilder.ClickHouse.NewSelectBuilder()

	getColumn := columnFactory(d.EventsTableName)
	timeColumn := getColumn("time")
	dataColumn := getColumn("data")

	// Buckets are aligned to UTC like the cache rows; the outer query re-windows both
	// into the query's timezone.
	selectColumns := []string{
		fmt.Sprintf("tumbleStart(%s, %s, 'UTC') AS %s", timeColumn, spec.tumbleInterval, meterCacheBucketAlias),
	}

	groupBySelectColumns, groupByGroupByColumns := groupBySelectExprs(d.GroupBy, d.Meter.GroupBy, getColumn("subject"), dataColumn)
	selectColumns = append(selectColumns, groupBySelectColumns...)

	combineColumns, err := valueExprsCombine(d.Meter, dataColumn, timeColumn)
	if err != nil {
		return nil, err
	}

	selectColumns = append(selectColumns, combineColumns...)

	sb.Select(selectColumns...)
	sb.From(getTableName(d.Database, d.EventsTableName))

	sb.Where(sb.Equal(getColumn("namespace"), d.Namespace))
	sb.Where(sb.Equal(getColumn("type"), d.Meter.EventType))
	sb = subjectWhere(d.EventsTableName, d.FilterSubject, sb)
	sb.Where(sb.GreaterEqualThan(timeColumn, from.Unix()))
	sb.Where(sb.LessThan(timeColumn, to.Unix()))

	err = d.applyGroupByFilters(sb, func(key string) (string, error) {
		jsonPath, ok := d.Meter.GroupBy[key]
		if !ok {
			return "", fmt.Errorf("filter group by %s is not a meter group by", key)
		}

		return groupByJSONExpr(dataColumn, jsonPath), nil
	})
	if err != nil {
		return nil, err
	}

	sb.GroupBy(append([]string{meterCacheBucketAlias}, groupByGroupByColumns...)...)

	return sb, nil
}

// applyGroupByFilters applies the query's FilterGroupBy predicates to one leg. columnFor
// maps a group-by key to that leg's expression for the dimension (a group_by array
// element on the cache leg, a JSON extraction on live legs). The gate guarantees filter
// keys are meter group-by dimensions — subject and customer_id can never appear here
// because meters carrying them as group-by keys are reserved-alias rejected.
func (d meterCacheReadQuery) applyGroupByFilters(sb *sqlbuilder.SelectBuilder, columnFor func(string) (string, error)) error {
	keys := make([]string, 0, len(d.FilterGroupBy))
	for key := range d.FilterGroupBy {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		filterString := d.FilterGroupBy[key]

		if filterString.IsEmpty() {
			continue
		}

		if err := filterString.Validate(); err != nil {
			return models.NewGenericValidationError(
				fmt.Errorf("invalid filter for group by %s: %w", key, err),
			)
		}

		column, err := columnFor(key)
		if err != nil {
			return err
		}

		sb.Where(filterString.SelectWhereExpr(column, sb))
	}

	return nil
}

// meterCacheMarkerOverlapQuery counts the late-event invalidation markers that overlap
// the cache leg's bucket range and are healed by neither the serving view's backfill nor
// its last refresh. Any non-zero count sends the whole query live: an unhealed marker
// means a settled bucket in range received events the cache has not provably recomputed
// yet.
//
// The heal rules compare ClickHouse-sourced timestamps — marker created_at (server-side
// DEFAULT), refreshStart (derived from system.view_refreshes), and BackfilledAt (read
// from the ClickHouse clock when the backfill started) — so app clock skew can never mark
// a stale bucket healed. A marker is healed iff
//
//   - created_at < BackfilledAt: the view's full-history (re-)backfill started after the
//     marker was written, so it re-aggregated every settled bucket including the marker's
//     late events (this is what lets a reconciler repair converge reads instead of
//     forcing the marked range live until the marker's TTL), or
//   - refreshStart > created_at AND refreshStart − created_at < HealBound (G1): a refresh
//     started after the marker, recently enough that its stored_at lookback provably
//     covered the late events.
//
// The query counts the complement.
type meterCacheMarkerOverlapQuery struct {
	Database  string
	Namespace string
	EventType string

	CacheLo *time.Time
	CacheHi time.Time

	RefreshStart time.Time
	HealBound    time.Duration
	BackfilledAt time.Time
}

func (q meterCacheMarkerOverlapQuery) toSQL() (string, []interface{}) {
	sb := sqlbuilder.ClickHouse.NewSelectBuilder()

	sb.Select("count()")
	sb.From(getTableName(q.Database, meterCacheInvalidationsTableName))

	sb.Where(sb.Equal("namespace", q.Namespace))
	sb.Where(sb.Equal("event_type", q.EventType))
	sb.Where(sb.LessThan("window_lo", q.CacheHi.Unix()))

	if q.CacheLo != nil {
		sb.Where(sb.GreaterThan("window_hi", q.CacheLo.Unix()))
	}

	sb.Where(sb.GreaterEqualThan("created_at", q.BackfilledAt))
	sb.Where(sb.Or(
		sb.GreaterEqualThan("created_at", q.RefreshStart),
		sb.LessEqualThan("created_at", q.RefreshStart.Add(-q.HealBound)),
	))

	return sb.Build()
}

// queryMeterCached attempts to serve the meter query from the cache. served is false —
// and no error is ever returned — whenever the query must run on the live path instead:
// gate rejections are expected steady-state behavior, and infrastructure failures on the
// cache path must degrade to a slower correct answer, never to a failed query.
func (c *Connector) queryMeterCached(ctx context.Context, query queryMeter, params streaming.QueryParams) ([]meterpkg.MeterQueryRow, bool) {
	bounds, reason, err := c.cacheGate.cacheEligibility(ctx, query, params)
	if err != nil {
		c.config.Logger.Warn("meter cache: eligibility check failed, serving live", "meter", query.Meter.Key, "error", err)

		return nil, false
	}

	if reason != cacheRejectReasonNone {
		c.config.Logger.Debug("meter cache: serving live", "meter", query.Meter.Key, "reason", string(reason))

		return nil, false
	}

	readQuery := meterCacheReadQuery{
		queryMeter: query,
		Grain:      c.config.Cache.WindowSize,
		CacheLo:    bounds.CacheLo,
		CacheHi:    bounds.CacheHi,
	}

	sql, args, err := readQuery.toSQL()
	if err != nil {
		c.config.Logger.Warn("meter cache: building cached query failed, serving live", "meter", query.Meter.Key, "error", err)

		return nil, false
	}

	start := time.Now()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		c.config.Logger.Warn("meter cache: cached query failed, serving live", "meter", query.Meter.Key, "error", err)

		return nil, false
	}

	defer rows.Close()

	c.config.Logger.Debug("clickhouse cached meter query executed", "elapsed", time.Since(start).String(), "sql", sql, "args", args)

	values, err := query.scanRows(rows)
	if err != nil {
		c.config.Logger.Warn("meter cache: scanning cached query rows failed, serving live", "meter", query.Meter.Key, "error", err)

		return nil, false
	}

	return values, true
}
