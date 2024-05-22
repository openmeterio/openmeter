package credit

import (
	"fmt"
	"time"
)

// FeatureID is the unique identifier for a feature.
type FeatureID string

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

// Feature is a feature or service offered to a customer.
// For example: CPU-Hours, Tokens, API Calls, etc.
type Feature struct {
	Namespace string     `json:"-"`
	ID        *FeatureID `json:"id,omitempty"`

	// Name The name of the feature.
	Name string `json:"name"`

	// MeterSlug The meter that the feature is associated with and decreases grants by usage.
	MeterSlug string `json:"meterSlug,omitempty"`

	// MeterGroupByFilters Optional meter group by filters. Useful if the meter scope is broader than what feature tracks.
	MeterGroupByFilters *map[string]string `json:"meterGroupByFilters,omitempty"`

	// Read-only fields
	Archived *bool `json:"archived,omitempty"`

	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}
