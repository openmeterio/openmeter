package meter

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/oliveagle/jsonpath"

	"github.com/openmeterio/openmeter/api/models"
)

// ParseEvent validates and parses an event against a meter.
func ParseEvent(meter Meter, ev event.Event) (*float64, *string, map[string]string, error) {
	// Parse CloudEvents data
	var data interface{}

	if len(ev.Data()) > 0 {
		err := ev.DataAs(&data)
		if err != nil {
			return nil, nil, map[string]string{}, errors.New("cannot unmarshal event data")
		}
	}

	// Parse group by fields
	groupBy := parseGroupBy(meter, data)

	// We can skip count events as they don't have value property
	if meter.Aggregation == MeterAggregationCount {
		value := 1.0
		return &value, nil, groupBy, nil
	}

	// Non count events require value property to be present
	// If the event data is null, we return an error as value property is missing
	if data == nil {
		return nil, nil, groupBy, errors.New("event data is null and missing value property")
	}

	// Get value from event data by value property
	rawValue, err := jsonpath.JsonPathLookup(data, meter.ValueProperty)
	if err != nil {
		return nil, nil, groupBy, fmt.Errorf("event data is missing value property at %q", meter.ValueProperty)
	}

	if rawValue == nil {
		return nil, nil, groupBy, errors.New("event data value cannot be null")
	}

	// Aggregation specific value validation
	switch meter.Aggregation {
	// UNIQUE_COUNT aggregation requires string property value
	case MeterAggregationUniqueCount:
		// We convert the value to string
		val := fmt.Sprintf("%v", rawValue)
		return nil, &val, groupBy, nil

	// SUM, AVG, MIN, MAX aggregations require float64 parsable value property value
	case MeterAggregationSum, MeterAggregationAvg, MeterAggregationMin, MeterAggregationMax:
		switch value := rawValue.(type) {
		case string:
			val, err := strconv.ParseFloat(value, 64)
			if err != nil {
				// TODO: omit value or make sure it's length is not too long
				return nil, nil, groupBy, fmt.Errorf("event data value cannot be parsed as float64: %s", value)
			}

			return &val, nil, groupBy, nil

		case float64:
			return &value, nil, groupBy, nil

		default:
			return nil, nil, groupBy, errors.New("event data value property cannot be parsed")
		}
	}

	return nil, nil, groupBy, fmt.Errorf("unknown meter aggregation: %s", meter.Aggregation)
}

// parseGroupBy parses the group by fields from the event data
// we allow the group by fields to be missing in the event data or the data to be null
// in such cases we set the group by value to empty string
func parseGroupBy(meter models.Meter, data interface{}) map[string]string {
	groupBy := map[string]string{}

	// Group by fields
	for groupByKey, groupByPath := range meter.GroupBy {
		var groupByValue string

		rawGroupBy, err := jsonpath.JsonPathLookup(data, groupByPath)
		if err != nil {
			groupByValue = ""
		} else {
			groupByValue = fmt.Sprintf("%v", rawGroupBy)
		}

		groupBy[groupByKey] = groupByValue
	}

	return groupBy
}
