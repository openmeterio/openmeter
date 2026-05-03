package clickhouse

import (
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/suite"

	progressmanager "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// EventsTableEngineSuite drives the engine-config code path against a live
// ClickHouse instance. It is gated by TEST_CLICKHOUSE_DSN like the other
// integration suites in this package.
//
// The ReplicatedMergeTree variant additionally requires a Keeper-backed
// cluster; it is gated by TEST_CLICKHOUSE_REPLICATED=1 and the docker-compose
// stack at the repository root (see `make up-replicated`). The CI single-node
// suite skips it; local runs against the replicated stack exercise the full
// code path.
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

// TestReplicatedMergeTreeEngine exercises the ReplicatedMergeTree code path
// end-to-end against a Keeper-backed cluster. It is skipped unless
// TEST_CLICKHOUSE_REPLICATED=1 is set; bring the cluster up with
// `make up-replicated` before running, and point TEST_CLICKHOUSE_DSN at one
// of the replicas (port 39000 or 39001 by default).
func (s *EventsTableEngineSuite) TestReplicatedMergeTreeEngine() {
	if os.Getenv("TEST_CLICKHOUSE_REPLICATED") != "1" {
		s.T().Skip("TEST_CLICKHOUSE_REPLICATED is not set; skipping ReplicatedMergeTree test")
	}

	t := s.T()
	ctx := t.Context()

	// CHTestSuite.CreateTempDatabase only creates the database on the
	// connected node. ReplicatedMergeTree with ON CLUSTER propagates the DDL
	// to every replica, so the database must exist on all of them. Re-issue
	// the CREATE DATABASE on the cluster — IF NOT EXISTS makes it a no-op on
	// the node that already has it.
	s.NoError(s.ClickHouse.Exec(ctx, fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS %s ON CLUSTER openmeter_cluster", s.Database,
	)))
	// Drop on the cluster so the DB is removed from every replica, not just
	// the one CHTestSuite.TearDownTest cleans up locally. We use defer here
	// (not t.Cleanup) because testify's TearDownTest closes s.ClickHouse and
	// nils the field before subtest t.Cleanup callbacks fire — defer runs
	// while the connection is still open.
	conn, dbName := s.ClickHouse, s.Database
	defer func() {
		_ = conn.Exec(t.Context(), fmt.Sprintf(
			"DROP DATABASE IF EXISTS %s ON CLUSTER openmeter_cluster SYNC", dbName,
		))
	}()

	connector, err := New(ctx, Config{
		Logger:          slog.Default(),
		ClickHouse:      s.ClickHouse,
		Database:        s.Database,
		EventsTableName: eventsTableName,
		EventsTableEngine: EventsTableEngine{
			Type:          EventsTableEngineReplicatedMergeTree,
			ZooKeeperPath: "/clickhouse/tables/{shard}/{database}/{table}",
			ReplicaName:   "{replica}",
			Cluster:       "openmeter_cluster",
		},
		ProgressManager: progressmanager.NewMockProgressManager(),
	})
	s.NoError(err, "failed to create connector with ReplicatedMergeTree engine")
	s.NotNil(connector)

	// system.tables.engine reports the bare engine name without arguments.
	s.assertEngine("ReplicatedMergeTree")

	// Table must exist on every replica in the cluster — this is what `ON
	// CLUSTER` is supposed to guarantee.
	var replicaCount uint64
	row := s.ClickHouse.QueryRow(ctx, fmt.Sprintf(`
		SELECT count() FROM clusterAllReplicas('openmeter_cluster', system.tables)
		WHERE database = '%s' AND name = '%s'`,
		s.Database, eventsTableName,
	))
	s.NoError(row.Scan(&replicaCount))
	s.EqualValues(2, replicaCount, "events table should exist on both replicas")

	// End-to-end replication probe: insert via the connector (which targets
	// node-01) and read the row back from node-02. If the {shard}/{replica}
	// macros were not substituted by Keeper, the second replica wouldn't
	// receive the data and the read would return zero rows.
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

	node02Opts, err := clickhouse.ParseDSN("clickhouse://default:default@127.0.0.1:39001/openmeter")
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
