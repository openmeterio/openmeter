package filters

import "errors"

// StringFilter represents a filter operation on a string field.
type StringFilter struct {
	// Eq requires the field to match the provided value exactly (case-insensitive).
	Eq *string `json:"eq,omitempty"`

	// Neq requires the field to not match the provided value (case-insensitive).
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

	// Oeq requires the field to match any of the provided comma-separated values (case-insensitive).
	Oeq *string `json:"oeq,omitempty"`

	// Ocontains requires the field to contain any of the provided comma-separated values (case-insensitive).
	Ocontains *string `json:"ocontains,omitempty"`

	// Exists requires the field to be present (true) or absent (false).
	Exists *bool `json:"exists,omitempty"`
}

// IsEmpty returns true if no filter operator is set.
func (f StringFilter) IsEmpty() bool {
	return f.Eq == nil &&
		f.Neq == nil &&
		f.Gt == nil &&
		f.Gte == nil &&
		f.Lt == nil &&
		f.Lte == nil &&
		f.Contains == nil &&
		f.Oeq == nil &&
		f.Ocontains == nil &&
		f.Exists == nil
}

// Validate validates the filter.
func (f StringFilter) Validate() error {
	if f.IsEmpty() {
		return nil
	}

	if f.Eq != nil && f.Neq != nil {
		return errors.New("eq and neq cannot be set at the same time")
	}

	if f.Contains != nil && f.Eq != nil {
		return errors.New("contains and eq cannot be set at the same time")
	}

	if f.Contains != nil && f.Neq != nil {
		return errors.New("contains and neq cannot be set at the same time")
	}

	return nil
}
