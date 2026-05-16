package clickhouse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCreateEventsTable(t *testing.T) {
	tests := []struct {
		name string
		data createEventsTable
		want string
	}{
		{
			name: "default (legacy MergeTree, no engine config)",
			data: createEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
			},
			want: "CREATE TABLE IF NOT EXISTS openmeter.om_events (namespace String, id String, type LowCardinality(String), subject String, source String, time DateTime, data String, ingested_at DateTime, stored_at DateTime, INDEX om_events_stored_at stored_at TYPE minmax GRANULARITY 4, store_row_id String) ENGINE = MergeTree PARTITION BY toYYYYMM(time) ORDER BY (namespace, type, subject, toStartOfHour(time))",
		},
		{
			name: "explicit MergeTree",
			data: createEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Engine: EventsTableEngine{
					Type: EventsTableEngineMergeTree,
				},
			},
			want: "CREATE TABLE IF NOT EXISTS openmeter.om_events (namespace String, id String, type LowCardinality(String), subject String, source String, time DateTime, data String, ingested_at DateTime, stored_at DateTime, INDEX om_events_stored_at stored_at TYPE minmax GRANULARITY 4, store_row_id String) ENGINE = MergeTree PARTITION BY toYYYYMM(time) ORDER BY (namespace, type, subject, toStartOfHour(time))",
		},
		{
			name: "ReplicatedMergeTree with macros",
			data: createEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Engine: EventsTableEngine{
					Type:          EventsTableEngineReplicatedMergeTree,
					ZooKeeperPath: "/clickhouse/tables/{shard}/{database}/{table}",
					ReplicaName:   "{replica}",
				},
			},
			want: "CREATE TABLE IF NOT EXISTS openmeter.om_events (namespace String, id String, type LowCardinality(String), subject String, source String, time DateTime, data String, ingested_at DateTime, stored_at DateTime, INDEX om_events_stored_at stored_at TYPE minmax GRANULARITY 4, store_row_id String) ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/{table}', '{replica}') PARTITION BY toYYYYMM(time) ORDER BY (namespace, type, subject, toStartOfHour(time))",
		},
		{
			name: "ReplicatedMergeTree with ON CLUSTER (simple identifier is backtick-quoted)",
			data: createEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Engine: EventsTableEngine{
					Type:          EventsTableEngineReplicatedMergeTree,
					ZooKeeperPath: "/clickhouse/tables/{shard}/{database}/{table}",
					ReplicaName:   "{replica}",
					Cluster:       "openmeter_cluster",
				},
			},
			want: "CREATE TABLE IF NOT EXISTS openmeter.om_events ON CLUSTER `openmeter_cluster` (namespace String, id String, type LowCardinality(String), subject String, source String, time DateTime, data String, ingested_at DateTime, stored_at DateTime, INDEX om_events_stored_at stored_at TYPE minmax GRANULARITY 4, store_row_id String) ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/{table}', '{replica}') PARTITION BY toYYYYMM(time) ORDER BY (namespace, type, subject, toStartOfHour(time))",
		},
		{
			name: "ReplicatedMergeTree with hyphenated cluster name",
			data: createEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Engine: EventsTableEngine{
					Type:          EventsTableEngineReplicatedMergeTree,
					ZooKeeperPath: "/clickhouse/tables/{shard}/{database}/{table}",
					ReplicaName:   "{replica}",
					Cluster:       "prod-cluster-1",
				},
			},
			want: "CREATE TABLE IF NOT EXISTS openmeter.om_events ON CLUSTER `prod-cluster-1` (namespace String, id String, type LowCardinality(String), subject String, source String, time DateTime, data String, ingested_at DateTime, stored_at DateTime, INDEX om_events_stored_at stored_at TYPE minmax GRANULARITY 4, store_row_id String) ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/{table}', '{replica}') PARTITION BY toYYYYMM(time) ORDER BY (namespace, type, subject, toStartOfHour(time))",
		},
		{
			name: "cluster name with embedded backtick is escaped",
			data: createEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Engine: EventsTableEngine{
					Type:          EventsTableEngineReplicatedMergeTree,
					ZooKeeperPath: "/clickhouse/tables/{shard}/{database}/{table}",
					ReplicaName:   "{replica}",
					Cluster:       "weird`name",
				},
			},
			want: "CREATE TABLE IF NOT EXISTS openmeter.om_events ON CLUSTER `weird``name` (namespace String, id String, type LowCardinality(String), subject String, source String, time DateTime, data String, ingested_at DateTime, stored_at DateTime, INDEX om_events_stored_at stored_at TYPE minmax GRANULARITY 4, store_row_id String) ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/{table}', '{replica}') PARTITION BY toYYYYMM(time) ORDER BY (namespace, type, subject, toStartOfHour(time))",
		},
		{
			name: "ReplicatedMergeTree escapes single quotes and backslashes in zk path and replica name",
			data: createEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Engine: EventsTableEngine{
					Type:          EventsTableEngineReplicatedMergeTree,
					ZooKeeperPath: `/path/with'quote\and-backslash`,
					ReplicaName:   "rep'lica",
				},
			},
			want: `CREATE TABLE IF NOT EXISTS openmeter.om_events (namespace String, id String, type LowCardinality(String), subject String, source String, time DateTime, data String, ingested_at DateTime, stored_at DateTime, INDEX om_events_stored_at stored_at TYPE minmax GRANULARITY 4, store_row_id String) ENGINE = ReplicatedMergeTree('/path/with\'quote\\and-backslash', 'rep\'lica') PARTITION BY toYYYYMM(time) ORDER BY (namespace, type, subject, toStartOfHour(time))`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := tt.data.toSQL()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEventsTableEngineValidate(t *testing.T) {
	tests := []struct {
		name    string
		engine  EventsTableEngine
		wantErr string
	}{
		{
			name:   "empty defaults to MergeTree and is valid",
			engine: EventsTableEngine{},
		},
		{
			name: "explicit MergeTree is valid without zk/replica",
			engine: EventsTableEngine{
				Type: EventsTableEngineMergeTree,
			},
		},
		{
			name: "MergeTree with cluster is rejected (non-replicated ON CLUSTER produces independent tables)",
			engine: EventsTableEngine{
				Type:    EventsTableEngineMergeTree,
				Cluster: "c1",
			},
			wantErr: "cluster requires ReplicatedMergeTree",
		},
		{
			name: "ReplicatedMergeTree requires zk path",
			engine: EventsTableEngine{
				Type:        EventsTableEngineReplicatedMergeTree,
				ReplicaName: "{replica}",
			},
			wantErr: "zooKeeperPath",
		},
		{
			name: "ReplicatedMergeTree requires replica name",
			engine: EventsTableEngine{
				Type:          EventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "/p",
			},
			wantErr: "replicaName",
		},
		{
			name: "ReplicatedMergeTree with both fields is valid",
			engine: EventsTableEngine{
				Type:          EventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "/p",
				ReplicaName:   "r",
			},
		},
		{
			name: "unknown engine type rejected",
			engine: EventsTableEngine{
				Type: "AggregatingMergeTree",
			},
			wantErr: "unsupported events table engine type",
		},
		{
			name: "cluster name with hyphen is valid (ClickHouse permits hyphens; we backtick-quote)",
			engine: EventsTableEngine{
				Type:          EventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "/p",
				ReplicaName:   "r",
				Cluster:       "prod-cluster-1",
			},
		},
		{
			name: "cluster name with leading digit is valid",
			engine: EventsTableEngine{
				Type:          EventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "/p",
				ReplicaName:   "r",
				Cluster:       "1cluster",
			},
		},
		{
			name: "cluster name with embedded backtick is valid (escaped at render time)",
			engine: EventsTableEngine{
				Type:          EventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "/p",
				ReplicaName:   "r",
				Cluster:       "weird`name",
			},
		},
		{
			name: "cluster name with whitespace-only is rejected (likely typo)",
			engine: EventsTableEngine{
				Type:          EventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "/p",
				ReplicaName:   "r",
				Cluster:       "   ",
			},
			wantErr: "must not be whitespace-only",
		},
		{
			name: "ReplicatedMergeTree with whitespace-only zk path is invalid",
			engine: EventsTableEngine{
				Type:          EventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "   ",
				ReplicaName:   "{replica}",
			},
			wantErr: "zooKeeperPath",
		},
		{
			name: "ReplicatedMergeTree with whitespace-only replica name is invalid",
			engine: EventsTableEngine{
				Type:          EventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "/p",
				ReplicaName:   "\t",
			},
			wantErr: "replicaName",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.engine.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestQueryEventsTable(t *testing.T) {
	subjectFilter := "customer-1"
	idFilter := "event-id-1"
	from := time.Now()
	to := time.Now().Add(time.Hour)

	tests := []struct {
		query    queryEventsTable
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: queryEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				From:            from,
				Limit:           100,
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, ingested_at, stored_at, store_row_id FROM openmeter.om_events WHERE namespace = ? AND time >= ? ORDER BY time DESC LIMIT ?",
			wantArgs: []interface{}{"my_namespace", from.Unix(), 100},
		},
		{
			query: queryEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				From:            from,
				Limit:           100,
				Subject:         &subjectFilter,
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, ingested_at, stored_at, store_row_id FROM openmeter.om_events WHERE namespace = ? AND time >= ? AND om_events.subject IN (?) ORDER BY time DESC LIMIT ?",
			wantArgs: []interface{}{"my_namespace", from.Unix(), []string{subjectFilter}, 100},
		},
		{
			query: queryEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				From:            from,
				Limit:           100,
				ID:              &idFilter,
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, ingested_at, stored_at, store_row_id FROM openmeter.om_events WHERE namespace = ? AND time >= ? AND id LIKE ? ORDER BY time DESC LIMIT ?",
			wantArgs: []interface{}{"my_namespace", from.Unix(), "%event-id-1%", 100},
		},
		{
			query: queryEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				From:            from,
				To:              &to,
				Limit:           100,
				ID:              &idFilter,
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, ingested_at, stored_at, store_row_id FROM openmeter.om_events WHERE namespace = ? AND time >= ? AND time < ? AND id LIKE ? ORDER BY time DESC LIMIT ?",
			wantArgs: []interface{}{"my_namespace", from.Unix(), to.Unix(), "%event-id-1%", 100},
		},
		// Customer filter
		{
			query: queryEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				From:            from,
				Limit:           100,
				Customers: &[]streaming.Customer{
					customer.Customer{
						ManagedResource: models.ManagedResource{
							NamespacedModel: models.NamespacedModel{
								Namespace: "my_namespace",
							},
							ID: "customer1",
						},
						UsageAttribution: &customer.CustomerUsageAttribution{
							SubjectKeys: []string{"subject1", "subject2"},
						},
					},
					customer.Customer{
						ManagedResource: models.ManagedResource{
							NamespacedModel: models.NamespacedModel{
								Namespace: "my_namespace",
							},
							ID: "customer2",
						},
						UsageAttribution: &customer.CustomerUsageAttribution{
							SubjectKeys: []string{"subject3"},
						},
					},
				},
			},
			wantSQL:  "WITH map('subject1', 'customer1', 'subject2', 'customer1', 'subject3', 'customer2') as subject_to_customer_id SELECT id, type, subject, source, time, data, ingested_at, stored_at, store_row_id, subject_to_customer_id[om_events.subject] AS customer_id FROM openmeter.om_events WHERE namespace = ? AND time >= ? AND om_events.subject IN (?) ORDER BY time DESC LIMIT ?",
			wantArgs: []interface{}{"my_namespace", from.Unix(), []string{"subject1", "subject2", "subject3"}, 100},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			gotSql, gotArgs := tt.query.toSQL()

			assert.Equal(t, tt.wantArgs, gotArgs)
			assert.Equal(t, tt.wantSQL, gotSql)
		})
	}
}

