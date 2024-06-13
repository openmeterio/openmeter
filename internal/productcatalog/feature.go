package productcatalog

// TODO: Move to product catalog once released

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// FeatureID is the unique identifier for a feature.
type FeatureID string

func (id FeatureID) String() string {
	return string(id)
}

type NamespacedFeatureID struct {
	Namespace string
	ID        FeatureID
}

func NewNamespacedFeatureID(namespace string, id FeatureID) NamespacedFeatureID {
	return NamespacedFeatureID{
		Namespace: namespace,
		ID:        id,
	}
}

type FeatureNotFoundError struct {
	ID FeatureID
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
	ID   FeatureID
}

func (e *FeatureWithNameAlreadyExistsError) Error() string {
	// Is it an issue that we leak ID on another Feature here?
	// Shouldn't be an isue as it's namespaced.
	return fmt.Sprintf("feature %s with name %s already exists", e.ID, e.Name)
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

// Feature is a feature or service offered to a customer.
// For example: CPU-Hours, Tokens, API Calls, etc.
type Feature struct {
	Namespace string    `json:"-"`
	ID        FeatureID `json:"id,omitempty"`

	// Name The name of the feature.
	Name string `json:"name"`

	// MeterSlug The meter that the feature is associated with and decreases grants by usage.
	MeterSlug string `json:"meterSlug,omitempty"`

	// MeterGroupByFilters Optional meter group by filters. Useful if the meter scope is broader than what feature tracks.
	MeterGroupByFilters *map[string]string `json:"meterGroupByFilters,omitempty"`

	// Read-only fields
	Archived bool `json:"archived,omitempty"`

	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}
