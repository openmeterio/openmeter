package slicesx

// Map maps elements of a slice from T to M, returning a new slice.
func Map[T any, S any](s []T, f func(T) S) []S {
	// Nil input, return early.
	if s == nil {
		return nil
	}

	n := make([]S, len(s))

	for i, v := range s {
		n[i] = f(v)
	}

	return n
}

func FromMap[T comparable, S any](m map[T]S) []S {
	n := make([]S, 0, len(m))

	for _, v := range m {
		n = append(n, v)
	}

	return n
}
