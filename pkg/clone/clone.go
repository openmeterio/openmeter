// Package clone has small helpers for deep-copying values that a plain struct copy would share.
package clone

// CloneEnumPtr returns a pointer to a copy of v, so the result doesn't alias the input.
// A nil input stays nil.
func CloneEnumPtr[T ~string](v *T) *T {
	if v == nil {
		return nil
	}

	out := *v

	return &out
}
