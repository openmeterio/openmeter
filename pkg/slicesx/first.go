package slicesx

// returns the first element in the slice where the predicate returns true
// if second argument is true the returns the last not the first
func First[T any](s []T, f func(T) bool, last bool) (*T, bool) {
	if s == nil {
		return nil, false
	}

	if last {
		for i := len(s) - 1; i >= 0; i-- {
			if f(s[i]) {
				return &s[i], true
			}
		}
		return nil, false
	}

	for _, v := range s {
		if f(v) {
			return &v, true
		}
	}

	return nil, false
}
