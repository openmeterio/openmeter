package clickhouse

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func auditMeter(agg meterpkg.MeterAggregation, groupBy map[string]string) meterpkg.Meter {
	vp := "$.value"
	m := meterpkg.Meter{
		Key:           "my_meter",
		Aggregation:   agg,
		EventType:     "my_event",
		ValueProperty: &vp,
		GroupBy:       groupBy,
	}
	return m
}

func dumpShape(t *testing.T, name string, q queryCachedMeter) {
	sql, args, err := q.toSQL()
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	nPlace := strings.Count(sql, "?")
	fmt.Printf("\n===== %s =====\n", name)
	fmt.Printf("placeholders=%d args=%d\n", nPlace, len(args))
	fmt.Printf("SQL:\n%s\n", sql)
	fmt.Printf("ARGS: %#v\n", args)
	if nPlace != len(args) {
		t.Errorf("%s: PLACEHOLDER/ARG MISMATCH placeholders=%d args=%d", name, nPlace, len(args))
	}
}

func TestAuditPlaceholderArgParity(t *testing.T) {
	from := time.Date(2026, 3, 1, 0, 30, 0, 0, time.UTC) // mid-hour -> head leg present
	to := time.Date(2026, 3, 5, 12, 30, 0, 0, time.UTC)  // mid-hour -> tail leg present
	ws := meterpkg.WindowSizeDay

	base := func(mod func(*queryMeter)) queryMeter {
		q := queryMeter{
			Database:               "openmeter",
			EventsTableName:        "om_events",
			Namespace:              "ns_A",
			Meter:                  auditMeter(meterpkg.MeterAggregationSum, map[string]string{"region": "$.region"}),
			From:                   &from,
			To:                     &to,
			WindowSize:             &ws,
			EnableDecimalPrecision: true,
		}
		if mod != nil {
			mod(&q)
		}
		return q
	}

	// Cutoff mid-range so cache leg + head + tail all present.
	cutoff := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)

	// Shape 1: window=day, cache+head+tail, no filters
	dumpShape(t, "day_cache_head_tail_nofilter", queryCachedMeter{
		Database: "openmeter", CacheTableName: meterQueryRowCacheTable, EventsTableName: "om_events",
		query: base(nil), Cutoff: cutoff,
	})

	// Shape 2: with subject filter
	dumpShape(t, "day_subject_filter", queryCachedMeter{
		Database: "openmeter", CacheTableName: meterQueryRowCacheTable, EventsTableName: "om_events",
		query: base(func(q *queryMeter) { q.FilterSubject = []string{"s1", "s2"} }), Cutoff: cutoff,
	})

	// Shape 3: with group-by filter
	dumpShape(t, "day_groupby_filter", queryCachedMeter{
		Database: "openmeter", CacheTableName: meterQueryRowCacheTable, EventsTableName: "om_events",
		query: base(func(q *queryMeter) {
			q.GroupBy = []string{"region"}
			q.FilterGroupBy = map[string]filter.FilterString{
				"region": {Eq: strptr("us")},
			}
		}), Cutoff: cutoff,
	})

	// Shape 4: total window (nil) - outer window args present
	dumpShape(t, "total_window", queryCachedMeter{
		Database: "openmeter", CacheTableName: meterQueryRowCacheTable, EventsTableName: "om_events",
		query: base(func(q *queryMeter) { q.WindowSize = nil }), Cutoff: cutoff,
	})

	// Shape 5: total window + subject filter + groupby filter (max placeholders)
	dumpShape(t, "total_subject_and_groupby_filter", queryCachedMeter{
		Database: "openmeter", CacheTableName: meterQueryRowCacheTable, EventsTableName: "om_events",
		query: base(func(q *queryMeter) {
			q.WindowSize = nil
			q.FilterSubject = []string{"s1"}
			q.GroupBy = []string{"region"}
			q.FilterGroupBy = map[string]filter.FilterString{"region": {Eq: strptr("us")}}
		}), Cutoff: cutoff,
	})

	// Shape 6: from hour-aligned (no head), cutoff hour-aligned but < to (tail present)
	fromAligned := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	dumpShape(t, "no_head_tail_present", queryCachedMeter{
		Database: "openmeter", CacheTableName: meterQueryRowCacheTable, EventsTableName: "om_events",
		query: base(func(q *queryMeter) { q.From = &fromAligned }), Cutoff: cutoff,
	})

	// Shape 7: fully cached (cutoff >= to, floored) - cache only, no head/tail? head present (mid-hour from)
	cutoffLate := to
	dumpShape(t, "cutoff_at_to", queryCachedMeter{
		Database: "openmeter", CacheTableName: meterQueryRowCacheTable, EventsTableName: "om_events",
		query: base(nil), Cutoff: cutoffLate,
	})
}

func strptr(s string) *string { return &s }

