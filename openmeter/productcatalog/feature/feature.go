package feature

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/filter"
)

type FeatureNotFoundError struct {
	ID string
}

func (e *FeatureNotFoundError) Error() string {
	return fmt.Sprintf("feature not found: %s", e.ID)
}

type FeatureInvalidFiltersError struct {
	RequestedFilters    MeterGroupByFilters
	MeterGroupByColumns []string
}

func (e *FeatureInvalidFiltersError) Error() string {
	return fmt.Sprintf("invalid filters for feature: %v, available columns: %v", e.RequestedFilters, e.MeterGroupByColumns)
}

type FeatureWithNameAlreadyExistsError struct {
	Name string
	ID   string
}

func (e *FeatureWithNameAlreadyExistsError) Error() string {
	// Is it an issue that we leak ID on another Feature here?
	// Shouldn't be an isue as it's namespaced.
	return fmt.Sprintf("feature %s with key %s already exists", e.ID, e.Name)
}

type FeatureInvalidMeterAggregationError struct {
	MeterSlug         string
	Aggregation       meter.MeterAggregation
	ValidAggregations []meter.MeterAggregation
}

func (e *FeatureInvalidMeterAggregationError) Error() string {
	validAggregations := ""
	for i, validAggregation := range e.ValidAggregations {
		if i > 0 {
			validAggregations += ", "
		}
		validAggregations += string(validAggregation)
	}
	return fmt.Sprintf("meter %s's aggregation is %s but features can only be created for %s", e.MeterSlug, e.Aggregation, validAggregations)
}

type ForbiddenError struct {
	Msg string
	ID  string
}

func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("forbidden for feature %s: %s", e.ID, e.Msg)
}

// MeterGroupByFilters is a map of filters that can be applied to a meter when querying the usage for a feature.
type MeterGroupByFilters map[string]filter.FilterString

func (f MeterGroupByFilters) Validate(meter meter.Meter) error {
	for filterProp := range f {
		if _, ok := meter.GroupBy[filterProp]; !ok {
			meterGroupByColumns := make([]string, 0, len(meter.GroupBy))
			for k := range meter.GroupBy {
				meterGroupByColumns = append(meterGroupByColumns, k)
			}
			return &FeatureInvalidFiltersError{
				RequestedFilters:    f,
				MeterGroupByColumns: meterGroupByColumns,
			}
		}
	}

	return nil
}

// ConvertMapStringToMeterGroupByFilters converts a map[string]string legacy format to MeterGroupByFilters
func ConvertMapStringToMeterGroupByFilters(m map[string]string) MeterGroupByFilters {
	if m == nil {
		return MeterGroupByFilters{}
	}

	result := make(MeterGroupByFilters, len(m))
	for k, v := range m {
		result[k] = filter.FilterString{Eq: &v}
	}

	return result
}

// ConvertMeterGroupByFiltersToMapString converts a MeterGroupByFilters to a legacy map[string]string format
// if all filters are equality filters, otherwise returns nil.
func ConvertMeterGroupByFiltersToMapString(f MeterGroupByFilters) map[string]string {
	if f == nil {
		return nil
	}

	result := make(map[string]string, len(f))
	for k, v := range f {
		if v.Eq == nil {
			return nil
		}
		result[k] = *v.Eq
	}

	return result
}

// Feature is a feature or service offered to a customer.
// For example: CPU-Hours, Tokens, API Calls, etc.
type Feature struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id,omitempty"`

	// Name The name of the feature.
	Name string `json:"name"`
	// Key The unique key of the feature.
	Key string `json:"key"`

	// MeterSlug The meter that the feature is associated with and decreases grants by usage.
	MeterSlug *string `json:"meterSlug,omitempty"`

	// MeterGroupByFilters Optional meter group by filters. Useful if the meter scope is broader than what feature tracks.
	MeterGroupByFilters MeterGroupByFilters `json:"meterGroupByFilters,omitempty"`

	// UnitCost is an optional per-unit cost: either a fixed manual amount or dynamic LLM cost lookup.
	UnitCost *UnitCost `json:"unitCost,omitempty"`

	// Metadata Additional metadata.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Read-only fields
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Validate validates the feature.
func (f *Feature) Validate() error {
	var errs []error

	if f.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if f.ID == "" {
		errs = append(errs, fmt.Errorf("id is required"))
	}

	if f.Name == "" {
		errs = append(errs, fmt.Errorf("name is required"))
	}

	if f.Key == "" {
		errs = append(errs, fmt.Errorf("key is required"))
	}

	if f.MeterSlug != nil {
		if *f.MeterSlug == "" {
			errs = append(errs, fmt.Errorf("meter slug cannot be empty"))
		}
	}

	if f.CreatedAt.IsZero() {
		errs = append(errs, fmt.Errorf("created at is required"))
	}

	if f.UpdatedAt.IsZero() {
		errs = append(errs, fmt.Errorf("updated at is required"))
	}

	return errors.Join(errs...)
}
