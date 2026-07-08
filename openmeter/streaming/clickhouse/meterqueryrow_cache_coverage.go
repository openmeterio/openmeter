package clickhouse

import (
	"fmt"
	"strings"
	"time"
)

// meterQueryRowCacheCoverageTable records, per meter shape, the contiguous
// settled range whose hourly rollups are already present in the cache table,
// so a read can populate only the missing prefix/suffix of its own range
// instead of re-scanning raw events for hours that are already rolled up.
const meterQueryRowCacheCoverageTable = "meterqueryrow_cache_coverage"

// cacheCoverageMarkerSlug is the reserved meter_slug of per-namespace
// invalidation marker rows in the coverage table. It can never collide with a
// real claim: meter keys are validated non-empty.
const cacheCoverageMarkerSlug = ""

// cacheCoverageClockSkewMargin pads the claim-vs-marker comparison. A claim's
// populated_at is stamped with the app server's clock while the marker's
// created_at is stamped by ClickHouse, so honoring a claim requires
// populated_at to beat the marker by more than any realistic clock skew.
// Over-distrusting costs one redundant (idempotent) populate; under-distrusting
// would serve a wiped range as empty.
const cacheCoverageClockSkewMargin = 5 * time.Second

// cacheCoverageTrustWindow bounds how long a coverage claim is honored,
// measured from the FIRST populate that established the interval. The cache
// table's TTL expires rollup rows 90 days after their created_at, and gap
// tracking means covered rows are no longer rewritten (and thus refreshed) on
// every read — so the oldest rows under a claim expire at
// first_written_at + 90d while the claim would still assert coverage, silently
// undercounting the settled range. Distrusting the claim one day earlier
// forces a full re-populate (which rewrites the rows and resets the clock)
// before any row can expire. If this window did not exist, a meter queried for
// longer than 90 days over the same coverage interval would lose its oldest
// windows.
const cacheCoverageTrustWindow = 89 * 24 * time.Hour

// createMeterQueryRowCacheCoverageTable builds the DDL for the coverage table.
//
// Storage decisions:
//   - One row per (namespace, meter_slug, meter_hash), same identity as the
//     rollup rows it describes: a meter shape change orphans both the rollups
//     and their claim together.
//   - ReplacingMergeTree(created_at) + read-time argMax: extensions simply
//     insert a new row and the newest interval wins; racing readers can only
//     UNDER-report coverage (each claims what it verified populated), which
//     costs a redundant populate, never a wrong result.
//   - first_written_at is carried forward on extension (it tracks the OLDEST
//     rollup rows under the claim, which expire first — see
//     cacheCoverageTrustWindow).
//   - TTL matches the rollup table: an expired claim is just absent coverage
//     and the next read re-populates, so expiry here is always safe.
type createMeterQueryRowCacheCoverageTable struct {
	Database  string
	TableName string
}

