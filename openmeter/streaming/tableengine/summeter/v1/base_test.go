package summeterv1

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	chconn "github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
)

// CHTestSuite provides ClickHouse setup and common helpers for tests in this package.
type CHTestSuite struct {
	suite.Suite
	*require.Assertions

	ClickHouse clickhouse.Conn
	Database   string
}

func (s *CHTestSuite) SetupSuite() {
	s.Assertions = require.New(s.T())
}

func (s *CHTestSuite) SetupTest() {
	dsn := os.Getenv("TEST_CLICKHOUSE_DSN")
	if dsn == "" {
		s.T().Skip("TEST_CLICKHOUSE_DSN is not set; skipping integration tests")
	}

	opts, err := clickhouse.ParseDSN(dsn)
	s.Require().NoError(err, "failed to parse ClickHouse DSN")
	conn, err := clickhouse.Open(opts)
	s.Require().NoError(err, "failed to open ClickHouse connection")
	s.ClickHouse = conn
}

// CreateTempDatabase creates a unique database and returns its name and a cleanup function.
// The cleanup function drops the database only if the test did not fail.
func (s *CHTestSuite) CreateTempDatabase(t *testing.T) (string, func()) {
	db := fmt.Sprintf("test_%s", ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String())
	s.NoError(s.ClickHouse.Exec(t.Context(), fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", db)))
	t.Log("created database", "database", db)
	cleanup := func() {
		if !t.Failed() {
			_ = s.ClickHouse.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s SYNC", db))
		}
	}
	return db, cleanup
}

// CreateEventsTable creates the om_events table in the given database.
func (s *CHTestSuite) CreateEventsTable(t *testing.T, database string) {
	createEvents := chconn.CreateEventsTableSQL(database, "om_events")
	s.NoError(s.ClickHouse.Exec(t.Context(), createEvents))
}

// InsertEvent inserts a single event row into the given database's om_events table.
func (s *CHTestSuite) InsertEvent(t *testing.T, database string, namespace, typ, subject string, at, ingestedAt, storedAt time.Time, id string, data string) {
	if data == "" {
		data = "{}"
	}
	sql := fmt.Sprintf("INSERT INTO %s.om_events (namespace, id, type, subject, source, time, data, ingested_at, stored_at, store_row_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", database)
	err := s.ClickHouse.Exec(t.Context(), sql,
		namespace, id, typ, subject, "src", at, data, ingestedAt, storedAt, id,
	)
	s.Require().NoError(err)
}

// NewEngine constructs an Engine bound to this suite's ClickHouse and database.
func (s *CHTestSuite) NewEngine(database string) Engine {
	return Engine{
		logger:     slog.Default(),
		database:   database,
		clickhouse: s.ClickHouse,
	}
}
