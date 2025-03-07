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

type MeterAggregation string

// Note: keep values up to date in the meter package
const (
	MeterAggregationSum         MeterAggregation = "SUM"
	MeterAggregationCount       MeterAggregation = "COUNT"
	MeterAggregationAvg         MeterAggregation = "AVG"
	MeterAggregationMin         MeterAggregation = "MIN"
	MeterAggregationMax         MeterAggregation = "MAX"
	MeterAggregationUniqueCount MeterAggregation = "UNIQUE_COUNT"
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
	} {
		kinds = append(kinds, string(s))
	}
	return
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
	WindowSizeMinute WindowSize = "MINUTE"
	WindowSizeHour   WindowSize = "HOUR"
	WindowSizeDay    WindowSize = "DAY"
)

// Values provides list valid values for Enum
func (WindowSize) Values() (kinds []string) {
	for _, s := range []WindowSize{
		WindowSizeMinute,
		WindowSizeHour,
		WindowSizeDay,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

func (w WindowSize) AddTo(t time.Time) (time.Time, error) {
	switch w {
	case WindowSizeMinute:
		return t.Add(time.Minute), nil
	case WindowSizeHour:
		return t.Add(time.Hour), nil
	case WindowSizeDay:
		return t.AddDate(0, 0, 1), nil
	default:
		return time.Time{}, fmt.Errorf("invalid window size: %s", w)
	}
}

func (w WindowSize) Truncate(t time.Time) (time.Time, error) {
	switch w {
	case WindowSizeMinute:
		return t.Truncate(time.Minute), nil
	case WindowSizeHour:
		return t.Truncate(time.Hour), nil
	case WindowSizeDay:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), nil
	default:
		return time.Time{}, fmt.Errorf("invalid window size: %s", w)
	}
}

// Duration returns the duration of the window size
// BEWARE: a day is NOT 24 hours
func (w WindowSize) Duration() time.Duration {
	var windowDuration time.Duration
	switch w {
	case WindowSizeMinute:
		windowDuration = time.Minute
	case WindowSizeHour:
		windowDuration = time.Hour
	case WindowSizeDay:
		windowDuration = 24 * time.Hour
	}

	return windowDuration
}

func WindowSizeFromDuration(duration time.Duration) (WindowSize, error) {
	switch duration.Minutes() {
	case time.Minute.Minutes():
		return WindowSizeMinute, nil
	case time.Hour.Minutes():
		return WindowSizeHour, nil
	case 24 * time.Hour.Minutes():
		return WindowSizeDay, nil
	default:
		return "", fmt.Errorf("invalid window size duration: %s", duration)
	}
}

type Meter struct {
	models.ManagedResource `mapstructure:",squash"`

	Key           string `mapstructure:"slug"`
	Aggregation   MeterAggregation
	EventType     string
	EventFrom     *time.Time
	ValueProperty *string
	GroupBy       map[string]string

	// Deprecated, always set to MINUTE
	WindowSize WindowSize
}

func (m *Meter) SupportsWindowSize(w *WindowSize) error {
	// Ensure `from` and `to` aligns with query param window size if any
	if w != nil {
		// Ensure query param window size is not smaller than meter window size
		switch m.WindowSize {
		case WindowSizeHour:
			if w != nil && *w == WindowSizeMinute {
				return fmt.Errorf("cannot query meter with window size %s on window size %s", m.WindowSize, *w)
			}
		case WindowSizeDay:
			if w != nil && (*w == WindowSizeMinute || *w == WindowSizeHour) {
				return fmt.Errorf("cannot query meter with window size %s on window size %s", m.WindowSize, *w)
			}
		}
	}
	return nil
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

	return nil
}

func (m *Meter) Validate() error {
	var errs []error

	if err := m.ManagedResource.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid managed resource: %w", err))
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

	if m.WindowSize != WindowSizeMinute {
		errs = append(errs, errors.New("meter window size is deprecated and always must be set to minute"))
	}

	// Validate aggregation
	if err := validateMeterAggregation(m.ValueProperty, m.Aggregation); err != nil {
		errs = append(errs, fmt.Errorf("invalid meter aggregation: %w", err))
	}

	// Validate group by values
	if err := validateMeterGroupBy(m.ValueProperty, m.GroupBy); err != nil {
		errs = append(errs, fmt.Errorf("invalid meter group by: %w", err))
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
	GroupBy     map[string]*string `json:"groupBy"`
}
