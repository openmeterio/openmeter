//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=codegen.yaml ./openapi.yaml
package v3

import (
	"net/http"
)

// FilterString A filter for a string field.
type FilterString struct {
	// Contains The field must contain the provided value.
	Contains *string `json:"contains,omitempty"`

	// Eq The field must match the provided value.
	Eq *string `json:"eq,omitempty"`

	// Neq The field must not match the provided value.
	Neq *string `json:"neq,omitempty"`

	// Ocontains asd
	Ocontains *string `json:"ocontains,omitempty"`

	// Oeq aasd
	Oeq *string `json:"oeq,omitempty"`
}

func (f *FilterString) ParseEq(name string, r *http.Request) {
	if f == nil {
		f = &FilterString{}
	}
	query := r.URL.Query()
	eq := query.Get(name)
	if eq != "" {
		f.Eq = &eq
	}
}
