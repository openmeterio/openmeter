package summeterv1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	db "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meterpgadapter "github.com/openmeterio/openmeter/openmeter/meter/adapter"
	meterservice "github.com/openmeterio/openmeter/openmeter/meter/service"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
)

type E2ESuite struct {
	CHTestSuite
	Engine Engine

	// Postgres (for meter service)
	TestDB      *testutils.TestDB
	DBClient    *db.Client
	MeterManage meter.ManageService
}

func TestE2E(t *testing.T) {
	suite.Run(t, new(E2ESuite))
}

func (s *E2ESuite) SetupSuite() {
	s.CHTestSuite.SetupSuite()
	// Initialize Postgres similarly to test/billing suite
	testDB := testutils.InitPostgresDB(s.T())
	s.TestDB = testDB

	dbClient := db.NewClient(db.Driver(testDB.EntDriver.Driver()))
	s.DBClient = dbClient

	// Migrate schema (don't ignore errors)
	s.NoError(dbClient.Schema.Create(context.Background()))

	// Meter adapter and manage service
	mAdapter, err := meterpgadapter.New(meterpgadapter.Config{
		Client: dbClient,
		Logger: slog.Default(),
	})
	s.NoError(err)
	// Namespace manager with a default (will be overridden by actual meter Namespace later as needed)
	nsManager, err := namespace.NewManager(namespace.ManagerConfig{DefaultNamespace: "default"})
	s.NoError(err)
	s.MeterManage = meterservice.NewManage(mAdapter, eventbus.NewMock(s.T()), nsManager, []*meter.EventTypePattern{})
}

func (s *E2ESuite) TearDownSuite() {
	// Cleanup DB only if suite did not fail
	if s.T().Failed() {
		return
	}
	if s.DBClient != nil {
		if err := s.DBClient.Close(); err != nil {
			s.T().Errorf("failed to close ent client: %v", err)
		}
	}
	if s.TestDB != nil {
		if err := s.TestDB.EntDriver.Close(); err != nil {
			s.T().Errorf("failed to close ent driver: %v", err)
		}
		if err := s.TestDB.PGDriver.Close(); err != nil {
			s.T().Errorf("failed to close postgres driver: %v", err)
		}
	}
}

func (s *E2ESuite) SetupTest() {
	s.CHTestSuite.SetupTest()
	// Create isolated database and table per test
	db, _ := s.CreateTempDatabase(s.T())
	s.Database = db
	s.CreateEventsTable(s.T(), s.Database)

	// Create meter table
	s.Engine = s.NewEngine(s.Database)
	// Wire meter service (required by Maintain)
	s.Engine.meterService = s.MeterManage
	createMeterSQL := s.Engine.CreateTableSQL()
	s.NoError(s.ClickHouse.Exec(s.T().Context(), createMeterSQL))
}

