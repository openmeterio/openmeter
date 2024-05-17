package credit

import (
	"fmt"
	"net/http"
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
}

// Render implements the chi renderer interface.
func (c Feature) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
