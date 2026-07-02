package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// queryMeterCached serves a cacheable meter query. It lazily rolls up the
// settled whole-hour range into the cache table, then runs the merge query that
// UNIONs the settled rollup with live scans of the sub-hour head and fresh tail.
//
// Population is lazy, idempotent, and gap-tracked: the coverage table records
// the contiguous settled interval already rolled up per meter shape, so a read
// populates only the missing prefix/suffix of its own range (nothing at all on
// a fully covered repeat read). Idempotence is what makes the tracking safe
// rather than merely fast: because the rollup table is a ReplacingMergeTree
// and the cache leg is collapsed newest-wins at read time, a redundant
// populate — from a racing reader, a lost coverage claim, or an expired trust
// window — is always harmless.
func (c *Connector) queryMeterCached(ctx context.Context, query queryMeter) ([]meterpkg.MeterQueryRow, error) {
	cutoff := c.prepareCacheableCutoff(query)

	from := query.from()
	if from == nil {
		return nil, fmt.Errorf("cached query requires a from time")
	}

	merge := queryCachedMeter{
		Database:        c.config.Database,
		CacheTableName:  meterQueryRowCacheTable,
		EventsTableName: c.config.EventsTableName,
		query:           query,
		Cutoff:          cutoff,
	}

	// The cache stores ONLY complete hour windows [cacheLo, cacheHi): the sub-hour
	// head [from, cacheLo) and the partial last hour [cacheHi, to) are served live.
	// Populate exactly the settled whole-hour range the cache leg reads.
	cacheLo := merge.headCeil()
	cacheHi := merge.cacheHi()

	ctx, span := c.tracer.Start(ctx, "streaming.query_cache.query", trace.WithAttributes(
		attribute.String("namespace", query.Namespace),
		attribute.String("meter_slug", query.Meter.Key),
		attribute.String("cache_lo", cacheLo.UTC().Format(time.RFC3339)),
		attribute.String("cache_hi", cacheHi.UTC().Format(time.RFC3339)),
	))
	defer span.End()

	logger := c.config.Logger.With(
		"namespace", query.Namespace,
		"type", query.Meter.EventType,
		"from", from,
		"cacheLo", cacheLo,
		"cacheHi", cacheHi,
		"to", query.To,
		"cutoff", cutoff,
	)

	if cacheLo.Before(cacheHi) {
		plan := c.planCachedRangePopulation(ctx, query, cacheLo, cacheHi)
		span.SetAttributes(attribute.Int("populate_ranges", len(plan.Populate)))

		for _, r := range plan.Populate {
			if err := c.populateMeterQueryRowCache(ctx, query, r.From, r.To); err != nil {
				logger.Error("failed to populate meter query cache, falling back to live query", "error", err)
				c.queryCacheMetrics.recordPopulateError(ctx)
				c.queryCacheMetrics.recordQuery(ctx, "live_fallback")
				span.RecordError(err)
				span.SetAttributes(attribute.Bool("live_fallback", true))

				return c.queryMeter(ctx, query)
			}
		}

		// The claim is stored strictly AFTER every populate above committed, so
		// a claim can never describe rows that were not written. A failed claim
		// write is not a query failure: the rollups exist and serve this read;
		// the next read just re-populates redundantly (idempotent).
		if plan.Store != nil {
			if err := c.storeMeterQueryRowCacheCoverage(ctx, query, *plan.Store); err != nil {
				logger.Warn("failed to store meter query cache coverage", "error", err)
				span.RecordError(err)
			}
		}
	}

	sql, args, err := merge.toSQL()
	if err != nil {
		return nil, fmt.Errorf("build cached query sql: %w", err)
	}

	start := time.Now()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		span.SetStatus(codes.Error, "cached query failed")
		span.RecordError(err)

		return nil, fmt.Errorf("clickhouse cached query: %w", err)
	}
	defer rows.Close()

	elapsed := time.Since(start)
	c.queryCacheMetrics.recordQueryDuration(ctx, elapsed)
	logger.Debug("clickhouse cached query executed", "elapsed", elapsed.String(), "sql", sql, "args", args)

	values, err := merge.scanRows(rows)
	if err != nil {
		span.SetStatus(codes.Error, "cached query scan failed")
		span.RecordError(err)

		return nil, fmt.Errorf("scan cached query rows: %w", err)
	}

	c.queryCacheMetrics.recordQuery(ctx, "cached")
	span.SetAttributes(attribute.Int("rows", len(values)))

	c.maybeShadowVerifyCachedResult(ctx, query, values)

	return values, nil
}

