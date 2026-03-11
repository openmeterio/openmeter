//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=codegen.yaml ./openapi.yaml
package v3

// FilterSingleString A filter for a single string field.
// TODO: This is a temporary solution to support the filter API.
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
