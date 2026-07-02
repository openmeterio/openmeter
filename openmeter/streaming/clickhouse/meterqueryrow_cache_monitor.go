package clickhouse

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

// parityCheckTimeout bounds the detached shadow verification query. It is
// generous on purpose: the shadow runs the FULL live query (the expensive scan
// the cache exists to avoid), off the request path.
const parityCheckTimeout = 2 * time.Minute

// queryCacheMetrics holds the OTel instruments that make the cache's health and
// correctness observable in production. All methods are nil-receiver safe so a
// connector constructed without a metric.Meter (tests, embedded use) pays
// nothing.
//
//   - streaming.query_cache.queries{result}: cached | live_fallback — how much
//     traffic the cache actually serves, and how often it degrades to live.
//   - streaming.query_cache.populate_errors: rollup INSERT failures (each one
//     also forced a live fallback).
//   - streaming.query_cache.invalidations{reason}: late_event | parity_mismatch
//     namespace wipes.
//   - streaming.query_cache.parity_checks{outcome}: match | mismatch | error —
//     the shadow verification results. ALERT ON mismatch > 0: it means a
//     cache-served result diverged from raw data.
//   - streaming.query_cache.query_duration_ms / populate_duration_ms: latency of
//     the merge query and of the rollup population — the two heavyweight SQL
//     operations; the cache pays off only if the first stays well under the live
//     scan it replaces.
type queryCacheMetrics struct {
	queries          metric.Int64Counter
	populateErrors   metric.Int64Counter
	invalidations    metric.Int64Counter
	parityChecks     metric.Int64Counter
	queryDuration    metric.Int64Histogram
	populateDuration metric.Int64Histogram
}

