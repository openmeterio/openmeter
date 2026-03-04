package filters

import "errors"

// StringFilter represents a filter operation on a string field.
// Exactly one of Eq, Neq, or Contains should be set.
type StringFilter struct {
	// Eq requires the field to match the provided value exactly (case-insensitive).
	Eq *string `json:"eq,omitempty"`

	// Neq requires the field to not match the provided value (case-insensitive).
	Neq *string `json:"neq,omitempty"`

	// Contains requires the field to contain the provided value (case-insensitive).
	Contains *string `json:"contains,omitempty"`
}

// IsEmpty returns true if no filter operator is set.
func (f StringFilter) IsEmpty() bool {
	return f.Eq == nil && f.Neq == nil && f.Contains == nil
}

// Validate validates the filter.
func (f StringFilter) Validate() error {
	if f.IsEmpty() {
		return nil
	}

	// Check for mutually exclusive filters
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