func (s *E2ESuite) TestEndToEndBackfill() {
	ctx := context.Background()

	s.Run("create meter and seed events", func() {
		// Create events affecting 2 subjects and 2 group-bys
		evBase := testutils.GetRFC3339Time(s.T(), "2025-03-01T10:00:00Z")
		payloads := []struct {
			subject string
			a       string
			b       string
			amount  string
			offset  time.Duration
		}{
			{"subj-1", "A1", "B1", "1.1", -2 * time.Hour},
			{"subj-1", "A1", "B2", "2.2", -90 * time.Minute},
			{"subj-1", "A2", "B1", "3.3", -80 * time.Minute},
			{"subj-1", "A2", "B2", "4.4", -70 * time.Minute},
			{"subj-2", "A1", "B1", "5.5", -60 * time.Minute},
			{"subj-2", "A1", "B2", "6.6", -50 * time.Minute},
			{"subj-2", "A2", "B1", "7.7", -40 * time.Minute},
			{"subj-2", "A2", "B2", "8.8", -30 * time.Minute},
			{"subj-1", "A1", "B1", "9.9", -20 * time.Minute},
			{"subj-2", "A2", "B2", "10.1", -10 * time.Minute},
		}
		for i, p := range payloads {
			st := evBase.Add(p.offset)
			data := fmt.Sprintf(`{"a":"%s","b":"%s","amount":%s}`, p.a, p.b, p.amount)
			s.InsertEvent(s.T(), s.Database, "ns-e2e", "evt.e2e", p.subject, st, st, st, fmt.Sprintf("id-%d", i), data)
		}
	})

	var m meter.Meter
	s.Run("alter meter to use table engine", func() {
		m = meter.Meter{
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "evt.e2e",
			ValueProperty: lo.ToPtr("$.amount"),
			GroupBy: map[string]string{
				"g1": "$.a",
				"g2": "$.b",
			},
		}
		m.ManagedResource.ID = "meter-e2e"
		m.ManagedResource.NamespacedModel.Namespace = "ns-e2e"

		// Create meter via service
		created, err := s.MeterManage.CreateMeter(ctx, meter.CreateMeterInput{
			Namespace:     m.Namespace,
			Name:          "e2e meter",
			Key:           m.ID,
			Aggregation:   m.Aggregation,
			EventType:     m.EventType,
			ValueProperty: m.ValueProperty,
			GroupBy:       m.GroupBy,
		})
		s.NoError(err)
		// Start with empty engine state and persist
		state := EngineState{}
		stateJSON, _ := json.Marshal(state)
		created.TableEngine = &meter.MeterTableEngine{
			Engine: TableEngineName,
			Status: meter.MeterTableEngineStateActive,
			State:  string(stateJSON),
		}
		s.NoError(s.MeterManage.UpdateTableEngine(ctx, created))
		// Use the persisted meter as base for subsequent steps
		m = created
	})

	// Single-pass maintain; advance time to after stream start + 5m
	clock.SetTime(testutils.GetRFC3339Time(s.T(), "2025-03-01T12:20:00Z"))
	defer clock.ResetTime()
	s.NoError(s.Engine.Maintain(ctx, m))

	s.Run("process chunks and verify records", func() {
		// Read current state
		var st EngineState
		s.NoError(json.Unmarshal([]byte(m.TableEngine.State), &st))

		// Validate that records have been inserted (10 events)
		countSQL := fmt.Sprintf("SELECT count() FROM %s.%s WHERE namespace = ? AND meter_id = ?", s.Database, TableName)
		rows, err := s.ClickHouse.Query(ctx, countSQL, m.Namespace, m.ID)
		s.NoError(err)
		defer rows.Close()
		var cnt uint64
		if rows.Next() {
			s.NoError(rows.Scan(&cnt))
		}
		s.NoError(rows.Err())
		s.Equal(uint64(10), cnt, "expected 10 inserted records")

		// Spot-check one record for subject and group-by
		q := fmt.Sprintf("SELECT subject, toString(value), group_by_filters['g1'], group_by_filters['g2'] FROM %s.%s WHERE namespace = ? AND meter_id = ? AND subject = ? ORDER BY stored_at LIMIT 1", s.Database, TableName)
		r, e := s.ClickHouse.Query(ctx, q, m.Namespace, m.ID, "subj-1")
		s.NoError(e)
		defer r.Close()
		var subj, vStr, g1, g2 string
		if r.Next() {
			s.NoError(r.Scan(&subj, &vStr, &g1, &g2))
		}
		s.NoError(r.Err())
		s.Equal("subj-1", subj)
		// Value should be a valid decimal
		_, derr := alpacadecimal.NewFromString(vStr)
		s.NoError(derr)
		// Group-by strings should be non-empty as per seeded data
		s.NotEmpty(g1)
		s.NotEmpty(g2)

		// Additional query validations
		s.Run("aggregate on a single subject", func() {
			rows, err := s.queryMeterSum(ctx, m, []string{"subject"}, []string{"subj-1"}, nil)
			s.NoError(err)
			s.Len(rows, 1)
			s.InDelta(20.9, rows[0].Value, 1e-9)
		})

		s.Run("filter by g1 on a single subject", func() {
			rows, err := s.queryMeterSum(ctx, m, []string{"subject"}, []string{"subj-1"}, map[string]filter.FilterString{
				"g1": {Eq: lo.ToPtr("A1")},
			})
			s.NoError(err)
			s.Len(rows, 1)
			s.InDelta(13.2, rows[0].Value, 1e-9)
		})
	})
}

// queryMeterSum builds the query using queryMeter (ToSQL) and scans rows via queryMeter.ScanRows.
func (s *E2ESuite) queryMeterSum(ctx context.Context, m meter.Meter, groupBy []string, filterSubject []string, filterGroupBy map[string]filter.FilterString) ([]meter.MeterQueryRow, error) {
	qm := queryMeter{
		Database:     s.Database,
		SumTableName: TableName,
		Namespace:    m.Namespace,
		Meter:        m,
		QueryParams: streaming.QueryParams{
			GroupBy: groupBy,
			// pass-through filters
			FilterSubject: filterSubject,
			FilterGroupBy: filterGroupBy,
		},
	}
	sql, args, err := qm.ToSQL()
	if err != nil {
		return nil, err
	}
	rows, err := s.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return qm.ScanRows(rows)
}
