package clickhouse

import (
	"github.com/huandu/go-sqlbuilder"
)

const (
	// meterCacheTableName is the shared rollup table all per-meter refreshable
	// materialized views append into.
	meterCacheTableName = "om_meter_cache"
	// meterCacheInvalidationsTableName holds late-event invalidation markers the reader
	// consults before trusting cached buckets.
	meterCacheInvalidationsTableName = "om_meter_cache_invalidations"
	// meterCacheMVNamePrefix is the naming prefix of every generated cache MV. The
	// lifecycle reconciler discovers the actual MV set by scanning system.tables for this
	// prefix, so all cache MV names must be produced by mvName.
	meterCacheMVNamePrefix = "om_meter_cache_mv_"
)

// createMeterCacheTable creates the shared meter cache rollup table.
//
// The engine is ReplacingMergeTree(created_at) on purpose: refreshes and backfills
// re-append full recomputations of dirty buckets, and readers pick the newest version per
// bucket (argMax over created_at), which makes overlapping backfills, retried refreshes,
// and late-event recomputes idempotent. An AggregatingMergeTree target would silently
// double-count re-appended additive columns instead.
//
// There is deliberately no TTL on created_at: refreshes only re-append dirty buckets, so
// settled rows keep their old versions forever; a version TTL would eventually delete the
// only remaining copy of valid data. Orphaned meter shapes are garbage collected by the
// lifecycle reconciler instead.
type createMeterCacheTable struct {
	Database string
}

func (d createMeterCacheTable) toSQL() string {
	sb := sqlbuilder.ClickHouse.NewCreateTableBuilder()
	sb.CreateTable(getTableName(d.Database, meterCacheTableName))
	sb.IfNotExists()
	sb.Define("namespace", "String")
	sb.Define("meter_key", "LowCardinality(String)")
	// meter_hash identifies the cached shape (event type, aggregation, value property,
	// group-by dimensions, grain). Every read filters on it, so rows written for an old
	// shape or grain are never co-read with rows for the current one.
	sb.Define("meter_hash", "UInt64")
	sb.Define("windowstart", "DateTime")
	sb.Define("subject", "String")
	// The meter's configured group-by dimension values in sorted key order. The key list
	// is part of meter_hash, so positional decoding on the read side is unambiguous.
	sb.Define("group_by", "Array(String)")
	sb.Define("created_at", "DateTime64(3)")
	// sum/min/max are Nullable so a bucket whose values are all JSON null stays NULL and
	// is dropped on read exactly like the live query drops it; non-null zero would
	// fabricate usage. The counts are non-null: an all-null bucket legitimately counts 0.
	sb.Define("sum_value", "Nullable(Decimal128(19))")
	sb.Define("count_value", "UInt64")
	// value_count is the non-null value count, the AVG denominator. It is distinct from
	// count_value (count of all events) because events missing the value property must
	// not deflate the average.
	sb.Define("value_count", "UInt64")
	sb.Define("min_value", "Nullable(Decimal128(19))")
	sb.Define("max_value", "Nullable(Decimal128(19))")
	// The state parameter is Nullable(String), not String: the shared value expression
	// (rawStringValueExpr) yields Nullable(String) so explicit JSON nulls are skipped,
	// and ClickHouse rejects inserting a Nullable-argument aggregate state into a
	// non-Nullable-argument state column (verified on 25.12: CANNOT_CONVERT_TYPE).
	sb.Define("uniq_state", "AggregateFunction(uniqExact, Nullable(String))")
	sb.SQL("ENGINE = ReplacingMergeTree(created_at)")
	sb.SQL("ORDER BY (namespace, meter_key, meter_hash, windowstart, subject, group_by)")

	sql, _ := sb.Build()
	return sql
}

// createMeterCacheInvalidationsTable creates the late-event invalidation marker table.
//
// created_at is a server-side DEFAULT, never a client-supplied timestamp: the reader's
// heal rule compares marker time against refresh times reported by ClickHouse
// (system.view_refreshes), and mixing an app clock into one side of that comparison would
// let clock skew mark stale buckets as healed. Writers must therefore omit created_at on
// insert.
//
// Markers only matter until a refresh provably re-covered their window, so a 7 day TTL
// bounds the table; gaps older than that are repaired by the reconciler, not by markers.
type createMeterCacheInvalidationsTable struct {
	Database string
}

func (d createMeterCacheInvalidationsTable) toSQL() string {
	sb := sqlbuilder.ClickHouse.NewCreateTableBuilder()
	sb.CreateTable(getTableName(d.Database, meterCacheInvalidationsTableName))
	sb.IfNotExists()
	sb.Define("namespace", "String")
	sb.Define("event_type", "LowCardinality(String)")
	sb.Define("window_lo", "DateTime")
	sb.Define("window_hi", "DateTime")
	sb.Define("created_at", "DateTime64(3) DEFAULT now64(3)")
	sb.SQL("ENGINE = MergeTree")
	sb.SQL("ORDER BY (namespace, event_type, window_lo)")
	sb.SQL("TTL toDateTime(created_at) + toIntervalDay(7)")

	sql, _ := sb.Build()
	return sql
}
