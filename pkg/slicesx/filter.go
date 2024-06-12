package slicesx

func Filter[T any](s []T, f func(T) bool) []T {
	// Nil input, return early.
	if s == nil {
		return nil
	}

	n := make([]T, 0, len(s))

	for _, v := range s {
		if f(v) {
			n = append(n, v)
		}
	}

	return n
}
