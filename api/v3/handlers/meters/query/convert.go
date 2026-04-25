package query

import (
	"fmt"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/meter"
)

var iso8601ToWindowSize = map[string]meter.WindowSize{
	"PT1M": meter.WindowSizeMinute,
	"PT1H": meter.WindowSizeHour,
	"P1D":  meter.WindowSizeDay,
	"P1M":  meter.WindowSizeMonth,
}

var windowSizeToISO8601 = map[meter.WindowSize]string{
	meter.WindowSizeMinute: "PT1M",
	meter.WindowSizeHour:   "PT1H",
	meter.WindowSizeDay:    "P1D",
	meter.WindowSizeMonth:  "P1M",
}

func ConvertISO8601DurationToWindowSize(duration string) (meter.WindowSize, error) {
	ws, ok := iso8601ToWindowSize[duration]
	if !ok {
		return "", NewInvalidWindowSizeError(duration)
	}
	return ws, nil
}

func ConvertWindowSizeToISO8601Duration(ws meter.WindowSize) (string, error) {
	if d, ok := windowSizeToISO8601[ws]; ok {
		return d, nil
	}
	return "", fmt.Errorf("unknown WindowSize: %q", ws)
}

// ExtractStringsFromQueryFilter extracts a flat list of string values from a QueryFilterString.
// Only the eq and in operators are supported; an error is returned if any other operator is set.
func ExtractStringsFromQueryFilter(f *api.QueryFilterString, fieldPath ...string) ([]string, error) {
	if f == nil {
		return nil, nil
	}

	if f.Neq != nil || f.Nin != nil ||
		f.Contains != nil || f.Ncontains != nil ||
		f.And != nil || f.Or != nil {
		return nil, NewUnsupportedFilterOperatorError(fieldPath...)
	}
	if f.Eq != nil && f.In != nil {
		return nil, NewUnsupportedFilterOperatorError(fieldPath...)
	}

	var result []string
	if f.Eq != nil {
		result = append(result, *f.Eq)
	}
	if f.In != nil {
		result = append(result, *f.In...)
	}
	return result, nil
}

// ExtractStringsFromQueryFilterMapItem extracts a flat list of string values from a QueryFilterStringMapItem.
// Only the eq and in operators are supported; an error is returned if any other operator is set.
func ExtractStringsFromQueryFilterMapItem(f *api.QueryFilterStringMapItem, fieldPath ...string) ([]string, error) {
	if f == nil {
		return nil, nil
	}

	if f.Neq != nil || f.Nin != nil ||
		f.Contains != nil || f.Ncontains != nil ||
		f.And != nil || f.Or != nil || f.Exists != nil {
		return nil, NewUnsupportedFilterOperatorError(fieldPath...)
	}
	if f.Eq != nil && f.In != nil {
		return nil, NewUnsupportedFilterOperatorError(fieldPath...)
	}

	var result []string
	if f.Eq != nil {
		result = append(result, *f.Eq)
	}
	if f.In != nil {
		result = append(result, *f.In...)
	}
	return result, nil
}
