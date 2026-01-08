package summeterv1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	chconn "github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type EngineTestSuite struct {
	CHTestSuite
	Engine Engine
}

func TestEngineSuite(t *testing.T) {
	suite.Run(t, new(EngineTestSuite))
}

func (s *EngineTestSuite) SetupTest() {
	s.CHTestSuite.SetupTest()
	s.Assertions = require.New(s.T())
	s.Engine = Engine{
		logger: slog.Default(),
	}
}

func (s *EngineTestSuite) TestGetRecordForMeter() {
	now := time.Now().UTC().Truncate(time.Second)
	minStoredAt := now.Add(-time.Minute)

	state := EngineState{
		StreamDataAfterStoredAt: minStoredAt,
	}
	stateJSON, err := json.Marshal(state)
	s.NoError(err)

	makeMeter := func(valuePath string, groupBy map[string]string) meter.Meter {
		m := meter.Meter{
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test.event",
			ValueProperty: lo.ToPtr(valuePath),
			GroupBy:       groupBy,
			TableEngine: &meter.MeterTableEngine{
				Engine: TableEngineName,
				Status: meter.MeterTableEngineStateActive,
				State:  string(stateJSON),
			},
		}
		// Minimal identifiers; not essential for assertions
		m.ManagedResource.ID = "m1"
		m.ManagedResource.NamespacedModel.Namespace = "ns"
		return m
	}

	type testCase struct {
		name            string
		meter           meter.Meter
		dataJSON        string
		storedAt        time.Time
		expectErr       bool
		expectNoRecord  bool
		expectedValue   *float64
		expectedGroupBy map[string]string
	}

	tests := []testCase{
		{
			name:            "valid float value and groupbys",
			meter:           makeMeter("$.amount", map[string]string{"g1": "$.foo", "g2": "$.bar"}),
			dataJSON:        `{"amount": 12.34, "foo": "a", "bar": "b"}`,
			storedAt:        now,
			expectedValue:   lo.ToPtr(12.34),
			expectedGroupBy: map[string]string{"g1": "a", "g2": "b"},
		},
		{
			name:            "numeric string value",
			meter:           makeMeter("$.amount", map[string]string{"g1": "$.foo"}),
			dataJSON:        `{"amount": "45.67", "foo": "x"}`,
			storedAt:        now,
			expectedValue:   lo.ToPtr(45.67),
			expectedGroupBy: map[string]string{"g1": "x"},
		},
		{
			name:            "missing value path returns nil record",
			meter:           makeMeter("$.missing", map[string]string{"g1": "$.foo"}),
			dataJSON:        `{"amount": 1, "foo": "y"}`,
			storedAt:        now,
			expectNoRecord:  true,
			expectedGroupBy: nil, // ignored
		},
		{
			name:           "non-numeric string value returns nil",
			meter:          makeMeter("$.amount", map[string]string{"g1": "$.foo"}),
			dataJSON:       `{"amount": "NaN", "foo": "z"}`,
			storedAt:       now,
			expectNoRecord: true,
		},
		{
			name:            "missing groupby path yields empty string",
			meter:           makeMeter("$.amount", map[string]string{"g1": "$.missing"}),
			dataJSON:        `{"amount": 10}`,
			storedAt:        now,
			expectedValue:   lo.ToPtr(10.0),
			expectedGroupBy: map[string]string{"g1": ""},
		},
		{
			name:            "groupby numeric/object are json-stringified",
			meter:           makeMeter("$.amount", map[string]string{"n": "$.num", "o": "$.obj"}),
			dataJSON:        `{"amount": 3, "num": 123, "obj": {"a":1}}`,
			storedAt:        now,
			expectedValue:   lo.ToPtr(3.0),
			expectedGroupBy: map[string]string{"n": "123", "o": `{"a":1}`},
		},
		{
			name:           "storedAt before MinStoredAt returns nil",
			meter:          makeMeter("$.amount", map[string]string{"g": "$.foo"}),
			dataJSON:       `{"amount": 1, "foo":"v"}`,
			storedAt:       minStoredAt.Add(-time.Second),
			expectNoRecord: true,
			// Skip CH parity for this case since record is gated out
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ev := serializer.CloudEventsKafkaPayload{
				Id:      "id1",
				Type:    "test.event",
				Source:  "src",
				Subject: "subj",
				Time:    now.Unix(),
				Data:    tt.dataJSON,
			}

			rec, err := s.Engine.GetRecordForMeter(ctx, tt.meter, ev, tt.storedAt)
			if tt.expectErr {
				s.Error(err, "expected error but got none: case=%s", tt.name)
				return
			}
			s.NoError(err, "unexpected error from GetRecordForMeter: case=%s", tt.name)

			if tt.expectNoRecord {
				s.Nil(rec, "expected no record (expectNoRecord) but got one: case=%s", tt.name)
				return
			}

			s.NotNil(rec, "expected a record but got nil: case=%s", tt.name)

			if tt.expectedValue != nil {
				expected := alpacadecimal.NewFromFloat(*tt.expectedValue)
				s.True(rec.Value.Equal(expected), "value mismatch: expected %s, got %s: case=%s", expected.String(), rec.Value.String(), tt.name)
			}

			if tt.expectedGroupBy != nil {
				s.Equal(tt.expectedGroupBy, rec.GroupByFilters, "groupBy mismatch: case=%s", tt.name)
			}

			// ClickHouse parity check for value and group-by
			if !tt.expectNoRecord {
				sql := "SELECT ifNotFinite(toFloat64OrNull(JSON_VALUE(?, ?)), null) AS v"
				rows, qerr := s.ClickHouse.Query(ctx, sql, tt.dataJSON, *tt.meter.ValueProperty)
				s.NoError(qerr, "ClickHouse value query failed: case=%s", tt.name)
				defer rows.Close()

				var chVal *float64
				if rows.Next() {
					s.NoError(rows.Scan(&chVal), "ClickHouse value scan failed: case=%s", tt.name)
				}
				s.NoError(rows.Err(), "ClickHouse rows error after value query: case=%s", tt.name)

				if tt.expectedValue != nil {
					s.NotNil(chVal, "CH consistency mismatch: expected numeric value but got NULL: case=%s", tt.name)
					// compare with small tolerance
					const eps = 1e-9
					s.InDelta(*tt.expectedValue, *chVal, eps, "CH consistency mismatch: value mismatch: expected %f, got %f: case=%s", *tt.expectedValue, lo.FromPtr(chVal), tt.name)
				} else {
					s.Nil(chVal, "CH consistency mismatch: expected NULL value but got %v: case=%s", chVal, tt.name)
				}

				// Validate group-by parity with ClickHouse JSON_VALUE
				for key, path := range tt.meter.GroupBy {
					gbRows, gErr := s.ClickHouse.Query(ctx, "SELECT JSON_VALUE(?, ?) AS g", tt.dataJSON, path)
					s.NoError(gErr, "ClickHouse group-by query failed for key=%s: case=%s", key, tt.name)
					var gb *string
					if gbRows.Next() {
						s.NoError(gbRows.Scan(&gb), "ClickHouse group-by scan failed for key=%s: case=%s", key, tt.name)
					}
					s.NoError(gbRows.Err(), "ClickHouse rows error after group-by query for key=%s: case=%s", key, tt.name)
					_ = gbRows.Close()

					exp := ""
					if tt.expectedGroupBy != nil {
						if v, ok := tt.expectedGroupBy[key]; ok {
							exp = v
						}
					}
					if gb == nil {
						// JSON_VALUE should return empty string for missing path
						s.Equal("", exp, "CH consistency mismatch: nil JSON_VALUE; expected %q for key=%s: case=%s", exp, key, tt.name)
					} else {
						s.Equal(exp, *gb, "CH consistency mismatch: groupBy '%s' mismatch: expected %q, got %q: case=%s", key, exp, *gb, tt.name)
					}
				}
			}
		})
	}
}