func (t createMeterQueryRowCacheCoverageTable) toSQL() string {
	tableName := getTableName(t.Database, t.TableName)

	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	namespace String,
	meter_slug String,
	meter_hash String,
	covered_from DateTime,
	covered_until DateTime,
	first_written_at DateTime,
	populated_at DateTime64(3),
	created_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = ReplacingMergeTree(created_at)
ORDER BY (namespace, meter_slug, meter_hash)
TTL toDateTime(created_at) + toIntervalDay(90)`, tableName)
}

// cacheCoverage is the newest stored coverage claim for one meter shape:
// hourly rollups for [From, Until) are present in the cache table, the oldest
// of them written at FirstWrittenAt. PopulatedAt is the claiming read's
// PLAN-START time (app-server clock, captured before it read the previous
// claim and ran its populates): a claim is honored only when PopulatedAt
// postdates the namespace's newest invalidation marker, which is what lets an
// invalidation win against a claim INSERT that lands after its deletes.
type cacheCoverage struct {
	From           time.Time
	Until          time.Time
	FirstWrittenAt time.Time
	PopulatedAt    time.Time
}

// timeRange is a half-open [From, To) hour-aligned populate target.
type timeRange struct {
	From time.Time
	To   time.Time
}

// cachePlan is the outcome of planning a cached read against the stored
// coverage: the sub-ranges of the query's settled window that must be
// populated, and the coverage claim to store once they succeed (nil = leave
// the stored claim untouched).
type cachePlan struct {
	Populate []timeRange
	Store    *cacheCoverage
}

// planCachePopulation decides which parts of the settled range [lo, hi) must
// be rolled up given the stored coverage claim (nil if none) and the
// namespace's newest invalidation marker (zero if none). `now` is the calling
// read's plan-start time and becomes the stored claim's PopulatedAt. Every
// branch preserves the invariant that a stored claim only ever describes rows
// whose populate has committed:
//
//   - No claim, a claim older than the trust window, or a claim whose
//     PopulatedAt does not clearly postdate the invalidation marker (it may
//     describe rows the invalidation wiped): populate the whole range and
//     start a fresh claim (re-populating over surviving rows is harmless —
//     newest created_at wins the read-time collapse — and resets their TTL
//     clock).
//   - Claim disjoint from the queried range: populate the whole range but keep
//     the stored claim, because a single contiguous interval cannot describe
//     both. The read is still correct (its rows exist after this populate);
//     it just isn't remembered, which is the v1 behavior for that range.
//   - Overlapping or adjacent claim: populate only the missing prefix and/or
//     suffix and extend the claim to the union. A fully covered range
//     populates nothing and stores nothing (no write amplification on the hot
//     path).
func planCachePopulation(lo, hi time.Time, cov *cacheCoverage, invalidatedAt time.Time, now time.Time) cachePlan {
	if cov != nil && now.After(cov.FirstWrittenAt.Add(cacheCoverageTrustWindow)) {
		cov = nil
	}
	if cov != nil && !invalidatedAt.IsZero() && !cov.PopulatedAt.After(invalidatedAt.Add(cacheCoverageClockSkewMargin)) {
		cov = nil
	}

	if cov == nil {
		return cachePlan{
			Populate: []timeRange{{From: lo, To: hi}},
			Store:    &cacheCoverage{From: lo, Until: hi, FirstWrittenAt: now, PopulatedAt: now},
		}
	}

	// Disjoint (not even adjacent): [lo,hi) ends before the claim starts or
	// starts after it ends.
	if hi.Before(cov.From) || lo.After(cov.Until) {
		return cachePlan{
			Populate: []timeRange{{From: lo, To: hi}},
		}
	}

	plan := cachePlan{}
	if lo.Before(cov.From) {
		plan.Populate = append(plan.Populate, timeRange{From: lo, To: cov.From})
	}
	if hi.After(cov.Until) {
		plan.Populate = append(plan.Populate, timeRange{From: cov.Until, To: hi})
	}

	if len(plan.Populate) > 0 {
		merged := &cacheCoverage{From: cov.From, Until: cov.Until, FirstWrittenAt: cov.FirstWrittenAt, PopulatedAt: now}
		if lo.Before(merged.From) {
			merged.From = lo
		}
		if hi.After(merged.Until) {
			merged.Until = hi
		}
		plan.Store = merged
	}

	return plan
}

// getMeterQueryRowCacheCoverage builds the newest-wins lookup of one meter
// shape's coverage claim AND the namespace's invalidation marker in a single
// round trip (grouped by meter_slug; the marker lives under the reserved empty
// slug). Absence of a group means "none stored".
//
// The claim fields are picked through ONE argMax over a tuple, never
// independent argMax calls: with per-column argMax, a created_at tie between
// two racing claims (e.g. disjoint first-claims landing in the same
// millisecond) could stitch covered_from from one row and covered_until from
// the other, asserting coverage over the gap between them that no populate
// ever wrote — a silent undercount. The tuple makes the pick atomic: whichever
// row wins the tie, the claim describes rows that writer actually populated.
type getMeterQueryRowCacheCoverage struct {
	Database  string
	TableName string
	Namespace string
	Meter     string
	Hash      string
}

func (g getMeterQueryRowCacheCoverage) toSQL() (string, []interface{}) {
	tableName := getTableName(g.Database, g.TableName)

	sql := fmt.Sprintf(`SELECT
	meter_slug,
	tupleElement(argMax(tuple(covered_from, covered_until, first_written_at, populated_at), created_at), 1),
	tupleElement(argMax(tuple(covered_from, covered_until, first_written_at, populated_at), created_at), 2),
	tupleElement(argMax(tuple(covered_from, covered_until, first_written_at, populated_at), created_at), 3),
	tupleElement(argMax(tuple(covered_from, covered_until, first_written_at, populated_at), created_at), 4),
	max(created_at)
FROM %s
WHERE namespace = ? AND ((meter_slug = ? AND meter_hash = ?) OR (meter_slug = '' AND meter_hash = ''))
GROUP BY meter_slug`, tableName)

	return sql, []interface{}{g.Namespace, g.Meter, g.Hash}
}

// insertMeterQueryRowCacheCoverage builds the claim upsert (a plain insert;
// ReplacingMergeTree + argMax reads make the newest row the effective claim).
type insertMeterQueryRowCacheCoverage struct {
	Database  string
	TableName string
	Namespace string
	Meter     string
	Hash      string
	Coverage  cacheCoverage
}

func (i insertMeterQueryRowCacheCoverage) toSQL() (string, []interface{}) {
	tableName := getTableName(i.Database, i.TableName)

	sql := fmt.Sprintf(`INSERT INTO %s (namespace, meter_slug, meter_hash, covered_from, covered_until, first_written_at, populated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`, tableName)

	return sql, []interface{}{
		i.Namespace, i.Meter, i.Hash,
		i.Coverage.From.Unix(), i.Coverage.Until.Unix(), i.Coverage.FirstWrittenAt.Unix(),
		i.Coverage.PopulatedAt.UTC(),
	}
}

// insertMeterQueryRowCacheInvalidationMarkers builds the per-namespace
// invalidation marker inserts. The marker (not the claim DELETE) is what makes
// invalidation win against in-flight claim writers: a racing claim INSERT can
// always land after any DELETE, but its populated_at was captured at plan
// start and therefore predates the marker's created_at, so every read
// distrusts it. Markers live under the reserved empty slug and must survive
// claim deletion; the table TTL garbage-collects them long after any claim
// that could predate them has aged out of the trust window.
type insertMeterQueryRowCacheInvalidationMarkers struct {
	Database   string
	TableName  string
	Namespaces []string
}

func (i insertMeterQueryRowCacheInvalidationMarkers) toSQL() (string, []interface{}) {
	tableName := getTableName(i.Database, i.TableName)

	var values []string
	args := make([]interface{}, 0, len(i.Namespaces))
	for _, ns := range i.Namespaces {
		values = append(values, "(?, '', '', 0, 0, 0, 0)")
		args = append(args, ns)
	}

	sql := fmt.Sprintf(`INSERT INTO %s (namespace, meter_slug, meter_hash, covered_from, covered_until, first_written_at, populated_at)
VALUES %s`, tableName, strings.Join(values, ", "))

	return sql, args
}

// deleteMeterQueryRowCacheCoverageClaims builds the claim cleanup for the
// given namespaces. Markers (the reserved empty slug) are deliberately
// excluded: they must outlive the deletion to kill racing claim writers.
type deleteMeterQueryRowCacheCoverageClaims struct {
	Database   string
	TableName  string
	Namespaces []string
}

func (d deleteMeterQueryRowCacheCoverageClaims) toSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.TableName)

	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(d.Namespaces)), ", ")

	sql := fmt.Sprintf("DELETE FROM %s WHERE namespace IN (%s) AND meter_slug != ''", tableName, placeholders)

	args := make([]interface{}, 0, len(d.Namespaces))
	for _, ns := range d.Namespaces {
		args = append(args, ns)
	}

	return sql, args
}
