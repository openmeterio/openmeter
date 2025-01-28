package slicesx

// Returns the last element in the slice where the predicate returns true
func Last[T any](s []T, f func(T) bool) (*T, int, bool) {
	if s == nil {
		return nil, -1, false
	}

	for i := len(s) - 1; i >= 0; i-- {
		if f(s[i]) {
			return &s[i], i, true
		}
	}
	return nil, -1, false
}