// planCachedRangePopulation reads the stored coverage claim and the
// namespace's invalidation marker for the query's meter shape and plans which
// sub-ranges of [cacheLo, cacheHi) still need rolling up. The plan-start time
// is captured BEFORE the coverage read: it becomes the stored claim's
// PopulatedAt, so an invalidation marker landing at any later instant —
// including between our populates and our claim INSERT — makes the claim
// distrusted. A coverage read failure degrades to "no claim" (populate the
// whole range), which is always safe — never fail or under-populate the query
// because the metadata was unreadable.
func (c *Connector) planCachedRangePopulation(ctx context.Context, query queryMeter, cacheLo, cacheHi time.Time) cachePlan {
	planStart := time.Now().UTC()

	coverage, invalidatedAt, err := c.readMeterQueryRowCacheCoverage(ctx, query)
	if err != nil {
		c.config.Logger.Warn("failed to read meter query cache coverage, repopulating the full range", "error", err, "namespace", query.Namespace, "meter", query.Meter.Key)
		coverage = nil
	}

	return planCachePopulation(cacheLo, cacheHi, coverage, invalidatedAt, planStart)
}

// readMeterQueryRowCacheCoverage returns the meter shape's newest claim (nil
// if none) and the namespace's newest invalidation marker time (zero if none).
func (c *Connector) readMeterQueryRowCacheCoverage(ctx context.Context, query queryMeter) (*cacheCoverage, time.Time, error) {
	get := getMeterQueryRowCacheCoverage{
		Database:  c.config.Database,
		TableName: meterQueryRowCacheCoverageTable,
		Namespace: query.Namespace,
		Meter:     query.Meter.Key,
		Hash:      meterShapeHash(query.Meter),
	}

	sql, args := get.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("read cache coverage: %w", err)
	}
	defer rows.Close()

	var coverage *cacheCoverage
	var invalidatedAt time.Time

	for rows.Next() {
		var slug string
		var from, until, firstWrittenAt, populatedAt, newestCreatedAt time.Time
		if err := rows.Scan(&slug, &from, &until, &firstWrittenAt, &populatedAt, &newestCreatedAt); err != nil {
			return nil, time.Time{}, fmt.Errorf("scan cache coverage: %w", err)
		}

		if slug == cacheCoverageMarkerSlug {
			invalidatedAt = newestCreatedAt.UTC()
			continue
		}

		coverage = &cacheCoverage{
			From:           from.UTC(),
			Until:          until.UTC(),
			FirstWrittenAt: firstWrittenAt.UTC(),
			PopulatedAt:    populatedAt.UTC(),
		}
	}

	if err := rows.Err(); err != nil {
		return nil, time.Time{}, fmt.Errorf("read cache coverage rows: %w", err)
	}

	return coverage, invalidatedAt, nil
}

func (c *Connector) storeMeterQueryRowCacheCoverage(ctx context.Context, query queryMeter, coverage cacheCoverage) error {
	insert := insertMeterQueryRowCacheCoverage{
		Database:  c.config.Database,
		TableName: meterQueryRowCacheCoverageTable,
		Namespace: query.Namespace,
		Meter:     query.Meter.Key,
		Hash:      meterShapeHash(query.Meter),
		Coverage:  coverage,
	}

	sql, args := insert.toSQL()
	if err := c.config.ClickHouse.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("store cache coverage: %w", err)
	}

	return nil
}

func (c *Connector) populateMeterQueryRowCache(ctx context.Context, query queryMeter, from, cutoff time.Time) error {
	ctx, span := c.tracer.Start(ctx, "streaming.query_cache.populate", trace.WithAttributes(
		attribute.String("namespace", query.Namespace),
		attribute.String("meter_slug", query.Meter.Key),
		attribute.String("from", from.UTC().Format(time.RFC3339)),
		attribute.String("to", cutoff.UTC().Format(time.RFC3339)),
	))
	defer span.End()

	populate := populateMeterQueryRowCache{
		Database:        c.config.Database,
		CacheTableName:  meterQueryRowCacheTable,
		EventsTableName: c.config.EventsTableName,
		query:           query,
		From:            from,
		Cutoff:          cutoff,
	}

	start := time.Now()

	sql, args := populate.toSQL()
	if err := c.config.ClickHouse.Exec(ctx, sql, args...); err != nil {
		span.SetStatus(codes.Error, "populate insert failed")
		span.RecordError(err)

		return fmt.Errorf("populate cache insert: %w", err)
	}

	c.queryCacheMetrics.recordPopulateDuration(ctx, time.Since(start))

	return nil
}

