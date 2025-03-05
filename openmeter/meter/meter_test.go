package meter

import (
	"fmt"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestWindowSizeFromDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  WindowSize
		error error
	}{
		{
			input: time.Minute,
			want:  WindowSizeMinute,
			error: nil,
		},
		{
			input: time.Hour,
			want:  WindowSizeHour,
			error: nil,
		},
		{
			input: 24 * time.Hour,
			want:  WindowSizeDay,
			error: nil,
		},
		{
			input: 2 * time.Minute,
			want:  "",
			error: fmt.Errorf("invalid window size duration: %s", 2*time.Minute),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := WindowSizeFromDuration(tt.input)
			if err != nil {
				if tt.error == nil {
					t.Error(err)
				}

				assert.Equal(t, tt.error, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMeterValidation(t *testing.T) {
	tests := []struct {
		description string
		meter       Meter
		error       error
	}{
		{
			description: "valid meter",
			meter: Meter{
				Key:           "slug-test",
				Aggregation:   MeterAggregationSum,
				EventType:     "event-type-test",
				ValueProperty: lo.ToPtr("$.my_property"),
				GroupBy:       map[string]string{"test_group": "$.test_group"},
			},
			error: nil,
		},
		{
			description: "valid without group by",
			meter: Meter{
				Key:           "slug-test",
				Aggregation:   MeterAggregationSum,
				EventType:     "event-type-test",
				ValueProperty: lo.ToPtr("$.my_property"),
			},
			error: nil,
		},
		{
			description: "count is valid without value property",
			meter: Meter{
				Key:         "slug-test",
				Aggregation: MeterAggregationCount,
				EventType:   "event-type-test",
				GroupBy:     map[string]string{"test_group": "$.test_group"},
			},
			error: nil,
		},
		{
			description: "count is invalid with value property",
			meter: Meter{
				Key:           "slug-test",
				Aggregation:   MeterAggregationCount,
				EventType:     "event-type-test",
				ValueProperty: lo.ToPtr("$.my_property"),
				GroupBy:       map[string]string{"test_group": "$.test_group"},
			},
			error: fmt.Errorf("meter value property is not allowed when the aggregation is count"),
		},
		{
			description: "slug is empty",
			meter: Meter{
				Aggregation: MeterAggregationCount,
				EventType:   "event-type-test",
				GroupBy:     map[string]string{"test_group": "$.test_group"},
			},
			error: fmt.Errorf("meter slug is required"),
		},
		{
			description: "aggregation is empty",
			meter: Meter{
				Key:           "slug-test",
				EventType:     "event-type-test",
				ValueProperty: lo.ToPtr("$.my_property"),
				GroupBy:       map[string]string{"test_group": "$.test_group"},
			},
			error: fmt.Errorf("meter aggregation is required"),
		},
		{
			description: "window size is empty",
			meter: Meter{
				Key:         "slug-test",
				Aggregation: MeterAggregationCount,
				EventType:   "event-type-test",
				GroupBy:     map[string]string{"test_group": "$.test_group"},
			},
			error: nil,
		},
		{
			description: "event type is empty",
			meter: Meter{
				Key:           "slug-test",
				Aggregation:   MeterAggregationSum,
				ValueProperty: lo.ToPtr("$.my_property"),
				GroupBy:       map[string]string{"test_group": "$.test_group"},
			},
			error: fmt.Errorf("meter event type is required"),
		},
		{
			description: "missing value property",
			meter: Meter{
				Key:         "slug-test",
				Aggregation: MeterAggregationSum,
				EventType:   "event-type-test",
				GroupBy:     map[string]string{"test_group": "$.test_group"},
			},
			error: fmt.Errorf("meter value property is required"),
		},
		{
			description: "invalid value property",
			meter: Meter{
				Key:           "slug-test",
				Aggregation:   MeterAggregationSum,
				EventType:     "event-type-test",
				ValueProperty: lo.ToPtr("invalid"),
				GroupBy:       map[string]string{"test_group": "$.test_group"},
			},
			error: fmt.Errorf("meter value property must start with $"),
		},
		{
			description: "invalid group by key",
			meter: Meter{
				Key:           "slug-test",
				Aggregation:   MeterAggregationSum,
				EventType:     "event-type-test",
				ValueProperty: lo.ToPtr("$.my_property"),
				GroupBy:       map[string]string{"in-valid": "$.test_group"},
			},
			error: fmt.Errorf("meter group by key in-valid is invalid, only alphanumeric and underscore characters are allowed"),
		},
		{
			description: "invalid group by key",
			meter: Meter{
				Key:           "slug-test",
				Aggregation:   MeterAggregationSum,
				EventType:     "event-type-test",
				ValueProperty: lo.ToPtr("$.my_property"),
				GroupBy:       map[string]string{"": "$.test_group"},
			},
			error: fmt.Errorf("meter group by key cannot be empty"),
		},
		{
			description: "invalid group by property",
			meter: Meter{
				Key:           "slug-test",
				Aggregation:   MeterAggregationSum,
				EventType:     "event-type-test",
				ValueProperty: lo.ToPtr("$.my_property"),
				GroupBy:       map[string]string{"test_group": "invalid"},
			},
			error: fmt.Errorf("meter group by value must start with $ for key test_group"),
		},
		{
			description: "value property cannot be in the group by",
			meter: Meter{
				Key:           "slug-test",
				Aggregation:   MeterAggregationUniqueCount,
				EventType:     "event-type-test",
				ValueProperty: lo.ToPtr("$.my_property"),
				GroupBy:       map[string]string{"test_group": "$.my_property"},
			},
			error: fmt.Errorf("meter group by value test_group cannot be the same as value property"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			err := tt.meter.Validate()

			if tt.error == nil && err != nil {
				t.Error(err)
			}

			if tt.error != nil && err == nil {
				t.Errorf("expected error %v, got nil", tt.error)
			}
		})
	}
}
