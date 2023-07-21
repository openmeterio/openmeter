package models

import (
	"encoding/json"
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
	MeterAggregationLatest MeterAggregation = "LATEST"
)

// Values provides list valid values for Enum
func (MeterAggregation) Values() (kinds []string) {
	for _, s := range []MeterAggregation{
		MeterAggregationSum,
		MeterAggregationCount,
		MeterAggregationAvg,
		MeterAggregationMin,
		MeterAggregationMax,
		MeterAggregationLatest,
	} {
		kinds = append(kinds, string(s))
	}
	return
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

type MeterApi struct {
	Slug          string           `json:"slug" yaml:"slug"`
	Description   string           `json:"description,omitempty" yaml:"description,omitempty"`
	Aggregation   MeterAggregation `json:"aggregation" yaml:"aggregation"`
	EventType     string           `json:"eventType" yaml:"eventType"`
	ValueProperty string           `json:"valueProperty,omitempty" yaml:"valueProperty,omitempty"`
	GroupBy       interface{}      `json:"groupBy,omitempty" yaml:"groupBy,omitempty"`
	WindowSize    WindowSize       `json:"windowSize,omitempty" yaml:"windowSize,omitempty"`
}

func (ma *MeterApi) Validate() error {
	m := &Meter{}
	err := ma.CopyToMeter(m)
	if err != nil {
		return err
	}

	return m.Validate()
}

func (ma *MeterApi) ToMeter() (*Meter, error) {
	m := &Meter{}
	err := ma.CopyToMeter(m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (ma *MeterApi) CopyToMeter(m *Meter) error {
	m.Slug = ma.Slug
	m.Description = ma.Description
	m.Aggregation = ma.Aggregation
	m.EventType = ma.EventType
	m.ValueProperty = ma.ValueProperty
	m.GroupBy = map[string]string{}
	m.WindowSize = ma.WindowSize

	// Parse group by
	list, ok := ma.GroupBy.([]string)
	// Converting list of property keys to map of JSONPaths
	if ok {
		for _, item := range list {
			m.GroupBy[item] = fmt.Sprintf("$.%s", item)
		}
		return nil
	}

	// Group by is map
	g, ok := ma.GroupBy.(map[string]interface{})
	if ok {
		for key, val := range g {
			s, ok := val.(string)
			if ok {
				m.GroupBy[key] = s
			} else {
				return fmt.Errorf("group by map value must be a JSON Path string: %s", key)
			}

		}
		return nil
	}

	// Group by is not defined
	if g == nil {
		return nil
	}
	return errors.New("group by must be a []strings or map(string, JSONPath)")
}

type Meter struct {
	Slug          string            `json:"slug" yaml:"slug"`
	Description   string            `json:"description,omitempty" yaml:"description,omitempty"`
	Aggregation   MeterAggregation  `json:"aggregation" yaml:"aggregation"`
	EventType     string            `json:"eventType" yaml:"eventType"`
	ValueProperty string            `json:"valueProperty,omitempty" yaml:"valueProperty,omitempty"`
	GroupBy       map[string]string `json:"groupBy,omitempty" yaml:"groupBy,omitempty"`
	WindowSize    WindowSize        `json:"windowSize,omitempty" yaml:"windowSize,omitempty"`
}

type MeterOptions struct {
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

func (m *Meter) UnmarshalJSON(data []byte) error {
	var ma MeterApi

	if err := json.Unmarshal(data, &ma); err != nil {
		return err
	}

	return ma.CopyToMeter(m)
}

func (m *Meter) Validate() error {
	if m.Slug == "" {
		return errors.New("meter id is required")
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
		for _, value := range m.GroupBy {
			if !strings.HasPrefix(value, "$") {
				return errors.New("meter group by value must be a JSONPath and start with $")
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
	if windowSize != nil {
		// no need to aggregate
		if *windowSize == m.WindowSize {
			return values, nil
		}

		// cannot aggregate with a lower resolution
		if m.WindowSize == WindowSizeDay && *windowSize != WindowSizeDay {
			return nil, fmt.Errorf("invalid aggregation: expected window size of %s for meter with slug %s, but got %s", WindowSizeDay, m.Slug, *windowSize)
		}
		if m.WindowSize == WindowSizeHour && *windowSize == WindowSizeMinute {
			return nil, fmt.Errorf("invalid aggregation: expected window size of %s or %s for meter with slug %s, but got %s", WindowSizeHour, WindowSizeDay, m.Slug, *windowSize)
		}
	}

	if len(values) == 0 {
		return values, nil
	}

	// key by subject, group by and window
	type key struct {
		Subject     string
		GroupBy     string
		WindowStart time.Time
		WindowEnd   time.Time
	}

	groupBy := make(map[key]*MeterValue)
	avgCount := make(map[key]int)
	fullWindowStart := make(map[key]time.Time)
	fullWindowEnd := make(map[key]time.Time)

	for _, value := range values {
		key := key{
			Subject: value.Subject,
			GroupBy: fmt.Sprint(value.GroupBy),
		}

		if windowSize != nil {
			windowDuration := windowSize.Duration()
			key.WindowStart = value.WindowStart.UTC().Truncate(windowDuration)
			key.WindowEnd = key.WindowStart.Add(windowDuration)
		}

		// set full window
		if fullWindowStart[key].IsZero() || value.WindowStart.Before(fullWindowStart[key]) {
			fullWindowStart[key] = value.WindowStart
		}
		if fullWindowEnd[key].IsZero() || value.WindowEnd.After(fullWindowEnd[key]) {
			fullWindowEnd[key] = value.WindowEnd
		}

		if _, ok := groupBy[key]; !ok {
			groupBy[key] = value
			groupBy[key].WindowStart = key.WindowStart
			groupBy[key].WindowEnd = key.WindowEnd
			if m.Aggregation == MeterAggregationAvg {
				avgCount[key] = 1
			}
		} else {
			// update value
			switch m.Aggregation {
			case MeterAggregationCount:
				groupBy[key].Value += value.Value
			case MeterAggregationSum:
				groupBy[key].Value += value.Value
			case MeterAggregationMax:
				groupBy[key].Value = math.Max(groupBy[key].Value, value.Value)
			case MeterAggregationMin:
				groupBy[key].Value = math.Min(groupBy[key].Value, value.Value)
			case MeterAggregationAvg:
				avgCount[key]++
				n := float64(avgCount[key])
				groupBy[key].Value = (groupBy[key].Value*(n-1) + value.Value) / n
			case MeterAggregationLatest:
				groupBy[key].Value = value.Value
			}
		}
	}

	v := make([]*MeterValue, 0, len(groupBy))
	for _, value := range groupBy {
		// set full window if window size is not set
		if windowSize == nil {
			key := key{
				Subject: value.Subject,
				GroupBy: fmt.Sprint(value.GroupBy),
			}

			value.WindowStart = fullWindowStart[key]
			value.WindowEnd = fullWindowEnd[key]
		}

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