func (s *EngineTestSuite) TestMaintainPopulatesMinStoredAtAndChunks() {
	ctx := context.Background()

	// Setup a dedicated database for this test and events table
	dbName, cleanup := s.createTestDatabaseWithEventsTable(ctx)
	defer cleanup()

	// Insert events: ensure min stored_at is on 2025-01-02 10:15:00 UTC
	ns := "ns-state"
	evt := "meter.evt"
	minTs := time.Date(2025, 1, 2, 10, 15, 0, 0, time.UTC)

	// Non-matching
	s.insertEvent(ctx, dbName, "other-ns", evt, minTs)
	s.insertEvent(ctx, dbName, ns, "other.evt", minTs)
	// Matching with different stored_at
	s.insertEvent(ctx, dbName, ns, evt, minTs.Add(5*time.Minute))
	s.insertEvent(ctx, dbName, ns, evt, minTs) // min
	s.insertEvent(ctx, dbName, ns, evt, minTs.Add(24*time.Hour))

	// Prepare engine and meter state
	s.Engine.database = dbName
	s.Engine.clickhouse = s.ClickHouse
	// Start from empty state; StreamDataAfterStoredAt will be set by Maintain to now + 5m
	frozenNow := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	clock.SetTime(frozenNow)
	defer clock.ResetTime()
	state := EngineState{}
	stateJSON, _ := json.Marshal(state)

	m := meter.Meter{
		Aggregation: meter.MeterAggregationSum,
		EventType:   evt,
		TableEngine: &meter.MeterTableEngine{
			Engine: TableEngineName,
			Status: meter.MeterTableEngineStateActive,
			State:  string(stateJSON),
		},
	}
	m.ManagedResource.ID = "m-state"
	m.ManagedResource.NamespacedModel.Namespace = ns

	s.T().Run("populate stream start", func(t *testing.T) {
		steps := Counter(1)
		err := s.Engine.Maintain(ctx, &steps, m)
		s.Require().NoError(err)

		var updated1 EngineState
		s.NoError(json.Unmarshal([]byte(m.TableEngine.State), &updated1))
		s.Equal(frozenNow.Add(5*time.Minute).UTC().Truncate(time.Second), updated1.StreamDataAfterStoredAt)
	})

	s.T().Run("compute min stored_at and generate chunks", func(t *testing.T) {
		steps := Counter(1)
		err := s.Engine.Maintain(ctx, &steps, m)
		s.Require().NoError(err)

		var updated2 EngineState
		s.NoError(json.Unmarshal([]byte(m.TableEngine.State), &updated2))
		s.Equal(minTs, updated2.Backfill.MinStoredAt)

		// Expected daily chunks: 2025-01-02, 2025-01-03, 2025-01-04, and partial on 5th up to 00:05
		exp := []timeutil.ClosedPeriod{
			{From: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), To: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)},
			{From: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), To: time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC)},
			{From: time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), To: time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)},
			{From: time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC), To: time.Date(2025, 1, 5, 0, 5, 0, 0, time.UTC)},
		}
		s.Len(updated2.Backfill.ImportChunks, len(exp))
		for i, ch := range updated2.Backfill.ImportChunks {
			s.Equal(exp[i].From, ch.From, "chunk %d From mismatch", i)
			s.Equal(exp[i].To, ch.To, "chunk %d To mismatch", i)
		}
	})
}

