package slicesx

func Dedupe[T comparable](s []T) []T {
	// Nil input, return early.
	if s == nil {
		return nil
	}

	n := make([]T, 0, len(s))
	m := make(map[T]struct{})

	for _, v := range s {
		if _, ok := m[v]; !ok {
			m[v] = struct{}{}
			n = append(n, v)
		}
	}

	return n
}
