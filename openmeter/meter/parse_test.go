package meter_test

import (
	"errors"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestParseEvent(t *testing.T) {
	meterSum := meter.Meter{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(),
			NamespacedModel: models.NamespacedModel{
				Namespace: "default",
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Test meter",
		},
		Key:           "m1",
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "api-calls",
		ValueProperty: lo.ToPtr("$.duration_ms"),
		GroupBy: map[string]string{
			"method": "$.method",
			"path":   "$.path",
		},
	}

	meterCount := meter.Meter{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(),
			NamespacedModel: models.NamespacedModel{
				Namespace: "default",
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Test meter",
		},
		Key:         "m2",
		Aggregation: meter.MeterAggregationCount,
		EventType:   "api-calls",
	}

	meterUniqueCount := meter.Meter{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(),
			NamespacedModel: models.NamespacedModel{
				Namespace: "default",
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Test meter",
		},
		Key:           "m3",
		Aggregation:   meter.MeterAggregationUniqueCount,
		EventType:     "spans",
		ValueProperty: lo.ToPtr("$.trace_id"),
	}

	tests := []struct {
		description string
		meter       meter.Meter
		data        string
		err         error
		value       *float64
		valueStr    *string
		groupBy     map[string]string
	}{
		{
			description: "should parse event",
			meter:       meterSum,
			data:        `{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`,
			value:       lo.ToPtr(100.0),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should parse event with numeric string value",
			meter:       meterSum,
			data:        `{"duration_ms": "100", "method": "GET", "path": "/api/v1"}`,
			value:       lo.ToPtr(100.0),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should parse count as value one",
			meter:       meterCount,
			data:        `{}`,
			value:       lo.ToPtr(1.0),
			groupBy:     map[string]string{},
		},
		{
			description: "should parse unique count as string",
			meter:       meterUniqueCount,
			data:        `{"trace_id": "test_trace_id"}`,
			valueStr:    lo.ToPtr("test_trace_id"),
			groupBy:     map[string]string{},
		},
		{
			description: "should parse event with missing group by properties",
			meter:       meterSum,
			data:        `{"duration_ms": 100}`,
			value:       lo.ToPtr(100.0),
			groupBy: map[string]string{
				"method": "",
				"path":   "",
			},
		},
		{
			description: "should return error with invalid json",
			meter:       meterSum,
			data:        `{`,
			err:         errors.New("cannot unmarshal event data"),
			groupBy:     map[string]string{},
		},
		{
			description: "should return error with data missing",
			meter:       meterSum,
			data:        `null`,
			err:         errors.New("event data is null and missing value property"),
			groupBy: map[string]string{
				"method": "",
				"path":   "",
			},
		},
		{
			description: "should return error with data null",
			meter:       meterSum,
			data:        `null`,
			err:         errors.New("event data is null and missing value property"),
			groupBy: map[string]string{
				"method": "",
				"path":   "",
			},
		},
		{
			description: "should return error with value property not found",
			meter:       meterSum,
			data:        `{"method": "GET", "path": "/api/v1"}`,
			err:         errors.New("event data is missing value property at \"$.duration_ms\""),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should return error when value property is null",
			meter:       meterSum,
			data:        `{"duration_ms": null, "method": "GET", "path": "/api/v1"}`,
			err:         errors.New("event data value cannot be null"),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should return error when value property cannot be parsed as number",
			meter:       meterSum,
			data:        `{"duration_ms": "not a number", "method": "GET", "path": "/api/v1"}`,
			err:         errors.New("event data value cannot be parsed as float64: not a number"),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.description, func(t *testing.T) {
			value, valueStr, groupBy, err := meter.ParseEvent(test.meter, test.data)

			assert.Equal(t, test.err, err)
			assert.Equal(t, test.value, value)
			assert.Equal(t, test.valueStr, valueStr)
			assert.Equal(t, test.groupBy, groupBy)
		})
	}
}
