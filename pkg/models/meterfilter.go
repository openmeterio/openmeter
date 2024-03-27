package models

import (
	"fmt"
	"github.com/huandu/go-sqlbuilder"
)

type MeterFilterOperator string

const (
	MeterFilterOperatorEquals MeterFilterOperator = "EQ"
	// TODO support more filters and value types
	// MeterFilterOperatorIn        MeterFilterOperator = "IN"
	// MeterFilterOperatorNotIn     MeterFilterOperator = "NOT IN"
	// MeterFilterOperatorNot       MeterFilterOperator = "NEQ"
	// MeterFilterLowerThan         MeterFilterOperator = "LT"
	// MeterFilterLowerThanOrEq     MeterFilterOperator = "LTE"
	// MeterFilterGreaterThan       MeterFilterOperator = "GT"
	// MeterFilterGreaterThanOrEq   MeterFilterOperator = "GTE"
	// MeterFilterOperatorIsNull    MeterFilterOperator = "IS NULL"
	// MeterFilterOperatorIsNotNull MeterFilterOperator = "IS NOT NULL"
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
		//MeterFilterOperatorNot,
		//MeterFilterLowerThan,
		//MeterFilterLowerThanOrEq,
		//MeterFilterGreaterThan,
		//MeterFilterGreaterThanOrEq,
		//MeterFilterOperatorIsNull,
		//MeterFilterOperatorIsNotNull,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

func (f MeterFilter) ToSql() (string, error) {
	switch f.Operator {
	case MeterFilterOperatorEquals:
		return fmt.Sprintf("JSON_VALUE(data, '%s') = '%s'", sqlbuilder.Escape(f.Property), sqlbuilder.Escape(f.Value)), nil
	default:
		return "", fmt.Errorf("filter not implemented yet: %s", f.Operator)
	}
}
