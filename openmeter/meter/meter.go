package meter

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

var groupByKeyRegExp = regexp.MustCompile(`^[a-zA-Z_][0-9a-zA-Z_]*$`)

type EventTypePattern = regexp.Regexp

type MeterAggregation string

// Note: keep values up to date in the meter package
const (
	MeterAggregationSum         MeterAggregation = "SUM"
	MeterAggregationCount       MeterAggregation = "COUNT"
	MeterAggregationAvg         MeterAggregation = "AVG"
	MeterAggregationMin         MeterAggregation = "MIN"
	MeterAggregationMax         MeterAggregation = "MAX"
	MeterAggregationUniqueCount MeterAggregation = "UNIQUE_COUNT"
	MeterAggregationLatest      MeterAggregation = "LATEST"
)

// Values provides list valid values for Enum
func (MeterAggregation) Values() (kinds []string) {
	for _, s := range []MeterAggregation{
		MeterAggregationSum,
		MeterAggregationCount,
		MeterAggregationAvg,
		MeterAggregationMin,
		MeterAggregationMax,
		MeterAggregationUniqueCount,
		MeterAggregationLatest,
	} {
		kinds = append(kinds, string(s))
	}
	return kinds
}

func (MeterAggregation) IsValid(input string) bool {
	m := MeterAggregation("")

	for _, v := range m.Values() {
		if v == input {
			return true
		}
	}

	return false
}

type WindowSize string

// Note: keep values up to date in the meter package
const (
	// WindowSizeSecond is the size of the window in seconds, this is possible to use for streaming queries
	// but not exposed to the metering API as this seems to be an overkill for most external use-cases.
	WindowSizeSecond WindowSize = "SECOND"
	WindowSizeMinute WindowSize = "MINUTE"
	WindowSizeHour   WindowSize = "HOUR"
	WindowSizeDay    WindowSize = "DAY"
	WindowSizeMonth  WindowSize = "MONTH"
)

// Values provides list valid values for Enum
func (WindowSize) Values() (kinds []string) {
	for _, s := range []WindowSize{
		WindowSizeSecond,
		WindowSizeMinute,
		WindowSizeHour,
		WindowSizeDay,
		WindowSizeMonth,
	} {
		kinds = append(kinds, string(s))
	}
	return kinds
}

func (w WindowSize) AddTo(t time.Time) (time.Time, error) {
	switch w {
	case WindowSizeSecond:
		return t.Add(time.Second), nil
	case WindowSizeMinute:
		return t.Add(time.Minute), nil
	case WindowSizeHour:
		return t.Add(time.Hour), nil
	case WindowSizeDay:
		return t.AddDate(0, 0, 1), nil
	case WindowSizeMonth:
		return t.AddDate(0, 1, 0), nil
	default:
		return time.Time{}, fmt.Errorf("invalid window size: %s", w)
	}
}

func (w WindowSize) Truncate(t time.Time) (time.Time, error) {
	switch w {
	case WindowSizeSecond:
		return t.Truncate(time.Second), nil
	case WindowSizeMinute:
		return t.Truncate(time.Minute), nil
	case WindowSizeHour:
		return t.Truncate(time.Hour), nil
	case WindowSizeDay:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), nil
	case WindowSizeMonth:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()), nil
	default:
		return time.Time{}, fmt.Errorf("invalid window size: %s", w)
	}
}

// OrderBy is the order by clause for features
type OrderBy string

const (
	OrderByKey         OrderBy = "key"
	OrderByName        OrderBy = "name"
	OrderByAggregation OrderBy = "aggregation"
	OrderByCreatedAt   OrderBy = "createdAt"
	OrderByUpdatedAt   OrderBy = "updatedAt"
)

func (f OrderBy) Values() []OrderBy {
	return []OrderBy{
		OrderByKey,
		OrderByName,
		OrderByAggregation,
		OrderByCreatedAt,
		OrderByUpdatedAt,
	}
}

// Meter is the meter model
type Meter struct {
	models.ManagedResource `mapstructure:",squash"`
	models.Metadata
	models.Annotations

	Key           string `mapstructure:"slug"`
	Aggregation   MeterAggregation
	EventType     string
	EventFrom     *time.Time
	ValueProperty *string
	GroupBy       map[string]string
}

