package clickhouse

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// CHTestSuite provides ClickHouse setup and common helpers for tests in this package.
type CHTestSuite struct {
	suite.Suite
	*require.Assertions

	ClickHouse clickhouse.Conn
	Database   string
	cleanup    func()
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

	s.cleanup = s.CreateTempDatabase(s.T())
}

func (s *CHTestSuite) TearDownTest() {
	if s.T().Skipped() {
		return
	}

	s.cleanup()
	s.cleanup = nil
}

// CreateTempDatabase creates a unique database and returns a cleanup function.
// The cleanup function drops the database only if the test did not fail.
func (s *CHTestSuite) CreateTempDatabase(t *testing.T) func() {
	db := fmt.Sprintf("test_%s", ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String())
	s.NoError(s.ClickHouse.Exec(t.Context(), fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", db)))
	t.Log("created database", "database", db)
	s.Database = db
	cleanup := func() {
		if !t.Failed() {
			_ = s.ClickHouse.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s SYNC", db))
			_ = s.ClickHouse.Close()
			s.Database = ""
			s.ClickHouse = nil
		}
	}

	return cleanup
}
