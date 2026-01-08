package summeterv1

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/samber/lo"
)

func TestQueryMinEventsStoredAt_ToSQL(t *testing.T) {
	tests := []struct {
		name     string
		q        QueryMinEventsStoredAt
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			name: "basic",
			q: QueryMinEventsStoredAt{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				EventType:       "event.type",
			},
			wantSQL:  "SELECT min(stored_at) FROM openmeter.om_events WHERE namespace = ? AND type = ?",
			wantArgs: []interface{}{"my_namespace", "event.type"},
		},
		{
			name: "custom db and table",
			q: QueryMinEventsStoredAt{
				Database:        "db",
				EventsTableName: "events",
				Namespace:       "ns",
				EventType:       "evt",
			},
			wantSQL:  "SELECT min(stored_at) FROM db.events WHERE namespace = ? AND type = ?",
			wantArgs: []interface{}{"ns", "evt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs := tt.q.ToSQL()
			assert.Equal(t, tt.wantSQL, gotSQL)
			assert.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}

// --- Integration tests with ClickHouse using a suite ---

type SQLTestSuite struct {
	CHTestSuite
	Engine Engine
}

func TestSQLSuite(t *testing.T) {
	suite.Run(t, new(SQLTestSuite))
}

func (s *SQLTestSuite) SetupTest() {
	s.CHTestSuite.SetupTest()
	// Create isolated database and table per test
	db, _ := s.CreateTempDatabase(s.T())
	s.Database = db
	s.CreateEventsTable(s.T(), s.Database)
	// Engine under test
	s.Engine = s.NewEngine(s.Database)
}

func (s *SQLTestSuite) TearDownTest() {
	// DB cleanup handled by CreateTempDatabase cleanup; nothing else to do here
}

func (s *SQLTestSuite) TestMinEventsStoredAt_DataFound() {
	ctx := s.T().Context()
	ns := "ns1"
	typ := "evt.type"
	now := time.Now().UTC().Truncate(time.Second)

	// Non-matching rows
	s.InsertEvent(s.T(), s.Database, "other-ns", typ, "s1", now, now, now, "id-1", "")
	s.InsertEvent(s.T(), s.Database, ns, "other.type", "s1", now, now, now, "id-2", "")

	// Matching rows with different stored_at
	minTs := now.Add(-10 * time.Minute)
	s.InsertEvent(s.T(), s.Database, ns, typ, "sA", now.Add(-1*time.Hour), now.Add(-1*time.Hour), now.Add(-5*time.Minute), "id-3", "")
	s.InsertEvent(s.T(), s.Database, ns, typ, "sB", now.Add(-2*time.Hour), now.Add(-2*time.Hour), minTs, "id-4", "")

	got, err := s.Engine.MinEventsStoredAt(ctx, "om_events", ns, typ)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal(minTs, got.UTC().Truncate(time.Second))
}

func (s *SQLTestSuite) TestMinEventsStoredAt_NoData() {
	ctx := s.T().Context()
	ns := "ns-empty"
	typ := "evt.type"

	got, err := s.Engine.MinEventsStoredAt(ctx, "om_events", ns, typ)
	s.Require().NoError(err)
	s.Nil(got)
}

func (s *SQLTestSuite) TestInsertFromEvents_ExecutesAndInserts() {
	ctx := s.T().Context()

	// Create numeric meter table
	s.Engine = Engine{
		logger:   slog.Default(),
		database: s.Database,
	}
	s.Engine.clickhouse = s.ClickHouse
	createMeterSQL := s.Engine.CreateTableSQL()
	s.NoError(s.ClickHouse.Exec(ctx, createMeterSQL))

	// Subtests driven by meter value path, group-by, and event payload
	type testcase struct {
		name           string
		valuePath      *string
		groupBy        map[string]string
		payload        string
		expectInserted bool
		expectedValue  *string
		expectedGroups map[string]string
	}
	testcases := []testcase{
		{
			name:           "basic",
			valuePath:      lo.ToPtr("$.amount"),
			groupBy:        map[string]string{"g1": "$.a", "g2": "$.b"},
			payload:        `{"a":"A","b":"B","amount":123.45}`,
			expectInserted: true,
			expectedValue:  lo.ToPtr("123.45"),
			expectedGroups: map[string]string{"g1": "A", "g2": "B"},
		},
		{
			name:           "value is null -> skip row",
			valuePath:      lo.ToPtr("$.amount"),
			groupBy:        map[string]string{"g1": "$.a", "g2": "$.b"},
			payload:        `{"a":"A","b":"B"}`, // missing amount
			expectInserted: false,
		},
		{
			name:           "group-by values missing -> empty strings",
			valuePath:      lo.ToPtr("$.amount"),
			groupBy:        map[string]string{"g1": "$.missing", "g2": "$.b"},
			payload:        `{"b":"B","amount":42}`,
			expectInserted: true,
			expectedValue:  lo.ToPtr("42"),
			expectedGroups: map[string]string{"g1": "", "g2": "B"},
		},
		{
			name:           "no group-by configured -> empty map",
			valuePath:      lo.ToPtr("$.amount"),
			groupBy:        map[string]string{},
			payload:        `{"a":"A","b":"B","amount":7.5}`,
			expectInserted: true,
			expectedValue:  lo.ToPtr("7.5"),
			expectedGroups: nil, // null map: expect length(group_by_filters) = 0
		},
		// TODO: invalid values such as array and object references
	}

	evTime := testutils.GetRFC3339Time(s.T(), "2025-02-01T13:00:00Z")
	ns := "ns-exec"
	subject := "subj-1"

	state := EngineState{
		StreamDataAfterStoredAt: evTime.Add(-time.Second),
	}
	stateJSON, _ := json.Marshal(state)

	for i, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			// fresh event time to isolate rows

			typ := fmt.Sprintf("evt.type.%d", i)
			s.InsertEvent(s.T(), s.Database, ns, typ, subject, evTime, evTime, evTime, "evt-"+tc.name, tc.payload)

			mc := meter.Meter{
				EventType:     typ,
				Aggregation:   meter.MeterAggregationSum,
				ValueProperty: tc.valuePath,
				GroupBy:       tc.groupBy,
				TableEngine: &meter.MeterTableEngine{
					Engine: TableEngineName,
					Status: meter.MeterTableEngineStateActive,
					State:  string(stateJSON),
				},
			}
			mc.ManagedResource.ID = "meter-" + tc.name
			mc.ManagedResource.NamespacedModel.Namespace = ns

			period := timeutil.ClosedPeriod{
				From: evTime.Add(-time.Minute),
				To:   evTime.Add(time.Minute),
			}
			err := s.Engine.InsertFromEvents(ctx, "om_events", mc, period)
			if !tc.expectInserted {
				s.NoError(err)
				qc := fmt.Sprintf("SELECT count() FROM %s.%s WHERE namespace = ? AND meter_id = ? AND stored_at >= ? AND stored_at < ?", s.Database, TableName)
				r, e := s.ClickHouse.Query(ctx, qc, ns, mc.ID, period.From, period.To)
				s.Require().NoError(e)
				defer r.Close()
				var cnt uint64
				if r.Next() {
					s.NoError(r.Scan(&cnt))
				}
				s.NoError(r.Err())
				s.Equal(uint64(0), cnt)
				return
			}
			s.Require().NoError(err)

			qv := fmt.Sprintf("SELECT toString(value), group_by_filters FROM %s.%s WHERE namespace = ? AND meter_id = ? AND stored_at = ?", s.Database, TableName)
			r, e := s.ClickHouse.Query(ctx, qv, ns, mc.ID, evTime)
			s.Require().NoError(e)
			defer r.Close()
			var vStr string
			var groups map[string]string
			if r.Next() {
				s.NoError(r.Scan(&vStr, &groups))
			} else {
				s.Fail("expected one row inserted")
				return
			}
			s.NoError(r.Err())
			if tc.expectedValue != nil {
				gotDec, gErr := alpacadecimal.NewFromString(vStr)
				s.Require().NoError(gErr)
				expDec, eErr := alpacadecimal.NewFromString(*tc.expectedValue)
				s.Require().NoError(eErr)
				s.True(gotDec.Equal(expDec), "expected %s, got %s", expDec.String(), gotDec.String())
			}
			if tc.expectedGroups == nil {
				s.Equal(0, len(groups), "expected null/empty map for group_by_filters")
			} else {
				s.Equal(tc.expectedGroups, groups)
			}

			// Parity with Engine.GetRecordForMeter
			// Build minimal engine state to pass gating

			ev := serializer.CloudEventsKafkaPayload{
				Type:    mc.EventType,
				Subject: subject,
				Time:    evTime.Unix(),
				Data:    tc.payload,
			}
			rec, rerr := s.Engine.GetRecordForMeter(ctx, mc, ev, evTime)
			s.Require().NoError(rerr)
			s.Require().NotNil(rec)

			// Identity fields
			s.Equal(ns, rec.Namespace)
			s.Equal(mc.ID, rec.MeterID)
			s.Equal(subject, rec.Subject)
			s.Equal(evTime.In(time.UTC), rec.Time.In(time.UTC))

			// Value parity using decimal comparison (avoid float rounding)
			chDec, err := alpacadecimal.NewFromString(vStr)
			s.Require().NoError(err)
			s.True(chDec.Equal(rec.Value), "decimal mismatch: ch=%s engine=%s", chDec.String(), rec.Value.String())

			if tc.expectedGroups == nil {
				s.Equal(0, len(rec.GroupByFilters), "expected null/empty map for group_by_filters")
			} else {
				s.Equal(tc.expectedGroups, rec.GroupByFilters)
			}
		})
	}
}