// TestAuditLiveTenantBinding executes the REAL generated merge SQL against live
// ClickHouse with positional args, proving the namespace placeholder binds the
// namespace value (no positional desync) and that a ns_A query never returns
// ns_B / other-type / other-meter / other-hash rows.
func TestAuditLiveTenantBinding(t *testing.T) {
	dsn := os.Getenv("TEST_CLICKHOUSE_DSN")
	if dsn == "" {
		t.Skip("TEST_CLICKHOUSE_DSN not set")
	}
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx := t.Context()

	// Hermetic database: CI ClickHouse has no pre-existing openmeter schema, so
	// the test creates everything it touches (events table for the live legs,
	// cache table for the seeded rows) via the production DDL builders.
	db := "zz_audit_tenant_binding"
	if err := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+db+" SYNC"); err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec(ctx, "CREATE DATABASE "+db); err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec(ctx, createEventsTable{Database: db, EventsTableName: "om_events"}.toSQL()); err != nil {
		t.Fatal(err)
	}

	// Build the total-window merge for ns_A over self-seeded cache rows: the
	// target row for ns_A plus decoys differing ONLY in namespace, type, or
	// meter_hash. With no matching events in om_events the live legs contribute
	// nothing, so the result is exactly the cache leg's value for ns_A — any
	// positional-arg desync or missing key filter surfaces as a decoy value.
	from := time.Date(2026, 3, 1, 0, 30, 0, 0, time.UTC)
	to := time.Date(2026, 3, 5, 12, 30, 0, 0, time.UTC)
	cutoff := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)

	q := queryMeter{
		Database:               db,
		EventsTableName:        "om_events",
		Namespace:              "ns_A",
		Meter:                  auditMeter(meterpkg.MeterAggregationSum, map[string]string{"region": "$.region"}),
		From:                   &from,
		To:                     &to,
		WindowSize:             nil, // total
		EnableDecimalPrecision: true,
	}
	hash := meterShapeHash(q.Meter)
	t.Logf("meter_hash=%s", hash)

	// Seed: create the cache table, then insert the target row and the decoys
	// under a windowstart inside [cacheLo, cacheHi).
	if err := conn.Exec(ctx, createMeterQueryRowCacheTable{Database: db, TableName: meterQueryRowCacheTable}.toSQL()); err != nil {
		t.Fatal(err)
	}
	table := getTableName(db, meterQueryRowCacheTable)
	windowstart := time.Date(2026, 3, 1, 1, 0, 0, 0, time.UTC) // = ceil(from, 1h)
	for _, seed := range []struct {
		namespace string
		eventType string
		hash      string
		value     int
	}{
		{"ns_A", "my_event", hash, 111},               // the target
		{"ns_B", "my_event", hash, 999},               // other tenant
		{"ns_A", "other_event", hash, 555},            // other type
		{"ns_A", "my_event", "deadbeef00000000", 333}, // other shape hash
	} {
		if err := conn.Exec(ctx, fmt.Sprintf(
			`INSERT INTO %s (namespace, type, meter_slug, meter_hash, windowstart, subject, group_by, sum_value, count_value, min_value, max_value)
			 VALUES (?, ?, 'my_meter', ?, ?, 's1', ['us'], toDecimal128(%d, 19), 0, NULL, NULL)`, table, seed.value),
			seed.namespace, seed.eventType, seed.hash, windowstart); err != nil {
			t.Fatal(err)
		}
	}

	merge := queryCachedMeter{
		Database: db, CacheTableName: meterQueryRowCacheTable, EventsTableName: "om_events",
		query: q, Cutoff: cutoff,
	}
	sql, args, err := merge.toSQL()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ARGS: %#v", args)

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		t.Fatalf("query: %v\nSQL:\n%s", err, sql)
	}
	defer rows.Close()

	var total float64
	n := 0
	for rows.Next() {
		var ws, we time.Time
		var v NullDecimal
		if err := rows.Scan(&ws, &we, &v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if v.Valid {
			total += v.Decimal.InexactFloat64()
		}
		n++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("ns_A total=%v rows=%d", total, n)
	if total != 111 {
		t.Fatalf("TENANT LEAK or desync: expected ns_A value 111, got %v", total)
	}
}

func TestAuditPopulateArgs(t *testing.T) {
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	cutoff := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	q := queryMeter{
		Database: "openmeter", EventsTableName: "om_events", Namespace: "ns_A",
		Meter:                  auditMeter(meterpkg.MeterAggregationSum, map[string]string{"region": "$.region"}),
		From:                   &from,
		EnableDecimalPrecision: true,
	}
	p := populateMeterQueryRowCache{
		Database: "openmeter", CacheTableName: meterQueryRowCacheTable, EventsTableName: "om_events",
		query: q, From: from, Cutoff: cutoff,
	}
	sql, args := p.toSQL()
	nPlace := strings.Count(sql, "?")
	fmt.Printf("\n===== POPULATE =====\nplaceholders=%d args=%d\nSQL:\n%s\nARGS: %#v\n", nPlace, len(args), sql, args)
	if nPlace != len(args) {
		t.Errorf("POPULATE PLACEHOLDER/ARG MISMATCH placeholders=%d args=%d", nPlace, len(args))
	}
}
