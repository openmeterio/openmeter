// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
)

type ConsumerGroupMetrics struct {
	// Local consumer group handler's state
	State metric.Int64Gauge
	// Time elapsed since last state change (milliseconds)
	StateAge metric.Int64Gauge
	// Local consumer group handler's join state
	JoinState metric.Int64Gauge
	// Time elapsed since last rebalance (assign or revoke) (milliseconds)
	RebalanceAge metric.Int64Gauge
	// Total number of rebalances (assign or revoke)
	RebalanceCount metric.Int64Counter
	// Current assignment's partition count
	PartitionAssigned metric.Int64Gauge
}

func (m *ConsumerGroupMetrics) Add(ctx context.Context, stats *stats.ConsumerGroupStats, attrs ...attribute.KeyValue) {
	m.State.Record(ctx, stats.State.Int64(), metric.WithAttributes(attrs...))
	m.StateAge.Record(ctx, stats.StateAge, metric.WithAttributes(attrs...))
	m.JoinState.Record(ctx, stats.JoinState.Int64(), metric.WithAttributes(attrs...))
	m.RebalanceAge.Record(ctx, stats.RebalanceAge, metric.WithAttributes(attrs...))
	m.RebalanceCount.Add(ctx, stats.RebalanceCount, metric.WithAttributes(attrs...))
	m.PartitionAssigned.Record(ctx, stats.PartitionAssigned, metric.WithAttributes(attrs...))
}

func NewConsumerGroupMetrics(meter metric.Meter) (*ConsumerGroupMetrics, error) {
	var err error
	m := &ConsumerGroupMetrics{}

	m.State, err = meter.Int64Gauge(
		"kafka.consumer_group.state",
		metric.WithDescription("Local consumer group handler's state: [Unknown(-1), Init(0), Term(1), QueryCoord(2), WaitCoord(3), WaitBroker(4), WaitBrokerTransport(5), Up(6)]"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.consumer_group.state: %w", err)
	}

	m.StateAge, err = meter.Int64Gauge(
		"kafka.consumer_group.state_age",
		metric.WithDescription("Time elapsed since last state change (milliseconds)"),
		metric.WithUnit("{milliseconds}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.consumer_group.state_age: %w", err)
	}

	m.JoinState, err = meter.Int64Gauge(
		"kafka.consumer_group.join_state",
		metric.WithDescription("Local consumer group handler's join state: [Unknown(-1), Init(0), WaitJoin(1), WaitMetadata(2), WaitSync(3), WaitAssignCall(4), WaitUnassignCall(5), WaitUnassignToComplete(6), WaitIncrementalUnassignToComplete(7), Steady(8)]"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.consumer_group.join_state: %w", err)
	}

	m.RebalanceAge, err = meter.Int64Gauge(
		"kafka.consumer_group.rebalance_age",
		metric.WithDescription("Time elapsed since last rebalance (assign or revoke) (milliseconds)"),
		metric.WithUnit("{milliseconds}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.consumer_group.rebalance_age: %w", err)
	}

	m.RebalanceCount, err = meter.Int64Counter(
		"kafka.consumer_group.rebalance_count",
		metric.WithDescription("Time elapsed since last rebalance (assign or revoke) (milliseconds)"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.consumer_group.rebalance_count: %w", err)
	}

	m.PartitionAssigned, err = meter.Int64Gauge(
		"kafka.consumer_group.partitions_assigned",
		metric.WithDescription("Current assignment's partition count"),
		metric.WithUnit("{partition}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.consumer_group.partitions_assigned: %w", err)
	}

	return m, nil
}
