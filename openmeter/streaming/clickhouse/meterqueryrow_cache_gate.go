package clickhouse

import (
	"slices"
	"time"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// cacheGrain is the fixed hourly grain the rollup table is stored at. The cache
// can only serve window sizes that re-aggregate cleanly from hourly rows.
const cacheGrain = time.Hour

// queryCacheFeatureFlag is the feature-gate flag that permits the meter
// query-result cache for a namespace. It follows the `om_ff_` convention. By
// default (the Noop gate) it is enabled for every namespace; a configured
// feature-gate backend evaluates this flag and can restrict it per namespace.
const queryCacheFeatureFlag = "om_ff_query_cache_enabled"

// canQueryBeCached reports whether a meter query is provably reproducible from
// the hourly-UTC rollup, i.e. the cached result is byte-identical to the live
// query. Anything not admitted falls through to the live path unchanged — that
// is what makes the cache safely optional. The guards, all of which must hold:
//
//   - The cache is enabled (the master switch, checked first).
//   - Decimal precision is enabled: the rollup stores exact Decimal128(19)
//     values, and only the decimal live path is provably byte-identical to a
//     recombination of them (Float64 sums at billing magnitudes can diverge at
//     the sixth decimal; the UNION would also hit a Decimal/Float64 supertype
//     error).
//   - The query opted in (params.Cachable) — the HTTP meter-query handlers set
//     this; internal callers do not, so they never hit the cache.
//   - No ClientID: progress tracking only exists on the live path — the cached
//     path creates no progress record, so a client polling the progress endpoint
//     would spin on a permanently missing entry.
//   - Both bounds are set. Without From the settled range is unbounded; without
//     To the live path applies no upper time bound (future-dated events count)
//     while the merge would cap at now() — and for the total (nil) window the
//     live path derives the single window end from tumbleEnd(max(event time)),
//     which the cache cannot reproduce.
//   - The effective span is at least MinimumCacheableQueryPeriod.
//   - The aggregation is one of the four mergeable ones (SUM/COUNT/MIN/MAX).
//     AVG and UNIQUE_COUNT are a permanent design boundary: per-window averages
//     don't compose into a global average and exact distinct-counts can't be
//     unioned from partial scalars.
//   - The window size is hour/day/month/total (nil). Minute windows can't be
//     reconstructed from an hourly rollup.
//   - The window timezone offset from UTC is a whole number of hours across the
//     queried range (fractional-hour zones like +05:30 can't be composed from
//     hourly-UTC windows; DST transitions are handled conservatively).
//   - No FilterStoredAt: the rollup has no stored_at dimension, so a stored-at
//     filter can't be reproduced.
//   - No customer_id involvement: customer_id is derived from subject via a
//     dictionary and is not stored in the rollup, so any customer_id group-by or
//     customer filter routes to live in v1.
//   - Every FilterGroupBy key is a meter dimension, mirroring the live path's
//     validation (unknown keys — including "subject" — are a 400 on the live
//     path; routing them live preserves that behavior instead of silently
//     accepting them or turning the 400 into a 500). customer_id stays excluded
//     even if a meter defines it as a dimension key, because it aliases the
//     derived top-level column on the live path.
//   - The feature gate permits the namespace (checked LAST — EvaluateBool may be
//     a remote lookup, LRU-cached per (flag, namespace), so only otherwise-
//     cacheable queries pay for it). By default the gate is enabled for every
//     namespace; a configured feature-gate backend can restrict it. An
//     evaluation error routes to live — never cache on a gate failure.
func (c *Connector) canQueryBeCached(namespace string, meter meterpkg.Meter, params streaming.QueryParams) bool {
	if !c.config.QueryCacheEnabled {
		return false
	}

	if !c.config.EnableDecimalPrecision {
		return false
	}

	if !params.Cachable {
		return false
	}

	if params.ClientID != nil {
		return false
	}

	if params.From == nil {
		return false
	}

	if params.To == nil {
		return false
	}

	// Effective from merges the meter's EventFrom (matches queryMeter.from()).
	from := *params.From
	if meter.EventFrom != nil && meter.EventFrom.After(from) {
		from = *meter.EventFrom
	}

	to := *params.To

	// The queried range must be old enough to have settled windows to serve.
	// If from is younger than the freshness horizon there is nothing to cache.
	horizonCutoff := time.Now().UTC().Add(-c.config.QueryCacheMinimumCacheableUsageAge)
	if !from.Before(horizonCutoff) {
		return false
	}

	if to.Sub(from) < c.config.QueryCacheMinimumCacheableQueryPeriod {
		return false
	}

	switch meter.Aggregation {
	case meterpkg.MeterAggregationSum,
		meterpkg.MeterAggregationCount,
		meterpkg.MeterAggregationMin,
		meterpkg.MeterAggregationMax:
		// mergeable
	default:
		return false
	}

	if !isCacheableWindowSize(params.WindowSize) {
		return false
	}

	if !isWholeHourTimeZone(params.WindowTimeZone, from, to) {
		return false
	}

	if params.FilterStoredAt != nil && !params.FilterStoredAt.IsEmpty() {
		return false
	}

	if len(params.FilterCustomer) > 0 {
		return false
	}
	if slices.Contains(params.GroupBy, "customer_id") {
		return false
	}

	// Every GroupBy key must be "subject" or a meter dimension, mirroring the
	// live path's validation: an unknown key is a validation error (400) there,
	// and routing it live preserves that instead of turning it into a
	// cached-path SQL-build error (500).
	for _, key := range params.GroupBy {
		if key == "subject" {
			continue
		}
		if _, ok := meter.GroupBy[key]; !ok {
			return false
		}
	}

	for key := range params.FilterGroupBy {
		if _, ok := meter.GroupBy[key]; !ok {
			return false
		}
		if key == "customer_id" {
			return false
		}
	}

	enabled, err := c.config.FeatureGate.Enabled(namespace, queryCacheFeatureFlag)
	if err != nil {
		c.config.Logger.Warn("query cache feature gate evaluation failed, serving live", "error", err, "namespace", namespace)
		return false
	}

	return enabled
}

// isCacheableWindowSize reports whether the requested window size re-aggregates
// from the hourly rollup. Minute is the only excluded size (finer than the
// grain); nil (total) and hour/day/month all compose from hourly rows.
func isCacheableWindowSize(ws *meterpkg.WindowSize) bool {
	if ws == nil {
		return true
	}
	switch *ws {
	case meterpkg.WindowSizeHour, meterpkg.WindowSizeDay, meterpkg.WindowSizeMonth:
		return true
	default:
		return false
	}
}

// isWholeHourTimeZone reports whether the zone's offset from UTC is a whole
// number of hours for the entire [from, to) range. Hourly-UTC windows can only
// be re-tumbled into day/month windows in a target zone when the zone boundary
// falls on a UTC hour boundary; fractional-hour zones (India +05:30, Nepal
// +05:45) break this.
//
// DST is handled conservatively: we sample the offset at from, at to, and at
// every day boundary in between, and require every sampled offset to be a whole
// number of hours. A zone that is whole-hour year-round (e.g. Europe/* is always
// a whole hour even across DST) passes; a zone that is ever fractional in the
// range fails. UTC and a nil zone always pass.
func isWholeHourTimeZone(loc *time.Location, from, to time.Time) bool {
	if loc == nil || loc == time.UTC || loc.String() == "UTC" {
		return true
	}

	isWholeHour := func(t time.Time) bool {
		_, offsetSeconds := t.In(loc).Zone()
		return offsetSeconds%3600 == 0
	}

	if !isWholeHour(from) || !isWholeHour(to) {
		return false
	}

	// Sample each day boundary to catch a DST transition into a fractional
	// offset within the range. Bounded by the query span; cacheable queries are
	// finite. Step by 24h from from up to to.
	for t := from; t.Before(to); t = t.Add(24 * time.Hour) {
		if !isWholeHour(t) {
			return false
		}
	}

	return true
}

// prepareCacheableCutoff returns the freshness boundary that splits a cacheable
// query into settled history [from, cutoff) served from the cache and a fresh
// tail [cutoff, to) served live. The cutoff is now - MinimumCacheableUsageAge,
// floored to the hour grid so it aligns with stored hourly windows and the tail
// leg's tumble boundaries.
//
// Clamped to [from, to]: if the whole range is older than the horizon the tail
// is empty (all-cached); if the whole range is fresher than the horizon the
// cache leg is empty (all-live) — though canQueryBeCached already rejects a
// wholly-fresh range, so in practice cutoff > from here.
func (c *Connector) prepareCacheableCutoff(query queryMeter) time.Time {
	now := time.Now().UTC()
	cutoff := now.Add(-c.config.QueryCacheMinimumCacheableUsageAge).Truncate(cacheGrain)

	from := query.from()
	if from != nil && cutoff.Before(*from) {
		cutoff = *from
	}

	if query.To != nil && cutoff.After(*query.To) {
		cutoff = *query.To
	}

	return cutoff
}
