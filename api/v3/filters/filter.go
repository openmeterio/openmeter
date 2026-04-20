package filters

import "time"

// FilterBoolean represents a filter operation on a boolean field.
type FilterBoolean struct {
	// Eq requires the field to match the provided value exactly.
	Eq *bool `json:"eq,omitempty"`
}

// FilterNumeric represents a filter operation on a numeric field.
type FilterNumeric struct {
	// Eq requires the field to match the provided value exactly.
	Eq *float64 `json:"eq,omitempty"`

	// Neq requires the field to not match the provided value.
	Neq *float64 `json:"neq,omitempty"`

	// Oeq requires the field to match any of the provided comma-separated values.
	Oeq []float64 `json:"oeq,omitempty"`

	// Gt requires the field to be greater than the provided value.
	Gt *float64 `json:"gt,omitempty"`

	// Gte requires the field to be greater than or equal to the provided value.
	Gte *float64 `json:"gte,omitempty"`

	// Lt requires the field to be less than the provided value.
	Lt *float64 `json:"lt,omitempty"`

	// Lte requires the field to be less than or equal to the provided value.
	Lte *float64 `json:"lte,omitempty"`
}

// FilterDateTime represents a filter operation on a datetime field.
type FilterDateTime struct {
	// Eq requires the field to match the provided value exactly.
	Eq *time.Time `json:"eq,omitempty"`

	// Gt requires the field to be greater than the provided value.
	Gt *time.Time `json:"gt,omitempty"`

	// Gte requires the field to be greater than or equal to the provided value.
	Gte *time.Time `json:"gte,omitempty"`

	// Lt requires the field to be less than the provided value.
	Lt *time.Time `json:"lt,omitempty"`

	// Lte requires the field to be less than or equal to the provided value.
	Lte *time.Time `json:"lte,omitempty"`
}

// FilterString represents a filter operation on a string field.
type FilterString struct {
	// Eq requires the field to match the provided value exactly (case-sensitive).
	Eq *string `json:"eq,omitempty"`

	// Neq requires the field to not match the provided value (case-sensitive).
	Neq *string `json:"neq,omitempty"`

	// Gt requires the field to be greater than the provided value.
	Gt *string `json:"gt,omitempty"`

	// Gte requires the field to be greater than or equal to the provided value.
	Gte *string `json:"gte,omitempty"`

	// Lt requires the field to be less than the provided value.
	Lt *string `json:"lt,omitempty"`

	// Lte requires the field to be less than or equal to the provided value.
	Lte *string `json:"lte,omitempty"`

	// Contains requires the field to contain the provided value (case-insensitive).
	Contains *string `json:"contains,omitempty"`

	// Oeq requires the field to match any of the provided comma-separated values (case-sensitive).
	Oeq []string `json:"oeq,omitempty"`

	// Ocontains requires the field to contain any of the provided comma-separated values (case-insensitive).
	Ocontains []string `json:"ocontains,omitempty"`

	// Exists requires the field to be present (true) or absent (false).
	Exists *bool `json:"$exists,omitempty"`
}

// FilterULID represents a filter operation on a string field that satisfies ULID format.
type FilterULID struct {
	// Eq requires the field to match the provided value exactly (case-sensitive).
	Eq *string `json:"eq,omitempty"`

	// Neq requires the field to not match the provided value (case-sensitive).
	Neq *string `json:"neq,omitempty"`

	// Gt requires the field to be greater than the provided value.
	Gt *string `json:"gt,omitempty"`

	// Gte requires the field to be greater than or equal to the provided value.
	Gte *string `json:"gte,omitempty"`

	// Lt requires the field to be less than the provided value.
	Lt *string `json:"lt,omitempty"`

	// Lte requires the field to be less than or equal to the provided value.
	Lte *string `json:"lte,omitempty"`

	// Contains requires the field to contain the provided value (case-insensitive).
	Contains *string `json:"contains,omitempty"`

	// Oeq requires the field to match any of the provided comma-separated values (case-sensitive).
	Oeq []string `json:"oeq,omitempty"`

	// Ocontains requires the field to contain any of the provided comma-separated values (case-insensitive).
	Ocontains []string `json:"ocontains,omitempty"`

	// Exists requires the field to be present (true) or absent (false).
	Exists *bool `json:"$exists,omitempty"`
}

// FilterLabel represents a filter operation on a label key.
type FilterLabel struct {
	// Eq requires the field to match the provided value exactly.
	Eq *string `json:"eq,omitempty"`

	// Contains requires the field to contain the provided value (case-insensitive).
	Contains *string `json:"contains,omitempty"`

	// Ocontains requires the field to contain any of the provided comma-separated values (case-insensitive).
	Ocontains []string `json:"ocontains,omitempty"`

	// Oeq requires the field to match any of the provided comma-separated values (case-sensitive).
	Oeq []string `json:"oeq,omitempty"`

	// Neq requires the field to not match the provided value.
	Neq *string `json:"neq,omitempty"`
}

// FilterLabels is a map of label keys to filter operations.
type FilterLabels = map[string]FilterLabel

// FilterStringExact represents a filter operation on a string field that only supports exact matching.
type FilterStringExact struct {
	// Eq requires the field to match the provided value exactly (case-sensitive).
	Eq *string `json:"eq,omitempty"`

	// Neq requires the field to not match the provided value exactly (case-sensitive).
	Neq *string `json:"neq,omitempty"`

	// Oeq requires the field to match any of the provided comma-separated values (case-sensitive).
	Oeq []string `json:"oeq,omitempty"`
}
