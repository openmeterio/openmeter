package clickhouse

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/suite"

	progressmanager "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// Gated by TEST_CLICKHOUSE_DSN (same as the rest of this package). The
// ReplicatedMergeTree variant is additionally gated by
// TEST_CLICKHOUSE_REPLICATED=1 and the TEST_CLICKHOUSE_REPLICATED_* topology
// env vars read inline in TestReplicatedMergeTreeEngine.
type EventsTableEngineSuite struct {
	CHTestSuite
}

// TestExplicitMergeTreeEngine ensures that supplying an explicit MergeTree
// engine config does not regress the default code path: the connector
// constructor must succeed and the table must be created with the
// MergeTree engine.
func (s *EventsTableEngineSuite) TestExplicitMergeTreeEngine() {
	t := s.T()
	ctx := t.Context()

	connector, err := New(ctx, Config{
		Logger:          slog.Default(),
		ClickHouse:      s.ClickHouse,
		Database:        s.Database,
		EventsTableName: eventsTableName,
		EventsTableEngine: EventsTableEngine{
			Type: EventsTableEngineMergeTree,
		},
		ProgressManager: progressmanager.NewMockProgressManager(),
	})
	s.NoError(err, "failed to create connector with explicit MergeTree engine")
	s.NotNil(connector)

	s.assertEngine("MergeTree")
}

// TestUnsetEngineConfigDefaultsToMergeTree confirms the zero-value
// EventsTableEngine still produces a working MergeTree table — i.e. the
// new field is fully backwards compatible.
func (s *EventsTableEngineSuite) TestUnsetEngineConfigDefaultsToMergeTree() {
	t := s.T()
	ctx := t.Context()

	connector, err := New(ctx, Config{
		Logger:          slog.Default(),
		ClickHouse:      s.ClickHouse,
		Database:        s.Database,
		EventsTableName: eventsTableName,
		ProgressManager: progressmanager.NewMockProgressManager(),
	})
	s.NoError(err)
	s.NotNil(connector)

	s.assertEngine("MergeTree")
}

// TestInvalidEngineConfigFailsValidation ensures bad engine configs are
// rejected at construction time before any DDL is issued.
func (s *EventsTableEngineSuite) TestInvalidEngineConfigFailsValidation() {
	t := s.T()
	ctx := t.Context()

	_, err := New(ctx, Config{
		Logger:          slog.Default(),
		ClickHouse:      s.ClickHouse,
		Database:        s.Database,
		EventsTableName: eventsTableName,
		EventsTableEngine: EventsTableEngine{
			Type: EventsTableEngineReplicatedMergeTree,
			// missing ZooKeeperPath and ReplicaName
		},
		ProgressManager: progressmanager.NewMockProgressManager(),
	})
	s.Error(err)
	s.Contains(err.Error(), "events table engine")
}

