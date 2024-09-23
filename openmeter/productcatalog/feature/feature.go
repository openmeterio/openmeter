package feature

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type FeatureNotFoundError struct {
	ID string
}

func (e *FeatureNotFoundError) Error() string {
	return fmt.Sprintf("feature not found: %s", e.ID)
}

type FeatureInvalidFiltersError struct {
	RequestedFilters    map[string]string
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
	Aggregation       models.MeterAggregation
	ValidAggregations []models.MeterAggregation
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

// MeterGroupByFilters is a map of filters that can be applied to a meter when querying the usage for a feature.
type MeterGroupByFilters map[string]string

func (f MeterGroupByFilters) Validate(meter models.Meter) error {
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

// Feature is a feature or service offered to a customer.
// For example: CPU-Hours, Tokens, API Calls, etc.
type Feature struct {
	Namespace string `json:"-"`
	ID        string `json:"id,omitempty"`

	// Name The name of the feature.
	Name string `json:"name"`
	// Key The unique key of the feature.
	Key string `json:"key"`

	// MeterSlug The meter that the feature is associated with and decreases grants by usage.
	MeterSlug *string `json:"meterSlug,omitempty"`

	// MeterGroupByFilters Optional meter group by filters. Useful if the meter scope is broader than what feature tracks.
	MeterGroupByFilters MeterGroupByFilters `json:"meterGroupByFilters,omitempty"`

	// Metadata Additional metadata.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Read-only fields
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
