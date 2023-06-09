package models

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

type MeterAggregation string

const (
	MeterAggregationSum    MeterAggregation = "SUM"
	MeterAggregationCount  MeterAggregation = "COUNT"
	MeterAggregationAvg    MeterAggregation = "AVG"
	MeterAggregationMin    MeterAggregation = "MIN"
	MeterAggregationMax    MeterAggregation = "MAX"
	MeterAggregationLatest MeterAggregation = "LATEST_BY_OFFSET"

	// these types can not be used for further aggregations
	MeterAggregationCountDistinct MeterAggregation = "COUNT_DISTINCT"
)

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

type WindowSize string

const (
	WindowSizeMinute WindowSize = "MINUTE"
	WindowSizeHour   WindowSize = "HOUR"
	WindowSizeDay    WindowSize = "DAY"
)

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

type Meter struct {
	ID            string            `json:"id" yaml:"id"`
	Name          string            `json:"name" yaml:"name"`
	Description   string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels        map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Type          string            `json:"type" yaml:"type"`
	Aggregation   MeterAggregation  `json:"aggregation" yaml:"aggregation"`
	ValueProperty string            `json:"valueProperty,omitempty" yaml:"valueProperty,omitempty"`
	GroupBy       []string          `json:"groupBy,omitempty" yaml:"groupBy,omitempty"`
	WindowSize    WindowSize        `json:"windowSize,omitempty" yaml:"windowSize,omitempty"`
	// TODO: add filter by
	// FilterBy      []MeterFilter
}

type MeterOptions struct {
	Description string
	Labels      map[string]string
	GroupBy     []string
	WindowSize  *WindowSize
}

func NewMeter(id, name, meterType, valueProperty string, aggregation MeterAggregation, options *MeterOptions) (*Meter, error) {
	meter := &Meter{
		ID:            id,
		Name:          name,
		ValueProperty: valueProperty,
		Aggregation:   aggregation,
		Type:          meterType,
		WindowSize:    WindowSizeHour,
	}

	// Apply optional parameters if provided
	if options != nil {
		meter.Description = options.Description
		meter.Labels = options.Labels
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
	if m.ID == "" {
		return errors.New("meter id is required")
	}
	if m.Name == "" {
		return errors.New("meter name is required")
	}
	if m.Type == "" {
		return errors.New("meter type is required")
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
		for _, field := range m.GroupBy {
			if !strings.HasPrefix(field, "$") {
				return errors.New("meter group by field must start with $")
			}
		}
	}

	return nil
}

func (m *Meter) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type MeterValue struct {
	Subject     string            `json:"subject"`
	WindowStart time.Time         `json:"windowStart"`
	WindowEnd   time.Time         `json:"windowEnd"`
	Value       float64           `json:"value"`
	GroupBy     map[string]string `json:"groupBy"`
}

func (m *MeterValue) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (m *Meter) AggregateMeterValues(values []*MeterValue, windowSize *WindowSize) ([]*MeterValue, error) {
	// no need to aggregate
	if windowSize == nil || *windowSize == m.WindowSize {
		return values, nil
	}

	// cannot aggregate count distinct
	if m.Aggregation == MeterAggregationCountDistinct {
		return nil, fmt.Errorf("invalid aggregation: expected window size of %s for meter with ID %s, but got %s", m.WindowSize, m.ID, *windowSize)
	}

	// cannot aggregate with a lower resolution
	if m.WindowSize == WindowSizeDay && *windowSize != WindowSizeDay {
		return nil, fmt.Errorf("invalid aggregation: expected window size of %s for meter with ID %s, but got %s", WindowSizeDay, m.ID, *windowSize)
	}
	if m.WindowSize == WindowSizeHour && *windowSize == WindowSizeMinute {
		return nil, fmt.Errorf("invalid aggregation: expected window size of %s or %s for meter with ID %s, but got %s", WindowSizeHour, WindowSizeDay, m.ID, *windowSize)
	}

	if len(values) == 0 {
		return values, nil
	}

	// group by subject, group by and window
	type groupByKey struct {
		Subject     string
		GroupBy     string
		WindowStart time.Time
		WindowEnd   time.Time
	}

	windowDuration := windowSize.Duration()
	groupBy := make(map[groupByKey]*MeterValue)
	avgCount := make(map[groupByKey]int)

	for _, value := range values {
		windowStart := value.WindowStart.UTC().Truncate(windowDuration)
		windowEnd := windowStart.Add(windowDuration)
		groupByKey := groupByKey{
			Subject:     value.Subject,
			GroupBy:     fmt.Sprint(value.GroupBy),
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
		}

		if _, ok := groupBy[groupByKey]; !ok {
			groupBy[groupByKey] = value
			groupBy[groupByKey].WindowStart = windowStart
			groupBy[groupByKey].WindowEnd = windowEnd
			if m.Aggregation == MeterAggregationAvg {
				avgCount[groupByKey] = 1
			}
		} else {
			switch m.Aggregation {
			case MeterAggregationCount:
				groupBy[groupByKey].Value += value.Value
			case MeterAggregationSum:
				groupBy[groupByKey].Value += value.Value
			case MeterAggregationMax:
				groupBy[groupByKey].Value = math.Max(groupBy[groupByKey].Value, value.Value)
			case MeterAggregationMin:
				groupBy[groupByKey].Value = math.Min(groupBy[groupByKey].Value, value.Value)
			case MeterAggregationAvg:
				avgCount[groupByKey]++
				n := float64(avgCount[groupByKey])
				groupBy[groupByKey].Value = (groupBy[groupByKey].Value*(n-1) + value.Value) / n
			case MeterAggregationLatest:
				groupBy[groupByKey].Value = value.Value
			}

			// adjust window start and end
			if groupBy[groupByKey].WindowStart.After(windowStart) {
				groupBy[groupByKey].WindowStart = windowStart
			}
			if groupBy[groupByKey].WindowEnd.Before(windowEnd) {
				groupBy[groupByKey].WindowEnd = windowEnd
			}
		}
	}

	v := make([]*MeterValue, 0, len(groupBy))
	for _, value := range groupBy {
		v = append(v, value)
	}

	// sort by subject and window start
	sort.Slice(v, func(i, j int) bool {
		if v[i].Subject == v[j].Subject {
			return v[i].WindowStart.Before(v[j].WindowStart)
		}

		return v[i].Subject < v[j].Subject
	})

	return v, nil
}
