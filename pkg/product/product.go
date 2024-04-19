package product

import (
	"fmt"
	"net/http"
)

type ProductNotFoundError struct {
	ID string
}

func (e *ProductNotFoundError) Error() string {
	return fmt.Sprintf("product not found: %s", e.ID)
}

// Product is a feature or service offered to a customer.
// For example: CPU-Hours, Tokens, API Calls, etc.
type Product struct {
	Namespace string  `json:"namespace"`
	ID        *string `json:"id"`

	// Name The name of the product.
	Name string `json:"name"`

	// MeterSlug The meter that the product is associated with and decreases grants by usage.
	MeterSlug string `json:"meterSlug,omitempty"`

	// MeterGroupByFilters Optional meter group by filters. Useful if the meter scope is broader than what product tracks.
	MeterGroupByFilters *map[string]string `json:"meterGroupByFilters,omitempty"`

	Archived bool `json:"archived"`
}

// Render implements the chi renderer interface.
func (c Product) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
