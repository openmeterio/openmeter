package meter

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/oliveagle/jsonpath"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

var _ models.GenericError = (*ErrInvalidEvent)(nil)

type ErrInvalidEvent struct {
	err error
}

func (e ErrInvalidEvent) Error() string {
	s := "invalid event"
	if e.err != nil {
		return s + ": " + e.err.Error()
	}

	return s
}

func (e ErrInvalidEvent) Unwrap() error {
	return e.err
}

func NewErrInvalidEvent(err error) error {
	return ErrInvalidEvent{err: err}
}

var _ models.GenericError = (*ErrInvalidMeter)(nil)

type ErrInvalidMeter struct {
	err error
}

func (e ErrInvalidMeter) Error() string {
	s := "invalid meter"
	if e.err != nil {
		return s + ": " + e.err.Error()
	}

	return s
}

func (e ErrInvalidMeter) Unwrap() error {
	return e.err
}

func NewErrInvalidMeter(err error) error {
	return &ErrInvalidMeter{err: err}
}

type ParsedEvent struct {
	Value       *float64
	ValueString *string
	GroupBy     map[string]string
}

func ParseEventString(meter Meter, data string) (*ParsedEvent, error) {
	return ParseEvent(meter, []byte(data))
}

// ParseEvent validates and parses an event against a meter.
func ParseEvent(meter Meter, data []byte) (*ParsedEvent, error) {
	// Parse CloudEvents data
	var (
		event interface{}
		err   error
	)

	parsedEvent := &ParsedEvent{
		GroupBy: map[string]string{},
	}

	if len(data) > 0 {
		err = json.Unmarshal(data, &event)
		if err != nil {
			return parsedEvent, NewErrInvalidEvent(fmt.Errorf("failed to parse event data: %w", err))
		}
	}

	// Parse group by fields
	parsedEvent.GroupBy = parseGroupBy(meter, event)

	// We can skip count events as they don't have value property
	if meter.Aggregation == MeterAggregationCount {
		parsedEvent.Value = lo.ToPtr(1.0)

		return parsedEvent, nil
	}

	// Non count events require value property to be present
	// If the event data is null, we return an error as value property is missing
	if event == nil {
		return parsedEvent, NewErrInvalidEvent(errors.New("null and missing value property"))
	}

	if meter.ValueProperty == nil {
		return parsedEvent, NewErrInvalidEvent(errors.New("non count meter value property is missing"))
	}

	// Get value from event data by value property
	var rawValue interface{}

	rawValue, err = jsonpath.JsonPathLookup(event, *meter.ValueProperty)
	if err != nil {
		return parsedEvent, NewErrInvalidEvent(fmt.Errorf("missing value property: %q", *meter.ValueProperty))
	}

	if rawValue == nil {
		return parsedEvent, NewErrInvalidEvent(errors.New("value cannot be null"))
	}

	// Aggregation specific value validation
	switch meter.Aggregation {
	// UNIQUE_COUNT aggregation requires string property value
	case MeterAggregationUniqueCount:
		// We convert the value to string
		parsedEvent.ValueString = lo.ToPtr(fmt.Sprintf("%v", rawValue))

		return parsedEvent, nil
	// SUM, AVG, MIN, MAX, LATEST aggregations require float64 parsable value property value
	case MeterAggregationSum, MeterAggregationAvg, MeterAggregationMin, MeterAggregationMax, MeterAggregationLatest:
		switch v := rawValue.(type) {
		case string:
			parsedValue, err := strconv.ParseFloat(v, 64)
			if err != nil {
				// TODO: omit value or make sure it's length is not too long
				return parsedEvent, NewErrInvalidEvent(fmt.Errorf("value cannot be parsed as float64: %s", v))
			}

			if err := validateFloat64(parsedValue); err != nil {
				return parsedEvent, NewErrInvalidEvent(err)
			}

			parsedEvent.Value = lo.ToPtr(parsedValue)

			return parsedEvent, nil
		case float64:
			if err := validateFloat64(v); err != nil {
				return parsedEvent, NewErrInvalidEvent(err)
			}

			parsedEvent.Value = lo.ToPtr(v)

			return parsedEvent, nil
		default:
			return parsedEvent, NewErrInvalidEvent(fmt.Errorf("unsupported value property type: %T", v))
		}
	}

	return parsedEvent, NewErrInvalidMeter(fmt.Errorf("unknown meter aggregation: %s", meter.Aggregation))
}

// valiodateFloat64 validates a float64 value
func validateFloat64(v float64) error {
	if math.IsNaN(v) {
		return errors.New("value cannot be NaN")
	}

	if math.IsInf(v, 0) {
		return errors.New("value cannot be infinity")
	}

	return nil
}

// parseGroupBy parses the group by fields from the event data
// we allow the group by fields to be missing in the event data or the data to be null
// in such cases we set the group by value to empty string
func parseGroupBy(meter Meter, data interface{}) map[string]string {
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