// End-to-end ReplicatedMergeTree probe against a Keeper-backed cluster.
// Skipped unless TEST_CLICKHOUSE_REPLICATED=1, and skipped (with a clear
// message) if any of the TEST_CLICKHOUSE_REPLICATED_* topology env vars below
// are missing.
func (s *EventsTableEngineSuite) TestReplicatedMergeTreeEngine() {
	if os.Getenv("TEST_CLICKHOUSE_REPLICATED") != "1" {
		s.T().Skip("TEST_CLICKHOUSE_REPLICATED is not set; skipping ReplicatedMergeTree test")
	}

	cluster := os.Getenv("TEST_CLICKHOUSE_REPLICATED_CLUSTER")
	if cluster == "" {
		s.T().Skip("TEST_CLICKHOUSE_REPLICATED_CLUSTER is not set; skipping ReplicatedMergeTree test")
	}
	zkPath := os.Getenv("TEST_CLICKHOUSE_REPLICATED_ZK_PATH")
	if zkPath == "" {
		s.T().Skip("TEST_CLICKHOUSE_REPLICATED_ZK_PATH is not set; skipping ReplicatedMergeTree test")
	}
	replicaName := os.Getenv("TEST_CLICKHOUSE_REPLICATED_REPLICA_NAME")
	if replicaName == "" {
		s.T().Skip("TEST_CLICKHOUSE_REPLICATED_REPLICA_NAME is not set; skipping ReplicatedMergeTree test")
	}
	node2DSN := os.Getenv("TEST_CLICKHOUSE_REPLICATED_NODE2_DSN")
	if node2DSN == "" {
		s.T().Skip("TEST_CLICKHOUSE_REPLICATED_NODE2_DSN is not set; skipping ReplicatedMergeTree test")
	}
	replicaCountEnv := os.Getenv("TEST_CLICKHOUSE_REPLICATED_REPLICA_COUNT")
	if replicaCountEnv == "" {
		s.T().Skip("TEST_CLICKHOUSE_REPLICATED_REPLICA_COUNT is not set; skipping ReplicatedMergeTree test")
	}
	expectedReplicas, err := strconv.ParseUint(replicaCountEnv, 10, 64)
	s.NoError(err, "TEST_CLICKHOUSE_REPLICATED_REPLICA_COUNT must be a positive integer")
	s.Greater(expectedReplicas, uint64(0))

	t := s.T()
	ctx := t.Context()

	// CHTestSuite.CreateTempDatabase only creates the database on the
	// connected node. ReplicatedMergeTree with ON CLUSTER propagates the DDL
	// to every replica, so the database must exist on all of them.
	s.NoError(s.ClickHouse.Exec(ctx, fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS %s ON CLUSTER %s", s.Database, quoteClusterIdentifier(cluster),
	)))
	// defer (not t.Cleanup) because testify's TearDownTest closes
	// s.ClickHouse before subtest t.Cleanup callbacks fire.
	conn, dbName := s.ClickHouse, s.Database
	defer func() {
		_ = conn.Exec(t.Context(), fmt.Sprintf(
			"DROP DATABASE IF EXISTS %s ON CLUSTER %s SYNC", dbName, quoteClusterIdentifier(cluster),
		))
	}()

	connector, err := New(ctx, Config{
		Logger:          slog.Default(),
		ClickHouse:      s.ClickHouse,
		Database:        s.Database,
		EventsTableName: eventsTableName,
		EventsTableEngine: EventsTableEngine{
			Type:          EventsTableEngineReplicatedMergeTree,
			ZooKeeperPath: zkPath,
			ReplicaName:   replicaName,
			Cluster:       cluster,
		},
		ProgressManager: progressmanager.NewMockProgressManager(),
	})
	s.NoError(err, "failed to create connector with ReplicatedMergeTree engine")
	s.NotNil(connector)

	s.assertEngine("ReplicatedMergeTree")

	// Table must exist on every replica — what `ON CLUSTER` guarantees.
	var replicaCount uint64
	row := s.ClickHouse.QueryRow(ctx, fmt.Sprintf(`
		SELECT count() FROM clusterAllReplicas('%s', system.tables)
		WHERE database = '%s' AND name = '%s'`,
		escapeStringLiteral(cluster), s.Database, eventsTableName,
	))
	s.NoError(row.Scan(&replicaCount))
	s.EqualValues(expectedReplicas, replicaCount,
		"events table should exist on every replica in the cluster")

	// End-to-end replication probe: insert via the connector and read back
	// from node-02. If Keeper didn't substitute {shard}/{replica}, node-02
	// would not receive the row.
	now := time.Now().UTC()
	s.NoError(connector.BatchInsert(ctx, []streaming.RawEvent{{
		Namespace:  "ns-replicated",
		ID:         "evt-1",
		Type:       "test",
		Source:     "test",
		Subject:    "subj-1",
		Time:       now,
		Data:       `{"v": 1}`,
		IngestedAt: now,
		StoredAt:   now,
		StoreRowID: "1",
	}}))

	node02Opts, err := clickhouse.ParseDSN(node2DSN)
	s.NoError(err)
	node02, err := clickhouse.Open(node02Opts)
	s.NoError(err)
	defer node02.Close()

	// Replication is asynchronous; poll briefly.
	var seen uint64
	for attempt := 0; attempt < 30; attempt++ {
		row := node02.QueryRow(ctx, fmt.Sprintf(
			"SELECT count() FROM %s.%s WHERE namespace = 'ns-replicated'",
			s.Database, eventsTableName,
		))
		s.NoError(row.Scan(&seen))
		if seen == 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.EqualValues(1, seen, "row inserted on node-01 should replicate to node-02")
}

func (s *EventsTableEngineSuite) assertEngine(want string) {
	t := s.T()
	ctx := t.Context()

	row := s.ClickHouse.QueryRow(ctx,
		fmt.Sprintf(
			"SELECT engine FROM system.tables WHERE database = '%s' AND name = '%s'",
			s.Database, eventsTableName,
		),
	)
	var engine string
	s.NoError(row.Scan(&engine))
	s.Equal(want, engine)
}

func TestEventsTableEngine(t *testing.T) {
	suite.Run(t, new(EventsTableEngineSuite))
}
