package credit

import (
	"fmt"
	"net/http"
)

type FeatureNotFoundError struct {
	ID string
}

func (e *FeatureNotFoundError) Error() string {
	return fmt.Sprintf("feature not found: %s", e.ID)
}

// Feature is a feature or service offered to a customer.
// For example: CPU-Hours, Tokens, API Calls, etc.
type Feature struct {
	Namespace string  `json:"namespace"`
	ID        *string `json:"id"`

	// Name The name of the feature.
	Name string `json:"name"`

	// MeterSlug The meter that the feature is associated with and decreases grants by usage.
	MeterSlug string `json:"meterSlug,omitempty"`

	// MeterGroupByFilters Optional meter group by filters. Useful if the meter scope is broader than what feature tracks.
	MeterGroupByFilters *map[string]string `json:"meterGroupByFilters,omitempty"`

	Archived bool `json:"archived"`
}

// Render implements the chi renderer interface.
func (c Feature) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
