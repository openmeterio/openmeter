package slicesx

func AsFilterIteratee[T any](f func(T) bool) func(T, int) bool {
	return func(v T, _ int) bool {
		return f(v)
	}
}
