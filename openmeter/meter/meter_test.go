package meter

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestMeterValidation(t *testing.T) {
	tests := []struct {
		description string
		meter       Meter
		error       error
	}{
		{
			description: "valid meter",
			meter: Meter{
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
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
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
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
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
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
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
				Key:           "slug-test",
				Aggregation:   MeterAggregationCount,
				EventType:     "event-type-test",
				ValueProperty: lo.ToPtr("$.my_property"),
				GroupBy:       map[string]string{"test_group": "$.test_group"},
			},
			error: fmt.Errorf("meter value property is not allowed when the aggregation is count"),
		},
		{
			description: "key is empty",
			meter: Meter{
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
				Aggregation: MeterAggregationCount,
				EventType:   "event-type-test",
				GroupBy:     map[string]string{"test_group": "$.test_group"},
			},
			error: fmt.Errorf("meter key is required"),
		},
		{
			description: "aggregation is empty",
			meter: Meter{
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
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
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
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
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
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
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
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
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
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
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
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
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
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
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
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
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: ulid.Make().String(),
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Test meter",
				},
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

func Test_EventTypeFilter(t *testing.T) {
	var reservedEventTypePatterns []*EventTypePattern

	patterns := []string{
		`^om\.`,
		`^_\.`,
	}

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		require.NoErrorf(t, err, "invalid regexp pattern '%s'", pattern)

		reservedEventTypePatterns = append(reservedEventTypePatterns, re)
	}

	validator := NewEventTypeValidator(reservedEventTypePatterns)

	tests := []struct {
		name      string
		eventType string

		expectedMatch bool
	}{
		{
			name:          "Random",
			eventType:     "event-type-1",
			expectedMatch: false,
		},
		{
			name:          "Openmeter",
			eventType:     "om.event-type-1",
			expectedMatch: true,
		},
		{
			name:          "System",
			eventType:     "_.event-type-1",
			expectedMatch: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator(test.eventType)
			assert.Equal(t, test.expectedMatch, err != nil)
		})
	}
}
