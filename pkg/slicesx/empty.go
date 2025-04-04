package slicesx

// EmptyAsNil returns nil if the slice is empty otherwise the slice is returned.
// This is useful when in tests we are checking struct equality, as we don't need to care
// about the slice being zero length or nil.
func EmptyAsNil[T any](s []T) []T {
	if len(s) == 0 {
		return nil
	}
	return s
}
