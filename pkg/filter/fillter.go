package filter

import (
	"encoding/json"
	"fmt"
)

type Filter struct {
	// Equality
	Eq  *FilterValue `json:"$eq,omitempty"`
	Gt  *FilterValue `json:"$gt,omitempty"`
	Gte *FilterValue `json:"$gte,omitempty"`
	Lt  *FilterValue `json:"$lt,omitempty"`
	Lte *FilterValue `json:"$lte,omitempty"`

	Ne *FilterValue `json:"$ne,omitempty"`

	// String
	Like    *FilterValue `json:"$like,omitempty"`
	NotLike *FilterValue `json:"$notLike,omitempty"`
	Match   *FilterValue `json:"$match,omitempty"`

	// Array
	In  *FilterValue `json:"$in,omitempty"`
	Nin *FilterValue `json:"$nin,omitempty"`

	// Controls
	And *[]Filter `json:"$and,omitempty"`
	Or  *[]Filter `json:"$or,omitempty"`
	Not *Filter   `json:"$not,omitempty"`
}

type FilterValue struct {
	json.RawMessage
}

// AsFilterValueNumber returns filter value as a FilterValueNumber
func (t FilterValue) AsFilterValueNumber() (FilterValueNumber, error) {
	var body FilterValueNumber
	err := json.Unmarshal(t.RawMessage, &body)
	return body, err
}

// AsFilterValueString returns filter value as a FilterValueString
func (t FilterValue) AsFilterValueString() (FilterValueString, error) {
	var body FilterValueString
	err := json.Unmarshal(t.RawMessage, &body)
	return body, err
}

// AsFilterValueArrayString returns filter value as a FilterValueArrayString
func (t FilterValue) AsFilterValueArrayString() (FilterValueArrayString, error) {
	var body FilterValueArrayString
	err := json.Unmarshal(t.RawMessage, &body)
	return body, err
}

// AsFilterValueArrayNumber returns filter value as a FilterValueArrayNumber
func (t FilterValue) AsFilterValueArrayNumber() (FilterValueArrayNumber, error) {
	var body FilterValueArrayNumber
	err := json.Unmarshal(t.RawMessage, &body)
	return body, err
}

// See: https://pkg.go.dev/encoding/json#Unmarshal
// float64, for JSON numbers
// string, for JSON strings
type JSONUnmarshald interface {
	~float64 | ~string
}

// FilterValueArrayNumber defines model for FilterValueArrayNumber.
type FilterValueArrayNumber = []float64

// FilterValueArrayString defines model for FilterValueArrayString.
type FilterValueArrayString = []string

// FilterValueNumber defines model for FilterValueNumber.
type FilterValueNumber = float64

// FilterValueString defines model for FilterValueString.
type FilterValueString = string

func ToFilter(str string) (Filter, error) {
	filter := Filter{}
	err := json.Unmarshal([]byte(str), &filter)
	if err != nil {
		return filter, fmt.Errorf("invalid filter: %s", str)
	}
	return filter, nil
}
