package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClickhouseEventsTableEngineConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ClickhouseEventsTableEngineConfig
		wantErr string
	}{
		{
			name: "empty type defaults to MergeTree and is valid",
			cfg:  ClickhouseEventsTableEngineConfig{},
		},
		{
			name: "explicit MergeTree is valid",
			cfg: ClickhouseEventsTableEngineConfig{
				Type: ClickhouseEventsTableEngineMergeTree,
			},
		},
		{
			name: "MergeTree with cluster is rejected (non-replicated ON CLUSTER produces independent tables)",
			cfg: ClickhouseEventsTableEngineConfig{
				Type:    ClickhouseEventsTableEngineMergeTree,
				Cluster: "c1",
			},
			wantErr: "cluster requires ReplicatedMergeTree",
		},
		{
			name: "ReplicatedMergeTree without zk path is invalid",
			cfg: ClickhouseEventsTableEngineConfig{
				Type:        ClickhouseEventsTableEngineReplicatedMergeTree,
				ReplicaName: "{replica}",
			},
			wantErr: "zooKeeperPath",
		},
		{
			name: "ReplicatedMergeTree without replica is invalid",
			cfg: ClickhouseEventsTableEngineConfig{
				Type:          ClickhouseEventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "/p",
			},
			wantErr: "replicaName",
		},
		{
			name: "ReplicatedMergeTree fully populated is valid",
			cfg: ClickhouseEventsTableEngineConfig{
				Type:          ClickhouseEventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "/clickhouse/tables/{shard}/{database}/{table}",
				ReplicaName:   "{replica}",
				Cluster:       "openmeter_cluster",
			},
		},
		{
			name: "unsupported engine type is rejected",
			cfg: ClickhouseEventsTableEngineConfig{
				Type: "SharedMergeTree",
			},
			wantErr: "unsupported events table engine type",
		},
		{
			name: "cluster name with hyphen is valid (backtick-quoted at render time)",
			cfg: ClickhouseEventsTableEngineConfig{
				Type:          ClickhouseEventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "/p",
				ReplicaName:   "{replica}",
				Cluster:       "prod-cluster-1",
			},
		},
		{
			name: "cluster name with whitespace-only is rejected",
			cfg: ClickhouseEventsTableEngineConfig{
				Type:          ClickhouseEventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "/p",
				ReplicaName:   "{replica}",
				Cluster:       "   ",
			},
			wantErr: "must not be whitespace-only",
		},
		{
			name: "ReplicatedMergeTree with whitespace-only zk path is invalid",
			cfg: ClickhouseEventsTableEngineConfig{
				Type:          ClickhouseEventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "  ",
				ReplicaName:   "{replica}",
			},
			wantErr: "zooKeeperPath",
		},
		{
			name: "ReplicatedMergeTree with whitespace-only replica is invalid",
			cfg: ClickhouseEventsTableEngineConfig{
				Type:          ClickhouseEventsTableEngineReplicatedMergeTree,
				ZooKeeperPath: "/p",
				ReplicaName:   "\t",
			},
			wantErr: "replicaName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestClickhouseEventsTableEngineConfigResolvedType(t *testing.T) {
	assert.Equal(t,
		ClickhouseEventsTableEngineMergeTree,
		ClickhouseEventsTableEngineConfig{}.ResolvedType(),
	)
	assert.Equal(t,
		ClickhouseEventsTableEngineReplicatedMergeTree,
		ClickhouseEventsTableEngineConfig{Type: ClickhouseEventsTableEngineReplicatedMergeTree}.ResolvedType(),
	)
}
