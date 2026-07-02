package clickhouse

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// This file holds the billing-safety parity gate: for every (meter, window size,
// filter subset, aggregation) the cacheability gate admits — and at the two
// extremes (all-fresh cutoff==from, all-cached cutoff==to) — the cached result
// must equal the live result. It ports the cachebench parity harness into the
// connector integration suite, comparing the merge (queryCachedMeter, driven at
// an explicit cutoff) against the live queryMeter path.

const parityCacheTable = meterQueryRowCacheTable

type parityMeter struct {
	name          string
	eventType     string
	valueProperty *string
	aggregation   meterpkg.MeterAggregation
	groupBy       map[string]string // meter group-by key -> JSON path
}

type parityEvent struct {
	subject string
	at      time.Time
	data    string
}

func (s *ConnectorTestSuite) newMeter(m parityMeter) meterpkg.Meter {
	now := time.Now().UTC()
	meter := meterpkg.Meter{
		ManagedResource: models.ManagedResource{
			ID:   ulid.Make().String(),
			Name: m.name,
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ManagedModel: models.ManagedModel{CreatedAt: now, UpdatedAt: now},
		},
		Key:           m.name,
		EventType:     m.eventType,
		ValueProperty: m.valueProperty,
		Aggregation:   m.aggregation,
		GroupBy:       m.groupBy,
	}
	s.NoError(meter.Validate())
	return meter
}

// seedEvents inserts raw events via BatchInsert. The late-event invalidation
// this triggers is harmless: every test populates the cache after seeding.
func (s *ConnectorTestSuite) seedEvents(ctx context.Context, eventType string, events []parityEvent) {
	raw := make([]streaming.RawEvent, 0, len(events))
	now := time.Now().UTC()
	for i, e := range events {
		raw = append(raw, streaming.RawEvent{
			Namespace:  namespace,
			ID:         fmt.Sprintf("%s-%d-%s", eventType, i, ulid.Make().String()),
			Time:       e.at,
			Type:       eventType,
			Source:     "parity-test",
			Subject:    e.subject,
			Data:       e.data,
			IngestedAt: now,
			StoredAt:   now,
		})
	}
	s.NoError(s.Connector.BatchInsert(ctx, raw))
}

// liveRows runs the live meter query path for the given query params, applying
// the same nil-window From/To overwrite that the public QueryMeter applies, so
// it compares apples-to-apples with cachedRows (which also applies it).
func (s *ConnectorTestSuite) liveRows(ctx context.Context, m meterpkg.Meter, params streaming.QueryParams) []meterpkg.MeterQueryRow {
	groupBy := append([]string(nil), params.GroupBy...)
	sort.Strings(groupBy)
	q := s.buildQueryMeter(m, params, groupBy)
	rows, err := s.Connector.queryMeter(ctx, q)
	s.NoError(err)
	return overwriteTotalWindow(rows, params)
}

// overwriteTotalWindow mirrors QueryMeter's behavior for the total (nil) window:
// each row's window start/end is replaced with the query's From/To. Both the
// live and cached comparison paths apply it so their windowstart/end line up.
func overwriteTotalWindow(rows []meterpkg.MeterQueryRow, params streaming.QueryParams) []meterpkg.MeterQueryRow {
	if params.WindowSize != nil {
		return rows
	}
	for i := range rows {
		if params.From != nil {
			rows[i].WindowStart = *params.From
		}
		if params.To != nil {
			rows[i].WindowEnd = *params.To
		}
	}
	return rows
}

// cachedRows populates the settled range then runs the merge at the explicit
// cutoff, returning the merged rows. It is populate-then-read; for tests that
// must control population separately (e.g. cross-meter coexistence), use
// populateCache + readCached.
func (s *ConnectorTestSuite) cachedRows(ctx context.Context, m meterpkg.Meter, params streaming.QueryParams, cutoff time.Time) []meterpkg.MeterQueryRow {
	s.populateCache(ctx, m, params, cutoff)
	return s.readCached(ctx, m, params, cutoff)
}

// populateCache rolls up the settled whole-hour range [ceil(from), floor(cutoff))
// for the meter into the cache table, mirroring queryMeterCached's populate step.
func (s *ConnectorTestSuite) populateCache(ctx context.Context, m meterpkg.Meter, params streaming.QueryParams, cutoff time.Time) {
	groupBy := append([]string(nil), params.GroupBy...)
	sort.Strings(groupBy)
	q := s.buildQueryMeter(m, params, groupBy)

	merge := s.newMergeQuery(q, cutoff)
	cacheLo := merge.headCeil()
	cacheHi := merge.cacheHi()
	if cacheLo.Before(cacheHi) {
		s.NoError(s.Connector.populateMeterQueryRowCache(ctx, q, cacheLo, cacheHi))
	}
}

