package testutils

import (
	"testing"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewTestMeters(t *testing.T, namespace string) []meter.Meter {
	t.Helper()

	return []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: NewTestULID(t),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Test Requests Meter",
			},
			Key:         "api_requests_total",
			Aggregation: meter.MeterAggregationCount,
			EventType:   "request",
			GroupBy: map[string]string{
				"method": "$.method",
				"path":   "$.path",
			},
		},
		{
			ManagedResource: models.ManagedResource{
				ID: NewTestULID(t),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Test Tokens Meter",
			},
			Key:           "tokens_total",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "prompt",
			ValueProperty: lo.ToPtr("$.tokens"),
			GroupBy: map[string]string{
				"model": "$.model",
				"type":  "$.type",
			},
		},
		{
			ManagedResource: models.ManagedResource{
				ID: NewTestULID(t),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Test Workloads Meter",
			},
			Key:           "workload_runtime_duration_seconds",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "workload",
			ValueProperty: lo.ToPtr("$.duration_seconds"),
			GroupBy: map[string]string{
				"region":        "$.region",
				"zone":          "$.zone",
				"instance_type": "$.instance_type",
			},
		},
	}
}
