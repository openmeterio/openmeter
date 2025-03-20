package defaultx

func WithDefault[T any](value *T, def T) T {
	if value != nil {
		return *value
	}

	return def
}

func IfZero[T comparable](val T, def T) T {
	var zero T
	if val == zero {
		return def
	}

	return val
}

func PtrEqual[T any](a, b *T, compareFn func(T, T) bool) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return compareFn(*a, *b)
}
