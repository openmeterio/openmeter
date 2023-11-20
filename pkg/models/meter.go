package models

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

type MeterAggregation string

const (
	MeterAggregationSum   MeterAggregation = "SUM"
	MeterAggregationCount MeterAggregation = "COUNT"
	MeterAggregationAvg   MeterAggregation = "AVG"
	MeterAggregationMin   MeterAggregation = "MIN"
	MeterAggregationMax   MeterAggregation = "MAX"
)

// Values provides list valid values for Enum
func (MeterAggregation) Values() (kinds []string) {
	for _, s := range []MeterAggregation{
		MeterAggregationSum,
		MeterAggregationCount,
		MeterAggregationAvg,
		MeterAggregationMin,
		MeterAggregationMax,
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

type MeterFilterOperator string

const (
	// MeterFilterOperatorIn        MeterFilterOperator = "IN"
	// MeterFilterOperatorNotIn     MeterFilterOperator = "NOT IN"
	MeterFilterOperatorEquals    MeterFilterOperator = "EQ"
	MeterFilterOperatorNot       MeterFilterOperator = "NEQ"
	MeterFilterLowerThan         MeterFilterOperator = "LT"
	MeterFilterLowerThanOrEq     MeterFilterOperator = "LTE"
	MeterFilterGreaterThan       MeterFilterOperator = "GT"
	MeterFilterGreaterThanOrEq   MeterFilterOperator = "GTE"
	MeterFilterOperatorIsNull    MeterFilterOperator = "IS NULL"
	MeterFilterOperatorIsNotNull MeterFilterOperator = "IS NOT NULL"
)

type MeterFilter struct {
	Property string              `json:"property" yaml:"property"`
	Operator MeterFilterOperator `json:"operator" yaml:"operator"`
	Value    string              `json:"value" yaml:"value"`
}

// Values provides list valid values for Enum
func (MeterFilterOperator) Values() (kinds []string) {
	for _, s := range []MeterFilterOperator{
		MeterFilterOperatorEquals,
		MeterFilterOperatorNot,
		MeterFilterLowerThan,
		MeterFilterLowerThanOrEq,
		MeterFilterGreaterThan,
		MeterFilterGreaterThanOrEq,
		MeterFilterOperatorIsNull,
		MeterFilterOperatorIsNotNull,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

type WindowSize string

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

// Duration returns the duration of the window size
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
	// We don't accept namespace via config, it's set by the `namespace.default`.`
	Namespace     string            `json:"-" yaml:"-"`
	ID            string            `json:"id,omitempty" yaml:"id,omitempty"`
	Slug          string            `json:"slug" yaml:"slug"`
	Description   string            `json:"description,omitempty" yaml:"description,omitempty"`
	Aggregation   MeterAggregation  `json:"aggregation" yaml:"aggregation"`
	EventType     string            `json:"eventType" yaml:"eventType"`
	ValueProperty string            `json:"valueProperty,omitempty" yaml:"valueProperty,omitempty"`
	GroupBy       map[string]string `json:"groupBy,omitempty" yaml:"groupBy,omitempty"`
	WindowSize    WindowSize        `json:"windowSize,omitempty" yaml:"windowSize,omitempty"`
	// TODO: add filter by
	// FilterBy      []MeterFilter
}

type MeterOptions struct {
	ID          string
	Description string
	GroupBy     map[string]string
	WindowSize  *WindowSize
}

func NewMeter(
	slug string,
	aggregatation MeterAggregation,
	eventType string,
	valueProperty string,
	options *MeterOptions,
) (*Meter, error) {
	meter := &Meter{
		Slug:          slug,
		Aggregation:   aggregatation,
		EventType:     eventType,
		ValueProperty: valueProperty,
	}

	// Apply optional parameters if provided
	if options != nil {
		meter.ID = options.ID
		meter.Description = options.Description
		meter.GroupBy = options.GroupBy
		if options.WindowSize != nil {
			meter.WindowSize = *options.WindowSize
		}
	}

	err := meter.Validate()
	if err != nil {
		return nil, err
	}

	return meter, nil
}

func (m *Meter) Validate() error {
	if m.ID != "" {
		_, err := ulid.ParseStrict(m.ID)
		if err != nil {
			return errors.New("meter id is not a valid ULID")
		}
	}
	if m.Slug == "" {
		return errors.New("meter slug is required")
	}
	// FIXME: prefix materialized views to allow `events` as a meter slug
	// `events` is a restricted name as it conflicts with the events table
	if m.Slug == "events" {
		return errors.New("meter slug cannot be `events`")
	}
	if len(m.Slug) > 63 {
		return errors.New("meter slug must be less than 64 characters")
	}
	if m.EventType == "" {
		return errors.New("meter event type is required")
	}
	if m.Aggregation == "" {
		return errors.New("meter aggregation is required")
	}

	// ValueProperty is required when the aggregation is not count
	if m.Aggregation != MeterAggregationCount {
		if m.ValueProperty == "" {
			return errors.New("meter value property is required when the aggregation is not count")
		}

		if !strings.HasPrefix(m.ValueProperty, "$") {
			return errors.New("meter value property must start with $")
		}
	}

	if len(m.GroupBy) != 0 {
		for key, value := range m.GroupBy {
			if !strings.HasPrefix(value, "$") {
				return fmt.Errorf("meter group by value must start with $ for key %s", key)
			}
		}
	}

	return nil
}

func (m *Meter) Render(w http.ResponseWriter, r *http.Request) error {
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

// Render implements the chi renderer interface.
func (m *MeterQueryRow) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