func (m1 Meter) Equal(m2 Meter) error {
	if m1.Namespace != m2.Namespace {
		return errors.New("namespace mismatch")
	}

	if m1.Key != m2.Key {
		return errors.New("key mismatch")
	}

	if m1.Name != m2.Name {
		return errors.New("name mismatch")
	}

	if m1.Description != nil && m2.Description != nil {
		if *m1.Description != *m2.Description {
			return errors.New("description mismatch")
		}
	}

	if m1.Description == nil && m2.Description != nil {
		return errors.New("description mismatch")
	}

	if m1.Description != nil && m2.Description == nil {
		return errors.New("description mismatch")
	}

	if m1.Aggregation != m2.Aggregation {
		return errors.New("aggregation mismatch")
	}

	if m1.EventType != m2.EventType {
		return errors.New("event type mismatch")
	}

	if m1.ValueProperty != nil && m2.ValueProperty != nil {
		if *m1.ValueProperty != *m2.ValueProperty {
			return errors.New("value property mismatch")
		}
	}

	if m1.ValueProperty == nil && m2.ValueProperty != nil {
		return errors.New("value property mismatch")
	}

	if m1.ValueProperty != nil && m2.ValueProperty == nil {
		return errors.New("value property mismatch")
	}

	if len(m1.GroupBy) != len(m2.GroupBy) {
		return errors.New("group by mismatch")
	}

	for key, value := range m1.GroupBy {
		if m2Value, ok := m2.GroupBy[key]; !ok || value != m2Value {
			return errors.New("group by mismatch")
		}
	}

	if !m1.Metadata.Equal(m2.Metadata) {
		return errors.New("metadata mismatch")
	}

	if !m1.Annotations.Equal(m2.Annotations) {
		return errors.New("annotations mismatch")
	}

	return nil
}

func (m *Meter) Validate() error {
	var errs []error

	if err := m.ManagedResource.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid managed resource: %w", err))
	}

	if m.Key == "" {
		errs = append(errs, errors.New("meter key is required"))
	}

	if m.EventType == "" {
		errs = append(errs, errors.New("meter event type is required"))
	}

	if m.EventFrom != nil && m.EventFrom.IsZero() {
		errs = append(errs, errors.New("meter event from must not be zero"))
	}

	if m.Aggregation == "" {
		errs = append(errs, errors.New("meter aggregation is required"))
	}

	// Validate aggregation
	if err := validateMeterAggregation(m.ValueProperty, m.Aggregation); err != nil {
		errs = append(errs, err)
	}

	// Validate group by values
	if err := validateMeterGroupBy(m.ValueProperty, m.GroupBy); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// validateMeterAggregation validates the aggregation value
func validateMeterAggregation(valueProperty *string, aggregation MeterAggregation) error {
	// ValueProperty is required for all aggregations except count
	if aggregation == MeterAggregationCount {
		if valueProperty != nil {
			return errors.New("meter value property is not allowed when the aggregation is count")
		}
	} else {
		if valueProperty == nil {
			return errors.New("meter value property is required when the aggregation is not count")
		}

		if *valueProperty == "" {
			return errors.New("meter value property cannot be empty when the aggregation is not count")
		}

		if !strings.HasPrefix(*valueProperty, "$") {
			return errors.New("meter value property must start with $")
		}
	}

	return nil
}

// validateMeterGroupBy validates the group by values
func validateMeterGroupBy(valueProperty *string, groupBy map[string]string) error {
	for key, value := range groupBy {
		if !strings.HasPrefix(value, "$") {
			return fmt.Errorf("meter group by value must start with $ for key %s", key)
		}
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("meter group by key cannot be empty")
		}
		if !groupByKeyRegExp.MatchString(key) {
			return fmt.Errorf("meter group by key %s is invalid, only alphanumeric and underscore characters are allowed", key)
		}
		if valueProperty != nil && value == *valueProperty {
			return fmt.Errorf("meter group by value %s cannot be the same as value property", key)
		}
		// keys must be unique
		seen := make(map[string]struct{}, len(groupBy))
		if _, ok := seen[key]; ok {
			return fmt.Errorf("meter group by key %s is not unique", key)
		}
		seen[key] = struct{}{}
	}

	return nil
}

// MeterQueryRow returns a single row from the meter dataset.
type MeterQueryRow struct {
	Value       float64            `json:"value"`
	WindowStart time.Time          `json:"windowStart"`
	WindowEnd   time.Time          `json:"windowEnd"`
	Subject     *string            `json:"subject"`
	CustomerID  *string            `json:"customerId"`
	GroupBy     map[string]*string `json:"groupBy"`
}
