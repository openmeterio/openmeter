package models

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

type MeterAggregation string

const (
	MeterAggregationSum           MeterAggregation = "SUM"
	MeterAggregationCount         MeterAggregation = "COUNT"
	MeterAggregationCountDistinct MeterAggregation = "COUNT_DISTINCT"
	MeterAggregationMax           MeterAggregation = "MAX"
	MeterAggregationLatest        MeterAggregation = "LATEST_BY_OFFSET"
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

type Meter struct {
	ID            string            `json:"id" yaml:"id"`
	Name          string            `json:"name" yaml:"name"`
	Description   string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels        map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Type          string            `json:"type" yaml:"type"`
	Aggregation   MeterAggregation  `json:"aggregation" yaml:"aggregation"`
	ValueProperty string            `json:"valueProperty,omitempty" yaml:"valueProperty,omitempty"`
	GroupBy       []string          `json:"groupBy,omitempty" yaml:"groupBy,omitempty"`
	// TODO: add filter by
	// FilterBy      []MeterFilter
}

type MeterOptions struct {
	Description string
	Labels      map[string]string
	Unit        string
	GroupBy     []string
}

func NewMeter(id, name, meterType, valueProperty string, aggregation MeterAggregation, options *MeterOptions) (*Meter, error) {
	meter := &Meter{
		ID:            id,
		Name:          name,
		ValueProperty: valueProperty,
		Aggregation:   aggregation,
		Type:          meterType,
	}

	// Apply optional parameters if provided
	if options != nil {
		meter.Description = options.Description
		meter.Labels = options.Labels
		meter.GroupBy = options.GroupBy
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
