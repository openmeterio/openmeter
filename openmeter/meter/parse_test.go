package meter_test

import (
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
		data        []byte
		err         error
		errString   string
		value       *float64
		valueStr    *string
		groupBy     map[string]string
	}{
		{
			description: "should parse event",
			meter:       meterSum,
			data:        []byte(`{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`),
			value:       lo.ToPtr(100.0),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should parse event with numeric string value",
			meter:       meterSum,
			data:        []byte(`{"duration_ms": "100", "method": "GET", "path": "/api/v1"}`),
			value:       lo.ToPtr(100.0),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should parse count as value one",
			meter:       meterCount,
			data:        []byte(`{}`),
			value:       lo.ToPtr(1.0),
			groupBy:     map[string]string{},
		},
		{
			description: "should parse unique count as string",
			meter:       meterUniqueCount,
			data:        []byte(`{"trace_id": "test_trace_id"}`),
			valueStr:    lo.ToPtr("test_trace_id"),
			groupBy:     map[string]string{},
		},
		{
			description: "should parse event with missing group by properties",
			meter:       meterSum,
			data:        []byte(`{"duration_ms": 100}`),
			value:       lo.ToPtr(100.0),
			groupBy: map[string]string{
				"method": "",
				"path":   "",
			},
		},
		{
			description: "should return error with invalid json",
			meter:       meterSum,
			data:        []byte(`{`),
			err:         meter.ErrInvalidEvent{},
			errString:   "invalid event: failed to parse event data: unexpected end of JSON input",
			groupBy:     map[string]string{},
		},
		{
			description: "should return error with data missing",
			meter:       meterSum,
			data:        []byte(`null`),
			err:         meter.ErrInvalidEvent{},
			errString:   "invalid event: null and missing value property",
			groupBy: map[string]string{
				"method": "",
				"path":   "",
			},
		},
		{
			description: "should return error with data null",
			meter:       meterSum,
			data:        []byte(`null`),
			err:         meter.ErrInvalidEvent{},
			errString:   "invalid event: null and missing value property",
			groupBy: map[string]string{
				"method": "",
				"path":   "",
			},
		},
		{
			description: "should return error with value property not found",
			meter:       meterSum,
			data:        []byte(`{"method": "GET", "path": "/api/v1"}`),
			err:         meter.ErrInvalidEvent{},
			errString:   `invalid event: missing value property: "$.duration_ms"`,
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should return error when value property is null",
			meter:       meterSum,
			data:        []byte(`{"duration_ms": null, "method": "GET", "path": "/api/v1"}`),
			err:         meter.ErrInvalidEvent{},
			errString:   "invalid event: value cannot be null",
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should return error when value property is NaN",
			meter:       meterSum,
			data:        []byte(`{"duration_ms": "NaN", "method": "GET", "path": "/api/v1"}`),
			err:         meter.ErrInvalidEvent{},
			errString:   "invalid event: value cannot be NaN",
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should return error when value property is infinity",
			meter:       meterSum,
			data:        []byte(`{"duration_ms": "Inf", "method": "GET", "path": "/api/v1"}`),
			err:         meter.ErrInvalidEvent{},
			errString:   "invalid event: value cannot be infinity",
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should return error when value property is postiive infinity",
			meter:       meterSum,
			data:        []byte(`{"duration_ms": "+Inf", "method": "GET", "path": "/api/v1"}`),
			err:         meter.ErrInvalidEvent{},
			errString:   "invalid event: value cannot be infinity",
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should return error when value property is negative infinity",
			meter:       meterSum,
			data:        []byte(`{"duration_ms": "-Inf", "method": "GET", "path": "/api/v1"}`),
			err:         meter.ErrInvalidEvent{},
			errString:   "invalid event: value cannot be infinity",
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should return error when value property cannot be parsed as number",
			meter:       meterSum,
			data:        []byte(`{"duration_ms": "not a number", "method": "GET", "path": "/api/v1"}`),
			err:         meter.ErrInvalidMeter{},
			errString:   "invalid event: value cannot be parsed as float64: not a number",
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			parsedEvent, err := meter.ParseEvent(test.meter, test.data)

			if test.err == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorAs(t, err, &test.err)
				assert.EqualError(t, err, test.errString)
			}

			assert.Equal(t, test.value, parsedEvent.Value)
			assert.Equal(t, test.valueStr, parsedEvent.ValueString)
			assert.Equal(t, test.groupBy, parsedEvent.GroupBy)
		})
	}
}
