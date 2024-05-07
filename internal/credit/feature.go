package credit

import (
	"fmt"
	"net/http"

	"github.com/oklog/ulid/v2"
)

type FeatureNotFoundError struct {
	ID ulid.ULID
}

func (e *FeatureNotFoundError) Error() string {
	return fmt.Sprintf("feature not found: %s", e.ID)
}

// Feature is a feature or service offered to a customer.
// For example: CPU-Hours, Tokens, API Calls, etc.
type Feature struct {
	Namespace string     `json:"-"`
	ID        *ulid.ULID `json:"id,omitempty"`

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