func newQueryCacheMetrics(meter metric.Meter) (*queryCacheMetrics, error) {
	queries, err := meter.Int64Counter(
		"streaming.query_cache.queries",
		metric.WithDescription("Number of meter queries admitted to the query cache, by result (cached, live_fallback)"),
	)
	if err != nil {
		return nil, fmt.Errorf("create query cache queries counter: %w", err)
	}

	populateErrors, err := meter.Int64Counter(
		"streaming.query_cache.populate_errors",
		metric.WithDescription("Number of failed query cache rollup populations"),
	)
	if err != nil {
		return nil, fmt.Errorf("create query cache populate errors counter: %w", err)
	}

	invalidations, err := meter.Int64Counter(
		"streaming.query_cache.invalidations",
		metric.WithDescription("Number of namespace-wide query cache invalidations, by reason (late_event, parity_mismatch)"),
	)
	if err != nil {
		return nil, fmt.Errorf("create query cache invalidations counter: %w", err)
	}

	parityChecks, err := meter.Int64Counter(
		"streaming.query_cache.parity_checks",
		metric.WithDescription("Number of sampled shadow parity checks of cache-served results against the live query, by outcome (match, mismatch, error)"),
	)
	if err != nil {
		return nil, fmt.Errorf("create query cache parity checks counter: %w", err)
	}

	queryDuration, err := meter.Int64Histogram(
		"streaming.query_cache.query_duration_ms",
		metric.WithDescription("Duration of the cache merge query (cache leg + live legs) in milliseconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("create query cache query duration histogram: %w", err)
	}

	populateDuration, err := meter.Int64Histogram(
		"streaming.query_cache.populate_duration_ms",
		metric.WithDescription("Duration of the query cache rollup population in milliseconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("create query cache populate duration histogram: %w", err)
	}

	return &queryCacheMetrics{
		queries:          queries,
		populateErrors:   populateErrors,
		invalidations:    invalidations,
		parityChecks:     parityChecks,
		queryDuration:    queryDuration,
		populateDuration: populateDuration,
	}, nil
}

func (m *queryCacheMetrics) recordQuery(ctx context.Context, result string) {
	if m == nil {
		return
	}
	m.queries.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func (m *queryCacheMetrics) recordPopulateError(ctx context.Context) {
	if m == nil {
		return
	}
	m.populateErrors.Add(ctx, 1)
}

func (m *queryCacheMetrics) recordInvalidation(ctx context.Context, reason string, namespaces int) {
	if m == nil {
		return
	}
	m.invalidations.Add(ctx, int64(namespaces), metric.WithAttributes(attribute.String("reason", reason)))
}

func (m *queryCacheMetrics) recordParityCheck(ctx context.Context, outcome string) {
	if m == nil {
		return
	}
	m.parityChecks.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", outcome)))
}

func (m *queryCacheMetrics) recordQueryDuration(ctx context.Context, elapsed time.Duration) {
	if m == nil {
		return
	}
	m.queryDuration.Record(ctx, elapsed.Milliseconds())
}

func (m *queryCacheMetrics) recordPopulateDuration(ctx context.Context, elapsed time.Duration) {
	if m == nil {
		return
	}
	m.populateDuration.Record(ctx, elapsed.Milliseconds())
}

// maybeShadowVerifyCachedResult samples cache-served results for shadow
// verification against the live query. The check runs in a detached goroutine
// (the caller's response is already on its way; cancellation of the request
// must not kill the verification) bounded by parityCheckTimeout.
//
// This is the production correctness net: unit and integration parity tests
// prove the design, the shadow check proves THIS deployment — catching whatever
// no test anticipated (replica lag, mutation races, operational surprises) at a
// configurable sampling cost of one extra live query per sampled request.
func (c *Connector) maybeShadowVerifyCachedResult(ctx context.Context, query queryMeter, cachedRows []meterpkg.MeterQueryRow) {
	rate := c.config.QueryCacheParityCheckSampleRate
	if rate <= 0 {
		return
	}

	if rate < 1 && rand.Float64() >= rate {
		return
	}

	// Detach from the request's cancellation but keep its values (tracing).
	detached, cancel := context.WithTimeout(context.WithoutCancel(ctx), parityCheckTimeout)

	go func() {
		defer cancel()

		c.verifyCachedResultParity(detached, query, cachedRows)
	}()
}

// verifyCachedResultParity re-runs the query on the live path and compares it
// to the cache-served rows on the same criteria as the parity test gate: the
// full (windowstart, windowend, subject, customer, group-by) tuple, values
// rounded to 6 decimals. On mismatch it reports loudly (error log + metric) and
// self-heals by invalidating the namespace's cache, so the next query
// repopulates from raw data. Returns whether a mismatch was found (for tests).
func (c *Connector) verifyCachedResultParity(ctx context.Context, query queryMeter, cachedRows []meterpkg.MeterQueryRow) bool {
	ctx, span := c.tracer.Start(ctx, "streaming.query_cache.parity_check", trace.WithAttributes(
		attribute.String("namespace", query.Namespace),
		attribute.String("meter_slug", query.Meter.Key),
	))
	defer span.End()

	logger := c.config.Logger.With(
		"namespace", query.Namespace,
		"meterSlug", query.Meter.Key,
		"from", query.From,
		"to", query.To,
	)

	liveRows, err := c.queryMeter(ctx, query)
	if err != nil {
		c.queryCacheMetrics.recordParityCheck(ctx, "error")
		span.SetAttributes(attribute.String("outcome", "error"))
		span.RecordError(err)
		logger.Warn("query cache parity check failed to run live query", "error", err)

		return false
	}

	// For the total (nil) window, QueryMeter overwrites windowstart/end with
	// From/To on whatever path served the request; normalize both sides the same
	// way so the comparison sees what callers see.
	cached := normalizeTotalWindowBounds(cachedRows, query)
	live := normalizeTotalWindowBounds(liveRows, query)

	diff := compareMeterQueryRowSets(live, cached)
	if diff == "" {
		c.queryCacheMetrics.recordParityCheck(ctx, "match")
		span.SetAttributes(attribute.String("outcome", "match"))

		return false
	}

	c.queryCacheMetrics.recordParityCheck(ctx, "mismatch")
	span.SetAttributes(attribute.String("outcome", "mismatch"))
	span.SetStatus(codes.Error, "cache-served result diverged from live query")
	logger.Error("QUERY CACHE PARITY MISMATCH: cache-served result diverged from live query, invalidating namespace cache", "diff", diff)

	if err := c.invalidateMeterQueryRowCache(ctx, []string{query.Namespace}); err != nil {
		span.RecordError(err)
		logger.Error("failed to invalidate meter query cache after parity mismatch", "error", err)

		return true
	}
	c.queryCacheMetrics.recordInvalidation(ctx, "parity_mismatch", 1)

	return true
}

// normalizeTotalWindowBounds applies QueryMeter's nil-window From/To overwrite
// to a copy of the rows, so cached and live results are compared in the shape
// callers actually receive.
func normalizeTotalWindowBounds(rows []meterpkg.MeterQueryRow, query queryMeter) []meterpkg.MeterQueryRow {
	out := append([]meterpkg.MeterQueryRow(nil), rows...)
	if query.WindowSize != nil {
		return out
	}

	for i := range out {
		if query.From != nil {
			out[i].WindowStart = *query.From
		}
		if query.To != nil {
			out[i].WindowEnd = *query.To
		}
	}

	return out
}

// compareMeterQueryRowSets returns "" when the two result sets are identical,
// keyed on the full row tuple with values compared to 6 decimal places —
// deliberately the same criteria as the parity test gate. Any difference
// returns a human-readable description for the mismatch log.
func compareMeterQueryRowSets(live, cached []meterpkg.MeterQueryRow) string {
	if len(live) != len(cached) {
		return fmt.Sprintf("row count differs: live=%d cached=%d", len(live), len(cached))
	}

	l := append([]meterpkg.MeterQueryRow(nil), live...)
	c := append([]meterpkg.MeterQueryRow(nil), cached...)
	sort.Slice(l, func(i, j int) bool { return meterQueryRowKey(l[i]) < meterQueryRowKey(l[j]) })
	sort.Slice(c, func(i, j int) bool { return meterQueryRowKey(c[i]) < meterQueryRowKey(c[j]) })

	for i := range l {
		lk, ck := meterQueryRowKey(l[i]), meterQueryRowKey(c[i])
		if lk != ck {
			return fmt.Sprintf("row key differs: live=%q cached=%q", lk, ck)
		}

		if math.Round(l[i].Value*1e6) != math.Round(c[i].Value*1e6) {
			return fmt.Sprintf("value differs for %q: live=%v cached=%v", lk, l[i].Value, c[i].Value)
		}
	}

	return ""
}

// meterQueryRowKey is the full grouping tuple of a result row: window bounds,
// subject, customer and every group-by dimension. Comparing on anything less
// would line up different groups' rows against each other.
func meterQueryRowKey(r meterpkg.MeterQueryRow) string {
	var b strings.Builder

	b.WriteString(r.WindowStart.UTC().Format(time.RFC3339))
	b.WriteByte(0)
	b.WriteString(r.WindowEnd.UTC().Format(time.RFC3339))
	b.WriteByte(0)
	if r.Subject != nil {
		b.WriteString(*r.Subject)
	}
	b.WriteByte(0)
	if r.CustomerID != nil {
		b.WriteString(*r.CustomerID)
	}
	b.WriteByte(0)

	keys := make([]string, 0, len(r.GroupBy))
	for k := range r.GroupBy {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte(1)
		if v := r.GroupBy[k]; v != nil {
			b.WriteString(*v)
		}
		b.WriteByte(2)
	}

	return b.String()
}