func TestQueryEventsCount(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	tests := []struct {
		query    queryCountEvents
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: queryCountEvents{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				From:            from,
			},
			wantSQL:  "SELECT count() as count, subject FROM openmeter.om_events WHERE namespace = ? AND time >= ? GROUP BY subject",
			wantArgs: []interface{}{"my_namespace", from.Unix()},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			gotSql, gotArgs := tt.query.toSQL()

			assert.Equal(t, tt.wantArgs, gotArgs)
			assert.Equal(t, tt.wantSQL, gotSql)
		})
	}
}

func TestInsertEventsQuery(t *testing.T) {
	now := time.Now()

	query := InsertEventsQuery{
		Database:        "database",
		EventsTableName: "om_events",
		Events: []streaming.RawEvent{
			{
				Namespace:  "my_namespace",
				ID:         "1",
				Source:     "source",
				Subject:    "subject-1",
				Time:       now,
				StoredAt:   now,
				IngestedAt: now,
				Type:       "api-calls",
				Data:       `{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`,
				StoreRowID: "1",
			},
			{
				Namespace:  "my_namespace",
				ID:         "2",
				Source:     "source",
				Subject:    "subject-2",
				Time:       now,
				StoredAt:   now,
				IngestedAt: now,
				Type:       "api-calls",
				Data:       `{"duration_ms": 80, "method": "GET", "path": "/api/v1"}`,
				StoreRowID: "2",
			},
			{
				Namespace:  "my_namespace",
				ID:         "3",
				Source:     "source",
				Subject:    "subject-2",
				Time:       now,
				StoredAt:   now,
				IngestedAt: now,
				Type:       "api-calls",
				Data:       `{"duration_ms": "foo", "method": "GET", "path": "/api/v1"}`,
				StoreRowID: "3",
			},
		},
	}

	sql, args := query.ToSQL()

	assert.Equal(t, []interface{}{
		"my_namespace", "1", "api-calls", "source", "subject-1", now, `{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`, now, now, "1",
		"my_namespace", "2", "api-calls", "source", "subject-2", now, `{"duration_ms": 80, "method": "GET", "path": "/api/v1"}`, now, now, "2",
		"my_namespace", "3", "api-calls", "source", "subject-2", now, `{"duration_ms": "foo", "method": "GET", "path": "/api/v1"}`, now, now, "3",
	}, args)
	assert.Equal(t, `INSERT INTO database.om_events (namespace, id, type, source, subject, time, data, ingested_at, stored_at, store_row_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, sql)
}