func (s *EngineTestSuite) createTestDatabaseWithEventsTable(ctx context.Context) (string, func()) {
	dbName := fmt.Sprintf("test_%s", ulid.Make().String())
	s.NoError(s.ClickHouse.Exec(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName)))
	createEvents := chconn.CreateEventsTableSQL(dbName, "om_events")
	s.NoError(s.ClickHouse.Exec(ctx, createEvents))
	cleanup := func() {
		_ = s.ClickHouse.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s SYNC", dbName))
	}
	return dbName, cleanup
}

func (s *EngineTestSuite) insertEvent(ctx context.Context, dbName string, namespace, typ string, storedAt time.Time) {
	sql := fmt.Sprintf("INSERT INTO %s.om_events (namespace, id, type, source, subject, time, data, ingested_at, stored_at, store_row_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", dbName)
	err := s.ClickHouse.Exec(ctx, sql,
		namespace, ulid.Make().String(), typ, "src", "subj",
		storedAt, "{}", storedAt, storedAt, ulid.Make().String(),
	)
	s.Require().NoError(err)
}

func BenchmarkGetRecordForMeter(b *testing.B) {
	logger := slog.Default()
	engine := Engine{logger: logger}

	// Prepare engine state so records are not gated out
	state := EngineState{
		StreamDataAfterStoredAt: time.Now().Add(-time.Hour),
	}
	stateJSON, _ := json.Marshal(state)

	// Build ~2KB JSON payload with 20 root keys
	filler := strings.Repeat("x", 100) // ~100 bytes per key value
	root := map[string]interface{}{
		"amount": 123.456,
		"foo":    "group-foo",
		"bar":    "group-bar",
		"obj": map[string]interface{}{
			"a": "group-obj-a",
		},
	}
	for i := 1; i <= 20; i++ {
		root["k"+strconv.Itoa(i)] = filler
	}
	dataBytes, _ := json.Marshal(root)
	dataJSON := string(dataBytes)

	m := meter.Meter{
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "bench.event",
		ValueProperty: lo.ToPtr("$.amount"),
		GroupBy: map[string]string{
			"g1": "$.foo",
			"g2": "$.bar",
			"g3": "$.obj.a",
		},
		TableEngine: &meter.MeterTableEngine{
			Engine: TableEngineName,
			Status: meter.MeterTableEngineStateActive,
			State:  string(stateJSON),
		},
	}
	m.ManagedResource.ID = "meter-bench"
	m.ManagedResource.NamespacedModel.Namespace = "ns-bench"

	ev := serializer.CloudEventsKafkaPayload{
		Id:      "bench-1",
		Type:    "bench.event",
		Source:  "bench",
		Subject: "subject-1",
		Time:    time.Now().Unix(),
		Data:    dataJSON,
	}
	storedAt := time.Now()

	// Sanity check
	if rec, err := engine.GetRecordForMeter(context.Background(), m, ev, storedAt); err != nil || rec == nil {
		b.Fatalf("setup failed: err=%v rec=%v", err, rec)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.GetRecordForMeter(context.Background(), m, ev, storedAt); err != nil {
			b.Fatal(err)
		}
	}
}