// readCached runs the merge (cache + live legs) WITHOUT populating, so a test can
// populate multiple meters into the shared table first and then read one.
func (s *ConnectorTestSuite) readCached(ctx context.Context, m meterpkg.Meter, params streaming.QueryParams, cutoff time.Time) []meterpkg.MeterQueryRow {
	groupBy := append([]string(nil), params.GroupBy...)
	sort.Strings(groupBy)
	q := s.buildQueryMeter(m, params, groupBy)

	merge := s.newMergeQuery(q, cutoff)
	sql, args, err := merge.toSQL()
	s.Require().NoError(err)

	rows, err := s.Connector.config.ClickHouse.Query(ctx, sql, args...)
	s.Require().NoError(err)
	defer rows.Close()

	out, err := merge.scanRows(rows)
	s.Require().NoError(err)

	// Mirror QueryMeter's nil-window overwrite so parity holds for total queries.
	return overwriteTotalWindow(out, params)
}

func (s *ConnectorTestSuite) newMergeQuery(q queryMeter, cutoff time.Time) queryCachedMeter {
	return queryCachedMeter{
		Database:        s.Connector.config.Database,
		CacheTableName:  parityCacheTable,
		EventsTableName: s.Connector.config.EventsTableName,
		query:           q,
		Cutoff:          cutoff,
	}
}

func (s *ConnectorTestSuite) buildQueryMeter(m meterpkg.Meter, params streaming.QueryParams, groupBy []string) queryMeter {
	return queryMeter{
		Database:               s.Connector.config.Database,
		EventsTableName:        s.Connector.config.EventsTableName,
		Namespace:              namespace,
		Meter:                  m,
		From:                   params.From,
		To:                     params.To,
		FilterSubject:          params.FilterSubject,
		FilterGroupBy:          params.FilterGroupBy,
		GroupBy:                groupBy,
		WindowSize:             params.WindowSize,
		WindowTimeZone:         params.WindowTimeZone,
		EnableDecimalPrecision: s.Connector.config.EnableDecimalPrecision,
	}
}

// rowKey is the full parity key: window + windowend + subject + customer +
// group-by. Optional values carry a nil/value marker so nil and "" — distinct
// in the API response — never compare equal.
func rowKey(r meterpkg.MeterQueryRow) string {
	var b strings.Builder

	writeOptional := func(v *string) {
		if v == nil {
			b.WriteByte('n')
		} else {
			b.WriteByte('v')
			b.WriteString(*v)
		}
	}

	b.WriteString(r.WindowStart.UTC().Format(time.RFC3339))
	b.WriteByte(0)
	b.WriteString(r.WindowEnd.UTC().Format(time.RFC3339))
	b.WriteByte(0)
	writeOptional(r.Subject)
	b.WriteByte(0)
	writeOptional(r.CustomerID)
	b.WriteByte(0)

	keys := make([]string, 0, len(r.GroupBy))
	for k := range r.GroupBy {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte(1)
		writeOptional(r.GroupBy[k])
		b.WriteByte(2)
	}
	return b.String()
}

// compareMeterRows returns "" if the two result sets match, keyed on the full
// tuple, exact value compared to 6 decimal places (matching cachebench's
// compareRows / digest rounding). Any difference returns a human-readable diff.
func compareMeterRows(live, cached []meterpkg.MeterQueryRow) string {
	if len(live) != len(cached) {
		return fmt.Sprintf("row count differs: live=%d cached=%d\nlive=%s\ncached=%s",
			len(live), len(cached), dumpRows(live), dumpRows(cached))
	}

	l := append([]meterpkg.MeterQueryRow(nil), live...)
	c := append([]meterpkg.MeterQueryRow(nil), cached...)
	sort.Slice(l, func(i, j int) bool { return rowKey(l[i]) < rowKey(l[j]) })
	sort.Slice(c, func(i, j int) bool { return rowKey(c[i]) < rowKey(c[j]) })

	for i := range l {
		if rowKey(l[i]) != rowKey(c[i]) {
			return fmt.Sprintf("key[%d] differs:\nlive=  %q\ncached=%q", i, rowKey(l[i]), rowKey(c[i]))
		}
		if round6(l[i].Value) != round6(c[i].Value) {
			return fmt.Sprintf("value @ %q differs: live=%v cached=%v", rowKey(l[i]), l[i].Value, c[i].Value)
		}
	}
	return ""
}

func round6(f float64) float64 {
	return math.Round(f*1e6) / 1e6
}

func dumpRows(rows []meterpkg.MeterQueryRow) string {
	var b strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&b, "\n  ws=%s we=%s subj=%v gb=%v val=%v",
			r.WindowStart.UTC().Format(time.RFC3339), r.WindowEnd.UTC().Format(time.RFC3339),
			ptrStr(r.Subject), groupByStr(r.GroupBy), r.Value)
	}
	return b.String()
}

func ptrStr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func groupByStr(m map[string]*string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(ptrStr(m[k]))
		b.WriteByte(',')
	}
	return b.String()
}

func ptr[T any](v T) *T { return &v }

func filterEq(v string) filter.FilterString {
	return filter.FilterString{Eq: &v}
}
