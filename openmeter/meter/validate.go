package meter

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/oliveagle/jsonpath"
)

// ValidateEvent validates an event against a meter.
func ValidateEvent(meter Meter, ev event.Event) error {
	// Parse CloudEvents data
	var data interface{}

	err := ev.DataAs(&data)
	if err != nil {
		return errors.New("cannot unmarshal event data")
	}

	// We can skip count events as they don't have value property
	if meter.Aggregation == MeterAggregationCount {
		return nil
	}

	// Get value from event data by value property
	rawValue, err := jsonpath.JsonPathLookup(data, meter.ValueProperty)
	if err != nil {
		return fmt.Errorf("event data is missing value property at %q", meter.ValueProperty)
	}

	if rawValue == nil {
		return errors.New("event data value cannot be null")
	}

	// Aggregation specific value validation
	switch meter.Aggregation {
	// UNIQUE_COUNT aggregation requires string property value
	case MeterAggregationUniqueCount:
		switch rawValue.(type) {
		case string, float64:
			// No need to do anything

		default:
			return errors.New("event data value property must be string for unique count aggregation")
		}

	// SUM, AVG, MIN, MAX aggregations require float64 parsable value property value
	case MeterAggregationSum, MeterAggregationAvg, MeterAggregationMin, MeterAggregationMax:
		switch value := rawValue.(type) {
		case string:
			_, err = strconv.ParseFloat(value, 64)
			if err != nil {
				// TODO: omit value or make sure it's length is not too long
				return fmt.Errorf("event data value cannot be parsed as float64: %s", value)
			}

		case float64:
			// No need to do anything

		default:
			return errors.New("event data value property cannot be parsed")
		}
	}

	return nil
}