// findNamespacesToInvalidateCache returns the distinct namespaces of any events
// whose time is older than the freshness horizon. Such an event mutates an
// already-settled window, so that namespace's cached rollup is now stale.
func (c *Connector) findNamespacesToInvalidateCache(rawEvents []streaming.RawEvent) []string {
	if c.config.QueryCacheMinimumCacheableUsageAge <= 0 {
		return nil
	}

	horizon := time.Now().UTC().Add(-c.config.QueryCacheMinimumCacheableUsageAge)

	seen := map[string]struct{}{}
	namespaces := []string{}

	for _, event := range rawEvents {
		if event.Time.Before(horizon) {
			if _, ok := seen[event.Namespace]; !ok {
				seen[event.Namespace] = struct{}{}
				namespaces = append(namespaces, event.Namespace)
			}
		}
	}

	return namespaces
}

// invalidateMeterQueryRowCache deletes all cached rows for the given namespaces.
func (c *Connector) invalidateMeterQueryRowCache(ctx context.Context, namespaces []string) error {
	if len(namespaces) == 0 {
		return nil
	}

	ctx, span := c.tracer.Start(ctx, "streaming.query_cache.invalidate", trace.WithAttributes(
		attribute.Int("namespaces", len(namespaces)),
	))
	defer span.End()

	// Step 1: insert the per-namespace invalidation markers FIRST. The marker,
	// not the claim DELETE, is what beats in-flight claim writers: a racing
	// claim INSERT can land after any DELETE, but its populated_at (captured at
	// plan start) predates the marker, so every read distrusts it. Once the
	// marker commits, all past and racing claims are dead even if the cleanup
	// deletes below fail — the next read re-populates, and its rows win the
	// newest-wins collapse over any stale survivors (this restores the
	// self-healing v1 had from populating on every read).
	markers := insertMeterQueryRowCacheInvalidationMarkers{
		Database:   c.config.Database,
		TableName:  meterQueryRowCacheCoverageTable,
		Namespaces: namespaces,
	}

	markerSQL, markerArgs := markers.toSQL()
	if err := c.config.ClickHouse.Exec(ctx, markerSQL, markerArgs...); err != nil {
		// A missing table (code 60) means nothing is cached: invalidation is
		// always-on even with the cache disabled.
		if !strings.Contains(err.Error(), "code: 60") {
			// Fall back to deleting the claims outright: that still kills every
			// COMMITTED claim (only a claim racing this exact invalidation could
			// survive, which is the pre-marker exposure). If the delete fails
			// too, report the error — stale claims then persist until the next
			// invalidation or trust-window expiry.
			delClaims := deleteMeterQueryRowCacheCoverageClaims{
				Database:   c.config.Database,
				TableName:  meterQueryRowCacheCoverageTable,
				Namespaces: namespaces,
			}

			delSQL, delArgs := delClaims.toSQL()
			if delErr := c.config.ClickHouse.Exec(ctx, delSQL, delArgs...); delErr != nil {
				span.SetStatus(codes.Error, "invalidation marker and claim delete failed")
				span.RecordError(err)

				return fmt.Errorf("insert invalidation markers: %w (claim delete fallback: %v)", err, delErr)
			}

			c.config.Logger.Warn("invalidation marker insert failed, fell back to deleting coverage claims", "error", err, "namespaces", namespaces)
		}
	}

	// Step 2 (cleanup, correctness no longer depends on it): drop the dead
	// claims and the rollup rows. Failures degrade to storage waste plus one
	// redundant repopulate, so they are logged but not propagated.
	claimsSQL, claimsArgs := deleteMeterQueryRowCacheCoverageClaims{
		Database:   c.config.Database,
		TableName:  meterQueryRowCacheCoverageTable,
		Namespaces: namespaces,
	}.toSQL()

	rollupSQL, rollupArgs := deleteMeterQueryRowCacheForNamespaces{
		Database:   c.config.Database,
		TableName:  meterQueryRowCacheTable,
		Namespaces: namespaces,
	}.toSQL()

	for _, cleanup := range []struct {
		sql  string
		args []interface{}
	}{
		{claimsSQL, claimsArgs},
		{rollupSQL, rollupArgs},
	} {
		if err := c.config.ClickHouse.Exec(ctx, cleanup.sql, cleanup.args...); err != nil {
			if strings.Contains(err.Error(), "code: 60") {
				continue
			}

			span.RecordError(err)
			c.config.Logger.Warn("invalidation cleanup delete failed (markers already invalidate the claims)", "error", err, "namespaces", namespaces)
		}
	}

	return nil
}
