package query

import (
	"fmt"
	"sort"

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

// ValidateQueryFilterString checks that a QueryFilterString contains no unknown operators.
func ValidateQueryFilterString(f *api.QueryFilterString, fieldPath ...string) error {
	if f == nil {
		return nil
	}

	if len(f.AdditionalProperties) > 0 {
		return NewUnknownFilterOperatorError(firstKey(f.AdditionalProperties), fieldPath...)
	}

	// Recursively validate nested filters
	if f.And != nil {
		for i := range *f.And {
			if err := ValidateQueryFilterString(&(*f.And)[i], fieldPath...); err != nil {
				return err
			}
		}
	}

	if f.Or != nil {
		for i := range *f.Or {
			if err := ValidateQueryFilterString(&(*f.Or)[i], fieldPath...); err != nil {
				return err
			}
		}
	}

	return nil
}

// ValidateQueryFilterStringMapItem checks that a QueryFilterStringMapItem contains no unknown operators.
func ValidateQueryFilterStringMapItem(f api.QueryFilterStringMapItem, fieldPath ...string) error {
	if len(f.AdditionalProperties) > 0 {
		return NewUnknownFilterOperatorError(firstKey(f.AdditionalProperties), fieldPath...)
	}

	// Recursively validate nested filters
	if f.And != nil {
		for i := range *f.And {
			if err := ValidateQueryFilterString(&(*f.And)[i], fieldPath...); err != nil {
				return err
			}
		}
	}

	if f.Or != nil {
		for i := range *f.Or {
			if err := ValidateQueryFilterString(&(*f.Or)[i], fieldPath...); err != nil {
				return err
			}
		}
	}

	return nil
}

// firstKey returns the first key from a map in sorted order for deterministic error messages.
func firstKey[V any](m map[string]V) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys[0]
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
